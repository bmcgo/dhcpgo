package dhcp

import (
	"bytes"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/insomniacslk/dhcp/iana"
	"log"
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

type ResponderSendUnicastCall struct {
	resp *dhcpv4.DHCPv4
	peer net.Addr
}

type FakeResponder struct {
	callsUnicast   []ResponderSendUnicastCall
	callsBroadcast []dhcpv4.DHCPv4
}

func NewFakeResponder() *FakeResponder {
	return &FakeResponder{
		callsUnicast:   make([]ResponderSendUnicastCall, 0),
		callsBroadcast: make([]dhcpv4.DHCPv4, 0),
	}
}

func (f *FakeResponder) SendUnicast(resp *dhcpv4.DHCPv4, peer net.Addr) error {
	f.callsUnicast = append(f.callsUnicast, ResponderSendUnicastCall{
		resp: resp,
		peer: peer,
	})
	return nil
}

func (f *FakeResponder) SendBroadcast(resp *dhcpv4.DHCPv4) error {
	f.callsBroadcast = append(f.callsBroadcast, *resp)
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
		Subnet:    "192.168.10.0/24",
		RangeFrom: "192.168.10.100",
		RangeTo:   "192.168.10.200",
		Gateway:   "192.168.10.1",
		DNS:       []string{"1.1.1.1", "2.2.2.2"},
		LeaseTime: 3600,
		Options:   nil,
	}
	_, err = InitializeSubnet(sn)
	assertNoError(t, err)
	err = s.HandleSubnet(sn)
	assertNoError(t, err)

	req := &dhcpv4.DHCPv4{
		OpCode:        dhcpv4.OpcodeBootRequest,
		HWType:        iana.HWTypeEthernet,
		TransactionID: dhcpv4.TransactionID{1, 2, 3, 4},
		GatewayIPAddr: net.ParseIP("192.168.10.1"),
		ClientHWAddr:  net.HardwareAddr{1, 2, 3, 4, 5, 6},
	}
	req.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover))
	fs.handler(&FakePacketConn{}, &FakeNetAddr{}, req)
	assertTrue(t, len(responder.callsUnicast) == 1)
	resp := responder.callsUnicast[0].resp
	assertEqual(t, dhcpv4.MessageTypeOffer, resp.MessageType())
	assertEqual(t, "10.1.1.1", resp.ServerIPAddr.String())
	assertTrue(t, 0 == bytes.Compare([]byte{0, 0, 14, 16}, resp.Options.Get(dhcpv4.OptionIPAddressLeaseTime)))
	assertEqual(t, "192.168.10.100", resp.YourIPAddr.String())
	log.Println(resp)
}
