package main

import (
	"context"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
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
	client     *clientv3.Client
}

func NewEtcdClient(ctx context.Context, c *EtcdClientConfig, timeout  time.Duration) (*EtcdClient, error) {
	client := &EtcdClient{}
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
