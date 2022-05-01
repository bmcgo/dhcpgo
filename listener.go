package main

import (
	"errors"
	"fmt"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"log"
	"net"
	"strings"
)

type ResponseGetter func(*dhcpv4.DHCPv4, *Listen) (*dhcpv4.DHCPv4, error)

type Listener struct {
	server         *server4.Server
	responseGetter ResponseGetter
	responder      Responder
	listen         *Listen
	serverIPAddr   net.IP
}

func (l Listener) String() string {
	return fmt.Sprintf("[Listener [if:%q subnet:%q laddr:%q]]", l.listen.Interface, l.listen.Subnet, l.listen.Laddr)
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

	s, err := net.Dial("udp", strings.Split(listen.Subnet, "/")[0] + ":67")
	if err != nil {
		return nil, err
	}
	defer s.Close()
	listener.serverIPAddr = net.ParseIP(strings.Split(s.LocalAddr().String(), ":")[0])
	return listener, err
}

func (l *Listener) Handler(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4) {
	var (
		err  error
		resp *dhcpv4.DHCPv4
	)
	log.Printf("%s <- %s (%s) %s [%s]", conn.LocalAddr().String(), req.ClientHWAddr, peer.String(), req.MessageType(), conn.LocalAddr().Network())
	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover:
		resp, err = l.handleDiscover(req)
	case dhcpv4.MessageTypeRequest:
		resp, err = l.handleRequest(req)
	}
	if err != nil {
		log.Println(err)
		return
	}
	err = l.responder.Send(resp)
	if err != nil {
		log.Printf("failed to send dhcp response: %s", err)
	}
}

func (l *Listener) handleDiscover(req *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, error) {
	resp, err := l.responseGetter(req, l.listen)
	if err != nil {
		return resp, err
	}
	if resp == nil {
		return resp, errors.New("nil response")
	}
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	resp.ServerIPAddr = l.serverIPAddr
	return resp, err
}

func (l *Listener) handleRequest(req *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, error) {
	resp, err := l.responseGetter(req, l.listen)
	if err != nil {
		return resp, err
	}
	if resp == nil {
		return resp, errors.New("nil response")
	}
	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	resp.ServerIPAddr = l.serverIPAddr
	return resp, err
}

func (l *Listener) Serve() error {
	log.Printf("starting server %v", l)
	defer l.responder.Close()
	return l.server.Serve()
}
