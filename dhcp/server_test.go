package dhcp

import (
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/insomniacslk/dhcp/iana"
	"net"
	"testing"
	"time"
)

type FakeDHCPServer struct {
	handler server4.Handler
}
type FakeDHCPServerFactory struct {
	fakeDHCPServer *FakeDHCPServer
}
type FakeResponderFactory struct{}

func (f *FakeDHCPServerFactory) NewServer(listenInterface string, listenAddress string, handler server4.Handler) (DHCPv4Server, error) {
	f.fakeDHCPServer.handler = handler
	return f.fakeDHCPServer, nil
}

type ResponderSendCall struct {
	req  *dhcpv4.DHCPv4
	resp *dhcpv4.DHCPv4
	peer net.Addr
}

type FakeResponder struct {
	calls []ResponderSendCall
}

func (f *FakeResponder) Send(resp *dhcpv4.DHCPv4, req *dhcpv4.DHCPv4, peer net.Addr) error {
	f.calls = append(f.calls, ResponderSendCall{
		req:  req,
		resp: resp,
		peer: peer,
	})
	return nil
}

func (f *FakeResponder) Close() {}

func (f *FakeResponderFactory) NewResponder(listen *Listen) (Responder, error) {
	return &FakeResponder{calls: make([]ResponderSendCall, 0)}, nil
}

func (f *FakeDHCPServer) Serve() error {
	return nil
}

func (f *FakeDHCPServer) Close() error {
	return nil
}

type FakeNetAddr struct{}

func (f *FakeNetAddr) String() string {
	return "fake-net-addr-string"
}

func (f *FakeNetAddr) Network() string {
	return "fake-net-addr-network"
}

type FakePacketConn struct{}

func (f *FakePacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	return 1, &FakeNetAddr{}, nil
}

func (f *FakePacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return 1, nil
}

func (f *FakePacketConn) LocalAddr() net.Addr {
	return &FakeNetAddr{}
}

func (f *FakePacketConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (f *FakePacketConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (f *FakePacketConn) SetDeadline(t time.Time) error {
	return nil
}

func (f *FakePacketConn) Close() error {
	return nil
}

func TestNewServer(t *testing.T) {
	fs := &FakeDHCPServer{}
	s := NewServer(ServerConfig{
		DHCPv4ServerFactory: &FakeDHCPServerFactory{fakeDHCPServer: fs},
		ResponderFactory:    &FakeResponderFactory{},
	})
	err := s.HandleListen(&Listen{
		Interface: "eth0",
		Subnet:    "10.1.1.0/24",
		Laddr:     "10.1.1.1",
	})
	assertNoError(t, err)
	defer s.Close()
	assertTrue(t, fs.handler != nil)
	req := &dhcpv4.DHCPv4{
		OpCode:         dhcpv4.OpcodeBootRequest,
		HWType:         iana.HWTypeEthernet,
		HopCount:       0,
		TransactionID:  dhcpv4.TransactionID{},
		NumSeconds:     0,
		Flags:          0,
		ClientIPAddr:   nil,
		YourIPAddr:     nil,
		ServerIPAddr:   nil,
		GatewayIPAddr:  nil,
		ClientHWAddr:   net.HardwareAddr{1, 2, 3, 4, 5, 6},
		ServerHostName: "",
		BootFileName:   "",
		Options:        nil,
	}
	req.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover))
	fs.handler(&FakePacketConn{}, &FakeNetAddr{}, req)
}
