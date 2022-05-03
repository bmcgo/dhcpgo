package dhcp

import (
	"context"
	"errors"
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

type Storage interface {
	WatchConfig(context.Context, ConfigWatchHandler)
}

type Server struct {
	storage   Storage
	listeners []*Listener
	responders []*Responder
	subnets    map[string]*Subnet
}

type ConfigWatchHandler interface {
	HandleListen(*Listen) error
	HandleSubnet(*Subnet) error
	//HandleHost(Host) error //TODO
}

func NewServer(storage Storage) *Server {
	return &Server{
		storage:   storage,
		listeners: make([]*Listener, 0),
		subnets:   make(map[string]*Subnet),
	}
}

func (s *Server) GetLease(req *dhcpv4.DHCPv4, listen *Listen) (*dhcpv4.DHCPv4, error) {
	var lease *Lease
	log.Println(req, listen)
	subnet, ok := s.subnets[listen.Subnet]
	if ok {
		lease = subnet.GetLeaseForMAC(req.ClientHWAddr.String())
		if lease == nil {
			return nil, errors.New("empty lease")
		}
	} else {
		//TODO: find subnet
		return nil, errors.New("not implemented")
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
	return resp, nil
}

func (s *Server) HandleListen(listen *Listen) error {
	listener, err := NewListener(listen, s.GetLease)
	if err != nil {
		return err
	}
	s.listeners = append(s.listeners, listener)
	go func() {
		err = listener.Serve()
		log.Printf("exited server %v: %s", s, err)
	}()
	return nil
}

func (s *Server) HandleSubnet(subnet *Subnet) error {
	var err error
	s.subnets[subnet.Subnet] = subnet
	log.Printf("Serving subnet %v", subnet)
	//TODO: load range cache
	return err
}

func (s *Server) Run(ctx context.Context) error {
	s.storage.WatchConfig(ctx, s)
	return nil
}
