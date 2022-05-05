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
type FakeResponderFactory struct {
	responder *FakeResponder
}

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

func NewFakeResponder() *FakeResponder {
	return &FakeResponder{calls: make([]ResponderSendCall, 0)}
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
	return f.responder, nil
}

func (f *FakeDHCPServer) Serve() error {
	time.Sleep(time.Hour)
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
	responder := NewFakeResponder()
	s := NewServer(ServerConfig{
		DHCPv4ServerFactory: &FakeDHCPServerFactory{fakeDHCPServer: fs},
		ResponderFactory:    &FakeResponderFactory{responder: responder},
	})
	err := s.HandleListen(&Listen{
		Interface: "eth0",
		Subnet:    "10.1.1.0/24",
		Laddr:     "10.1.1.1",
	})
	assertNoError(t, err)
	assertTrue(t, fs.handler != nil)
	sn := &Subnet{
		Subnet:     "192.168.10.0/24",
		RangeFrom:  "192.168.10.100",
		RangeTo:    "192.168.10.200",
		Gateway:    "192.168.10.1",
		DNS:        []string{"1.1.1.1", "2.2.2.2"},
		Options:    nil,
	}
	_, err = InitializeSubnet(sn)
	assertNoError(t, err)
	err = s.HandleSubnet(sn)
	assertNoError(t, err)

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
		GatewayIPAddr:  net.ParseIP("192.168.10.1"),
		ClientHWAddr:   net.HardwareAddr{1, 2, 3, 4, 5, 6},
		ServerHostName: "",
		BootFileName:   "",
		Options:        nil,
	}
	req.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover))
	fs.handler(&FakePacketConn{}, &FakeNetAddr{}, req)
	assertTrue(t, len(responder.calls) == 1)
	resp := responder.calls[0].resp
	assertEqual(t, dhcpv4.MessageTypeOffer, resp.MessageType())
}
