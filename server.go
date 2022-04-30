package main

import (
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"log"
	"net"
	"time"
)

type Server struct {
	subnet    Subnet
	etcd      *EtcdClient
	server    *server4.Server
	responder *Responder
	ipv4Range *Range
	laddr     string
}

func NewServer(ifname string, laddr string, responder *Responder, etcd *EtcdClient, r *Range, subnet Subnet) (*Server, error) {
	var err error
	addr := &net.UDPAddr{
		IP:   net.ParseIP(laddr),
		Port: dhcpv4.ServerPort,
	}
	server := &Server{
		responder: responder,
		etcd:      etcd,
		ipv4Range: r,
		subnet:    subnet,
		laddr:     laddr,
	}
	server.server, err = server4.NewServer(ifname, addr, server.Handler)
	return server, err
}

func (s *Server) Handler(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4) {
	log.Printf("<-%s %s", req.ClientHWAddr, req.MessageType())
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

func updateResp(lease *Lease, resp *dhcpv4.DHCPv4, subnet Subnet) {
	log.Printf("got lease %v", lease)
	resp.YourIPAddr = net.ParseIP(lease.IP)
	resp.GatewayIPAddr = net.ParseIP(subnet.Gateway)
	resp.ServerIPAddr = net.ParseIP(subnet.Laddr)

	for _, opt := range subnet.Options {
		var value dhcpv4.OptionValue
		code := opt.ID
		switch opt.Type {
		case "string":
			value = dhcpv4.String(opt.Value)
		default:
			log.Printf("invalid option value type in subnet: %v", subnet)
		}
		resp.UpdateOption(dhcpv4.Option{Code: dhcpv4.GenericOptionCode(code), Value: value})
	}

	resp.UpdateOption(dhcpv4.OptSubnetMask(net.IPv4Mask(255, 255, 255, 0))) //TODO
	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(time.Hour * 8))          //TODO

}

func (s *Server) handleDiscover(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	lease := s.ipv4Range.GetLeaseForMAC(req.ClientHWAddr.String())
	if lease == nil {
		//TODO: send NAK
		return nil
	}
	updateResp(lease, resp, s.subnet)
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	return nil
}

func (s *Server) handleRequest(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) error {
	lease := s.ipv4Range.GetLeaseForMAC(req.ClientHWAddr.String())
	if lease == nil {
		//TODO: send NAK
		return nil
	}
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	updateResp(lease, resp, s.subnet)
	return nil
}

func (s *Server) Serve() error {
	log.Printf("starting server %v", s)
	return s.server.Serve()
}
