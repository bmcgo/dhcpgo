package main

import (
	"context"
	"log"
	"time"
)

type ServerManager struct {
	etcd    *EtcdClient
	servers map[string]*Server
}

type ConfigWatchHandler interface {
	HandleSubnet(Subnet) error
}

func NewServerManager(client *EtcdClient) *ServerManager {
	return &ServerManager{
		etcd:    client,
		servers: make(map[string]*Server),
	}
}

func (s *ServerManager) HandleSubnet(subnet Subnet) error {
	responder, err := NewResponder(subnet.Interface)
	if err != nil {
		return err
	}
	//TODO: default lease time
	r, err := NewRange(subnet.RangeFrom, subnet.RangeTo, 4 * time.Hour)
	if err != nil {
		return err
	}
	server, err := NewServer(subnet.Interface, "0.0.0.0", responder, s.etcd, r, subnet)
	if err != nil {
		return err
	}
	s.servers[subnet.AddressMask] = server
	go func() {
		log.Printf("starting server for subnet %s", subnet.AddressMask)
		err = server.Serve()
		log.Printf("exited server %v: %s", s, err)
	}()
	return nil
}

func (s *ServerManager) Run(ctx context.Context) error {
	s.etcd.WatchConfig(ctx, s)
	return nil
}
