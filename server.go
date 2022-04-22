package main

import (
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"log"
	"net"
	"time"
)

type Server struct {
	etcd      *EtcdClient
	server    *server4.Server
	responder *Responder
}

func NewServer(ifname string, laddr string, responder *Responder, etcd *EtcdClient) (*Server, error) {
	var err error
	addr := &net.UDPAddr{
		IP:   net.ParseIP(laddr),
		Port: dhcpv4.ServerPort,
	}
	server := &Server{
		responder: responder,
		etcd:      etcd,
	}
	server.server, err = server4.NewServer(ifname, addr, server.Handler)
	return server, err
}

func (s *Server) Handler(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4) {
	log.Printf("%s, %s, %s", conn, peer, req)
	resp, err := dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		log.Println(err)
		return
	}

	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover:
		err = s.handleDiscover(req, resp)
	case dhcpv4.MessageTypeRequest:
		err = s.handleRequest(req, resp)
	}
	if err != nil {
		log.Println(err)
		return
	}

	err = s.responder.Send(resp)
	if err != nil {
		log.Println(err)
		return
	}
}

func (s *Server) handleDiscover(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	resp.YourIPAddr = net.ParseIP("10.12.1.34")
	resp.GatewayIPAddr = net.ParseIP("10.12.1.1")
	resp.ServerIPAddr = net.ParseIP("10.12.1.1")
	resp.UpdateOption(dhcpv4.OptSubnetMask(net.IPv4Mask(255, 255, 255, 0)))
	//resp.UpdateOption(dhcpv4.Option{Code: dhcpv4.GenericOptionCode(66), Value: dhcpv4.String("10.12.1.2")}) //not working
	resp.UpdateOption(dhcpv4.Option{Code: dhcpv4.GenericOptionCode(67), Value: dhcpv4.String("pxe/boot.ok")})
	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(time.Hour * 8))
	return nil
}

func (s *Server) handleRequest(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	resp.ServerIPAddr = net.ParseIP("10.12.1.1")
	resp.YourIPAddr = net.ParseIP("10.12.1.34")
	resp.GatewayIPAddr = net.ParseIP("10.12.1.1")
	resp.UpdateOption(dhcpv4.OptSubnetMask(net.IPv4Mask(255, 255, 255, 0)))
	resp.UpdateOption(dhcpv4.Option{Code: dhcpv4.GenericOptionCode(66), Value: dhcpv4.String("10.12.1.2")})
	resp.UpdateOption(dhcpv4.Option{Code: dhcpv4.GenericOptionCode(67), Value: dhcpv4.String("pxe/boot.ok")})
	return nil
}

func (s *Server) Serve() error {
	return s.server.Serve()
}
