package main

import (
	"errors"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"log"
	"net"
)

type ResponseGetter func(*dhcpv4.DHCPv4, *Listen) (*dhcpv4.DHCPv4, error)

type Listener struct {
	server         *server4.Server
	responseGetter ResponseGetter
	responder      Responder
	listen         *Listen
}

func NewListener(listen *Listen, handler ResponseGetter) (*Listener, error) {
	var err error
	addr := &net.UDPAddr{
		IP:   net.ParseIP(listen.Laddr),
		Port: dhcpv4.ServerPort,
	}
	//TODO: unicast responder
	responder, err := NewBroadcastResponder(listen.Interface)
	if err != nil {
		return nil, err
	}
	listener := &Listener{responseGetter: handler, responder: responder, listen: listen}
	listener.server, err = server4.NewServer(listen.Interface, addr, listener.Handler)
	log.Printf("new listener at %s (%s)", addr, listen.Interface)
	return listener, err
}

func (s *Listener) Handler(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4) {
	var (
		err  error
		resp *dhcpv4.DHCPv4
	)
	log.Printf("%s<-%s(%s) %s [%s]", conn.LocalAddr().String(), req.ClientHWAddr, peer.String(), req.MessageType(), conn.LocalAddr().Network())
	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover:
		resp, err = s.handleDiscover(req)
	case dhcpv4.MessageTypeRequest:
		resp, err = s.handleRequest(req)
	}
	if err != nil {
		log.Println(err)
		return
	}
	err = s.responder.Send(resp)
	if err != nil {
		log.Printf("failed to send dhcp response: %s", err)
	}
}

func (s *Listener) handleDiscover(req *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, error) {
	resp, err := s.responseGetter(req, s.listen)
	if err != nil {
		return resp, err
	}
	if resp == nil {
		return resp, errors.New("nil response")
	}
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	return resp, err
}

func (s *Listener) handleRequest(req *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, error) {
	resp, err := s.responseGetter(req, s.listen)
	if err != nil {
		return resp, err
	}
	if resp == nil {
		return resp, errors.New("nil response")
	}
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	return resp, err
}

func (s *Listener) Serve() error {
	log.Printf("starting server %v", s)
	defer s.responder.Close()
	return s.server.Serve()
}
