package main

import (
	"context"
	"fmt"
	"log"
	"strings"
)

type DhcpgoClient interface {
	PutSubnet(context.Context, Subnet) error
}

type DhcpgoTool struct {
	ctx    context.Context
	client DhcpgoClient
}

func NewDhcpgoTool(ctx context.Context, client *EtcdClient) *DhcpgoTool {
	return &DhcpgoTool{
		client: client,
		ctx:    ctx,
	}
}

func (c *DhcpgoTool) Configure(args []string) error {
	switch args[0] {
	case "subnet":
		return c.configureSubnet(args[1:])
	case "host":
		return c.configureHost(args[1:])
	}
	log.Println(args)
	return nil
}

func (c *DhcpgoTool) configureSubnet(args []string) error {
	// 10.1.1.0/24 10.1.1.10-10.1.1.99 if=eth0,gw=10.1.1.1,dns=10.1.1.1,dns=10.2.1.1,option-67=string:boot.pxe
	if len(args) != 3 {
		//TODO: print usage
		return fmt.Errorf("invalid args %v", args)
	}
	subnet := Subnet{
		DNS:         make([]string, 0),
		Options:     make([]Option, 0),
	}

	//TODO: validate address/mask
	subnet.AddressMask = args[0]

	ipRange := strings.Split(args[1], "-")
	if len(ipRange) != 2 {
		return fmt.Errorf("invalid range: %q", args[1])
	}
	//TODO: validate range from>to
	//TODO: validate range within subnet
	subnet.RangeFrom = ipRange[0]
	subnet.RangeTo = ipRange[1]

	for _, bit := range strings.Split(args[2], ",") {
		nameVal := strings.Split(bit, "=")
		switch nameVal[0] {
		case "if":
			subnet.Interface = nameVal[1]
		case "gw":
			subnet.Gateway = nameVal[1]
		case "dns":
			subnet.DNS = append(subnet.DNS, nameVal[1])
		default:
			if strings.HasPrefix(nameVal[0], "option-") {
				//TODO: parse options
				continue
			}
		}
	}

	return c.client.PutSubnet(c.ctx, subnet)
}

func (c *DhcpgoTool) configureHost(args []string) error {
	// 00:01:02:03:04:05 ipv4=192.168.1.101,option-67=string:boot-101.pxe
	return nil
}
