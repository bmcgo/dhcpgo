package main

import (
	"context"
	"log"
)

type ServerManager struct {
	etcd    *EtcdClient
	servers []*Server
}

type Subnet struct {
	Key string
	Interface   string `json:"interface"`
	AddressMask string `json:"addressMask"`
	IPFrom      string `json:"ipFrom"`
	IPTo        string `json:"ipTo"`
}

type ConfigWatchHandler interface {
	HandleSubnet(Subnet) error
}

func NewServerManager(client *EtcdClient) *ServerManager {
	return &ServerManager{
		etcd:    client,
		servers: make([]*Server, 1),
	}
}

func (s *ServerManager) HandleSubnet(subnet Subnet) error {
	log.Println(subnet)
	return nil
}

func (s *ServerManager) Run(ctx context.Context) error {
	s.etcd.WatchConfig(ctx, s)
	return nil
}
