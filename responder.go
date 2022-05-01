package main

import (
	"fmt"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"log"
	"net"
	"syscall"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type Responder interface {
	Send(resp *dhcpv4.DHCPv4) error
	Close()
}

type BroadcastResponder struct {
	fd    int
	layer syscall.SockaddrLinklayer
	eth   layers.Ethernet
	ip    layers.IPv4
	udp   layers.UDP
	buf   gopacket.SerializeBuffer
	opts  gopacket.SerializeOptions
}

func NewBroadcastResponder(ifaceName string) (*BroadcastResponder, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, err
	}
	log.Printf("new responder on %s %v", ifaceName, iface)
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot open socket: %v", err)
	}

	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return nil, fmt.Errorf("cannot set option for socket: %v", err)
	}
	responder := &BroadcastResponder{
		fd: fd,
		layer: syscall.SockaddrLinklayer{
			Protocol: 0,
			Ifindex:  iface.Index,
			Halen:    6,
			//Addr:     hwAddr, //not used
		},
		eth: layers.Ethernet{
			EthernetType: layers.EthernetTypeIPv4,
			SrcMAC:       iface.HardwareAddr,
		},
		ip: layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolUDP,
			Flags:    layers.IPv4DontFragment,
		},
		udp: layers.UDP{
			SrcPort: dhcpv4.ServerPort,
			DstPort: dhcpv4.ClientPort,
		},
		opts: gopacket.SerializeOptions{
			ComputeChecksums: true,
			FixLengths:       true,
		},
		buf: gopacket.NewSerializeBuffer(),
	}
	err = responder.udp.SetNetworkLayerForChecksum(&responder.ip)
	if err != nil {
		return nil, fmt.Errorf("couldn't set network layer: %v", err)
	}
	return responder, nil
}

func (r *BroadcastResponder) Send(resp *dhcpv4.DHCPv4) error {
	r.eth.DstMAC = resp.ClientHWAddr
	r.ip.SrcIP = resp.ServerIPAddr
	r.ip.DstIP = resp.YourIPAddr

	packet := gopacket.NewPacket(resp.ToBytes(), layers.LayerTypeDHCPv4, gopacket.NoCopy)
	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	dhcp, ok := dhcpLayer.(gopacket.SerializableLayer)
	if !ok {
		return fmt.Errorf("layer %q is not serializable", dhcpLayer.LayerType().String())
	}
	err := gopacket.SerializeLayers(r.buf, r.opts, &r.eth, &r.ip, &r.udp, dhcp)
	if err != nil {
		return fmt.Errorf("cannot serialize layer: %v", err)
	}
	data := r.buf.Bytes()

	var hwAddr [8]byte
	copy(hwAddr[0:6], resp.ClientHWAddr[0:6])

	log.Printf("->%s %s %s %v", resp.ClientHWAddr, resp.MessageType(), resp.YourIPAddr, resp.Options)
	return syscall.Sendto(r.fd, data, 0, &r.layer)
}

func (r *BroadcastResponder) Close() {
	err := syscall.Close(r.fd)
	if err != nil {
		log.Printf("error closing socket: %s", err)
	}
}
