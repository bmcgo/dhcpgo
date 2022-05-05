package dhcp

import (
	"errors"
	"fmt"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"log"
	"net"
)

type ResponseGetter func(*dhcpv4.DHCPv4, *Listen) (*dhcpv4.DHCPv4, error)

type Listener struct {
	server         DHCPv4Server
	responseGetter ResponseGetter
	responder      Responder
	listen         *Listen
	serverIPAddr   net.IP
}

type DHCPv4Server interface {
	Serve() error
	Close() error
}

type DHCPv4ServerFactory interface {
	NewServer(listenInterface string, listenAddress string, handler server4.Handler) (DHCPv4Server, error)
}

type DefaultDHCPServerFactory struct{}

func (f *DefaultDHCPServerFactory) NewServer(listenInterface string, listenAddress string, handler server4.Handler) (DHCPv4Server, error) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(listenAddress),
		Port: dhcpv4.ServerPort,
	}
	return server4.NewServer(listenInterface, addr, handler)
}

func (l Listener) String() string {
	return fmt.Sprintf("[Listener [if:%q subnet:%q laddr:%q]]", l.listen.Interface, l.listen.Subnet, l.listen.Laddr)
}

func NewListener(listen *Listen, handler ResponseGetter, serverFactory DHCPv4ServerFactory, responderFactory ResponderFactory) (*Listener, error) {
		responder, err := responderFactory.NewResponder(listen)
	if err != nil {
		return nil, err
	}
	listener := &Listener{responseGetter: handler, responder: responder, listen: listen}
	listener.server, err = serverFactory.NewServer(listen.Interface, listen.Laddr, listener.Handler)
	if err != nil {
		return nil, err
	}
	listener.serverIPAddr = net.ParseIP(listen.Laddr).To4()
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
	default:
		log.Printf("unknown dhcp packet type %s", req.MessageType())
		return
	}
	if err != nil {
		log.Println(err)
		return
	}
	err = l.responder.Send(resp, req, peer)
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
	defer l.responder.Close()
	return l.server.Serve()
}
