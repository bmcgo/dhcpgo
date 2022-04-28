package main

import (
	"context"
	"encoding/json"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"net"
	"path"
	"time"
)

type EtcdClientConfig struct {
	endpoints  []string
	caCertPath string
	certPath   string
	keyPath    string
	prefix     string
}

type EtcdClient struct {
	client *clientv3.Client
	prefix string
	leases map[string]Lease
}

func NewEtcdClient(ctx context.Context, c *EtcdClientConfig, timeout time.Duration) (*EtcdClient, error) {
	client := &EtcdClient{
		leases: make(map[string]Lease),
		prefix: c.prefix,
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

func (c *EtcdClient) processSubnets(ctx context.Context, prefix string, handler ConfigWatchHandler) {
	resp, err := c.client.Get(ctx, path.Join(prefix, "subnets"), clientv3.WithPrefix())
	if err != nil {
		log.Printf("Failed to list config prefix: %s", err)
		return
	}
	for _, kv := range resp.Kvs {
		s := Subnet{}
		err = json.Unmarshal(kv.Value, &s)
		if err != nil {
			log.Printf("failed to unmarshal subnet %q", kv.Key)
		} else {
			s.Key = string(kv.Key)
			err = handler.HandleSubnet(s)
			if err != nil {
				log.Printf("error handling subnet %q, %s", kv.Key, err)
			}
		}
	}
}

func (c *EtcdClient) WatchConfig(ctx context.Context, handler ConfigWatchHandler) {
	prefix := path.Join("/", c.prefix, "v1", "config")
	log.Printf("Watching config with prefix: %s", prefix)

	c.processSubnets(ctx, prefix, handler)

	ch := c.client.Watch(ctx, prefix, clientv3.WithPrefix())
	for {
		resp, ok := <- ch
		for _, ev := range resp.Events {
			log.Println(ev)
		}
		if !ok {
			log.Println("Config watcher stopped")
			return
		}
	}
}

func (c *EtcdClient) GetLease(mac net.HardwareAddr) *Lease {
	lease, ok := c.leases[mac.String()]
	if ok {
		return &lease
	}
	return nil
}

func (c *EtcdClient) GetFreeIP() {

}

func (c *EtcdClient) UpdateLease(mac net.HardwareAddr, lease Lease) error {
	c.leases[mac.String()] = lease
	return nil
}
