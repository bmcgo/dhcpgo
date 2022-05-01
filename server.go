package main

import (
	"context"
	"errors"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"log"
	"net"
	"time"
)

type Server struct {
	etcd       *EtcdClient
	listeners  []*Listener
	responders []*BroadcastResponder
	subnets    map[string]*Subnet
}

type ConfigWatchHandler interface {
	HandleListen(*Listen) error
	HandleSubnet(*Subnet) error
	//HandleHost(Host) error //TODO
}

func NewServer(client *EtcdClient) *Server {
	return &Server{
		etcd:      client,
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

	resp.YourIPAddr = net.ParseIP(lease.IP)
	resp.GatewayIPAddr = net.ParseIP(lease.Gateway)
	resp.ServerIPAddr = net.ParseIP("0.0.0.0") //TODO
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
	resp.UpdateOption(dhcpv4.OptSubnetMask(net.IPv4Mask(255, 255, 255, 0))) //TODO
	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(time.Hour * 8))          //TODO
	return resp, nil
}

func (s *Server) HandleListen(listen *Listen) error {
	listener, err := NewListener(listen, s.GetLease)
	if err != nil {
		return err
	}
	s.listeners = append(s.listeners, listener)
	go func() {
		log.Printf("listening %s", listen.Laddr)
		err = listener.Serve()
		log.Printf("exited server %v: %s", s, err)
	}()
	return nil
}

func (s *Server) HandleSubnet(subnet *Subnet) error {
	var err error
	s.subnets[subnet.AddressMask] = subnet
	//TODO: load range cache
	return err
}

func (s *Server) Run(ctx context.Context) error {
	s.etcd.WatchConfig(ctx, s)
	return nil
}
