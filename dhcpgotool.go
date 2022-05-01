package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type DhcpgoClient interface {
	PutListen(context.Context, Listen) error
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
	case "listen":
		return c.configureListen(args[1:])
	case "subnet":
		return c.configureSubnet(args[1:])
	case "host":
		return c.configureHost(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c *DhcpgoTool) configureListen(args []string) error {
	// if=eth0,laddr=192.168.1.1,subnet=192.168.1.0/24
	if len(args) != 1 {
		return fmt.Errorf("invalid args %v", args)
	}
	listen := Listen{}
	for _, bit := range strings.Split(args[0], ",") {
		keyVal := strings.Split(bit, "=")
		//TODO: validate all
		switch keyVal[0] {
		case "if":
			listen.Interface = keyVal[1]
		case "laddr":
			listen.Laddr = keyVal[1]
		case "subnet":
			listen.Subnet = keyVal[1]
		default:
			return fmt.Errorf("invalid args %v", args)
		}
	}
	return c.client.PutListen(c.ctx, listen)
}

func (c *DhcpgoTool) configureSubnet(args []string) error {
	// 10.1.1.0/24 10.1.1.10-10.1.1.99 if=eth0,gw=10.1.1.1,dns=10.1.1.1,dns=10.2.1.1,option-67=string:boot.pxe,option-66=string:10.12.1.1
	if len(args) != 3 {
		//TODO: print usage
		return fmt.Errorf("invalid args %v", args)
	}
	subnet := Subnet{
		DNS:     make([]string, 0),
		Options: make([]Option, 0),
	}

	//TODO: validate address/mask
	subnet.Subnet = args[0]

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
		case "gw":
			subnet.Gateway = nameVal[1]
		case "dns":
			subnet.DNS = append(subnet.DNS, nameVal[1])
		default:
			if strings.HasPrefix(nameVal[0], "option-") {
				num, err := strconv.ParseInt(nameVal[0][7:], 10, 8)
				if err != nil {
					return fmt.Errorf("invalid option %s", nameVal)
				}
				typeVal := strings.Split(nameVal[1], ":")
				if len(typeVal) != 2 {
					return fmt.Errorf("invalid option %s", nameVal)
				}
				subnet.Options = append(subnet.Options, Option{
					ID:    uint8(num),
					Type:  typeVal[0],
					Value: typeVal[1],
				})
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
