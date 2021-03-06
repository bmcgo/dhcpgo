package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"net"
	"path"
	"time"

	"github.com/bmcgo/dhcpgo/dhcp"
)

type EtcdClientConfig struct {
	endpoints  []string
	caCertPath string
	certPath   string
	keyPath    string
	prefix     string
}

type EtcdClient struct {
	client             *clientv3.Client
	leases             map[string]dhcp.Lease
	prefix             string
	prefixConfigSubnet string
	prefixConfigListen string
	prefixLeases       string
}

func NewEtcdClient(ctx context.Context, c *EtcdClientConfig, timeout time.Duration) (*EtcdClient, error) {
	prefix := path.Join("/", c.prefix, "v1")
	client := &EtcdClient{
		leases:             make(map[string]dhcp.Lease),
		prefix:             prefix,
		prefixConfigSubnet: path.Join(prefix, "subnet"),
		prefixConfigListen: path.Join(prefix, "listen"),
	}
	tlsInfo := transport.TLSInfo{
		CertFile:      c.certPath,
		KeyFile:       c.keyPath,
		TrustedCAFile: c.caCertPath,
	}
	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, err
	}
	cfg := clientv3.Config{
		Endpoints:   c.endpoints,
		TLS:         tlsConfig,
		DialTimeout: 5 * time.Second,
	}

	client.client, err = clientv3.New(cfg)
	if err != nil {
		return client, err
	}

	ct, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	return client, client.client.Sync(ct)
}

func (c *EtcdClient) processListens(ctx context.Context, handler func(*dhcp.Listen) error) error {
	resp, err := c.client.Get(ctx, c.prefixConfigListen, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to list config prefix: %s", err)
	}
	for _, kv := range resp.Kvs {
		l := &dhcp.Listen{}
		err = json.Unmarshal(kv.Value, l)
		if err != nil {
			log.Printf("failed to unmarshal listener %q", kv.Key)
		} else {
			err = handler(l)
			if err != nil {
				log.Printf("error handling listener %q, %s", kv.Key, err)
			}
		}
	}
	return nil
}

func (c *EtcdClient) processSubnets(ctx context.Context, handler func(*dhcp.Subnet) error) error {
	resp, err := c.client.Get(ctx, c.prefixConfigSubnet, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to list config prefix: %s", err)
	}
	for _, kv := range resp.Kvs {
		s := &dhcp.Subnet{}
		err = json.Unmarshal(kv.Value, s)
		if err != nil {
			log.Printf("failed to unmarshal subnet %q", kv.Key)
		} else {
			s, err = dhcp.InitializeSubnet(s)
			if err != nil {
				return err
			}
			err = handler(s)
			if err != nil {
				log.Printf("error handling subnet %q, %s", kv.Key, err)
			}
		}
	}
	return nil
}

func (c *EtcdClient) WatchConfig(ctx context.Context, server *dhcp.Server) {
	var err error
	log.Printf("Watching config with prefix: %s", c.prefix)
	err = c.processListens(ctx, server.HandleListen)
	if err != nil {
		log.Println(err)
	}
	err = c.processSubnets(ctx, server.HandleSubnet)
	if err != nil {
		log.Println(err)
	}
	ch := c.client.Watch(ctx, c.prefixConfigSubnet, clientv3.WithPrefix())
	for {
		resp, ok := <-ch
		for _, ev := range resp.Events {
			log.Println(ev)
			//TODO: update config
		}
		if !ok {
			log.Println("Config watcher stopped")
			return
		}
	}
}

func (c *EtcdClient) GetLease(mac net.HardwareAddr) *dhcp.Lease {
	lease, ok := c.leases[mac.String()]
	if ok {
		return &lease
	}
	return nil
}

func (c *EtcdClient) GetFreeIP() {

}

func (c *EtcdClient) UpdateLease(mac net.HardwareAddr, lease dhcp.Lease) error {
	c.leases[mac.String()] = lease
	return nil
}

func (c *EtcdClient) PutListen(ctx context.Context, l dhcp.Listen) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	p := path.Join(c.prefixConfigListen, l.Subnet)
	resp, err := c.client.Put(ctx, p, string(data))
	if err != nil {
		log.Printf("failed to put listen:%s : %v", err, resp)
	}
	log.Println(resp)
	return err
}

func (c *EtcdClient) PutSubnet(ctx context.Context, sn dhcp.Subnet) error {
	data, err := json.Marshal(sn)
	if err != nil {
		return err
	}
	p := path.Join(c.prefixConfigSubnet, sn.Subnet)
	resp, err := c.client.Put(ctx, p, string(data))
	if err != nil {
		log.Printf("failed to put subnet:%s : %v", err, resp)
	}
	return err
}
