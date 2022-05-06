package dhcp

import (
	"errors"
	"fmt"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"log"
	"net"
	"time"
)

type Listen struct {
	Interface string `json:"interface,omitempty"`
	Subnet    string `json:"subnet"`
	Laddr     string `json:"laddr"`
}

type Option struct {
	ID    uint8  `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Server struct {
	listeners         []*Listener
	responders        []*Responder
	subnets           map[string]*Subnet
	dhcpServerFactory DHCPv4ServerFactory
	responderFactory  ResponderFactory
}

type ServerConfig struct {
	DHCPv4ServerFactory DHCPv4ServerFactory
	ResponderFactory    ResponderFactory
	HandleLease         func(*Lease) error
}

func GetDefaultServerConfig(leaseHandler func(*Lease) error) ServerConfig {
	return ServerConfig{
		DHCPv4ServerFactory: &DefaultDHCPServerFactory{},
		ResponderFactory:    &DefaultResponderFactory{},
		HandleLease:         leaseHandler,
	}
}

func NewServer(config ServerConfig) *Server {
	return &Server{
		listeners:         make([]*Listener, 0),
		subnets:           make(map[string]*Subnet),
		dhcpServerFactory: config.DHCPv4ServerFactory,
		responderFactory:  config.ResponderFactory,
	}
}

func (s *Server) getLease(req *dhcpv4.DHCPv4, listen *Listen) (*dhcpv4.DHCPv4, error) {
	var lease *Lease
	subnet, ok := s.subnets[listen.Subnet]
	if ok {
		lease = subnet.GetLeaseForMAC(req)
		if lease == nil {
			return nil, errors.New("empty lease")
		}
	} else {
		for _, s := range s.subnets {
			if s.Contains(req.GatewayIPAddr) {
				log.Printf("found subnet: %v", s)
				lease = s.GetLeaseForMAC(req)
				break
			} else {
				log.Printf("%s not in %s (%s)", req.GatewayIPAddr.String(), s.Subnet, s.ipNet)
			}
		}
	}

	if lease == nil {
		//TODO: send NAK
		return nil, errors.New("NAK not implemented")
	}
	log.Printf("got lease %v", lease)

	resp, err := dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		return resp, err
	}

	resp.YourIPAddr = net.ParseIP(lease.IP).To4()
	resp.GatewayIPAddr = net.ParseIP(lease.Gateway).To4()
	for _, opt := range lease.Options {
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
	resp.UpdateOption(dhcpv4.OptSubnetMask(net.IPMask(net.ParseIP(lease.NetMask).To4())))
	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(time.Duration(lease.LeaseTime) * time.Second))
	resp.UpdateOption(dhcpv4.Option{Code: dhcpv4.GenericOptionCode(3), Value: dhcpv4.IP{resp.GatewayIPAddr[0], resp.GatewayIPAddr[1], resp.GatewayIPAddr[2], resp.GatewayIPAddr[3]}})
	//resp.UpdateOption(dhcpv4.Option{Code: dhcpv4.GenericOptionCode(28), Value: dhcpv4.IP{10, 12, 1, 255}})
	dnsServers := make([]net.IP, 0)
	for _, dns := range lease.DNS {
		dnsServers = append(dnsServers, net.ParseIP(dns).To4())
	}
	resp.UpdateOption(dhcpv4.OptDNS(dnsServers...))

	//TODO: option 54 server id
	resp.UpdateOption(dhcpv4.Option{Code: dhcpv4.GenericOptionCode(54), Value: dhcpv4.IP{resp.GatewayIPAddr[0], resp.GatewayIPAddr[1], resp.GatewayIPAddr[2], resp.GatewayIPAddr[3]}})

	if resp.MessageType() == dhcpv4.MessageTypeAck {
		err = s.HandleLease(lease)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (s *Server) HandleListen(listen *Listen) error {
	listener, err := NewListener(listen, s.getLease, s.dhcpServerFactory, s.responderFactory)
	if err != nil {
		return err
	}
	s.listeners = append(s.listeners, listener)
	log.Printf("starting server %v", listener)
	go func() {
		err = listener.Serve()
		log.Printf("exited server %v: %s", s, err)
		//TODO: panic if exited unexpectedly
	}()
	return nil
}

func (s *Server) HandleSubnet(subnet *Subnet) error {
	var err error
	subnet, err = InitializeSubnet(subnet)
	if err != nil {
		return err
	}
	s.subnets[subnet.Subnet] = subnet
	log.Printf("Serving subnet %v", subnet)
	return err
}

func (s *Server) HandleLease(lease *Lease) error {
	for _, sn := range s.subnets {
		if sn.ipNet.Contains(net.ParseIP(lease.IP)) {
			sn.leaseCache[lease.MAC] = lease
			return nil
		}
	}
	return fmt.Errorf("subnet for lease not found: %v", lease)
}

func (s *Server) StopListen(subnet string) {
	for _, l := range s.listeners {
		if l.listen.Subnet == subnet {
			err := l.server.Close()
			if err != nil {
				log.Printf("Failed to stop server: %s", err)
			}
			return
		}
	}
	log.Printf("Listener for subnet %q not found", subnet)
}

func (s *Server) Close() {
	for _, l := range s.listeners {
		err := l.server.Close()
		if err != nil {
			log.Printf("failed to close listener: %s %s", l, err)
		}
	}
}
