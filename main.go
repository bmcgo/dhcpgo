package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		config EtcdClientConfig
		endpoints string
	)

	flag.StringVar(&endpoints, "etcd-url", "", "etcd server endpoints comma separated")
	flag.StringVar(&config.keyPath, "etcd-key", "", "etcd tls key")
	flag.StringVar(&config.certPath, "etcd-cert", "", "etcd tls cert")
	flag.StringVar(&config.caCertPath, "etcd-ca", "", "etcd tls ca")
	flag.StringVar(&config.prefix, "etcd-path", "/dhcpg", "etcd prefix for data")
	flag.Parse()

	if config.keyPath == "" || config.certPath == "" || config.caCertPath == "" || config.prefix == "" || endpoints == "" {
		flag.Usage()
		os.Exit(1)
	}

	config.endpoints = strings.Split(endpoints, ",")
	etcd, err := NewEtcdClient(context.TODO(), &config, time.Second * 10)
	if err != nil {
		log.Fatalf("unable to connect: %s", err)
	}

	manager := NewServerManager(etcd)

	etcd.WatchConfig(context.Background(), manager)

	iface, err := net.InterfaceByName("br0")
	if err != nil {
		log.Println("invalid interface")
		os.Exit(1)
	}
	responder, err := NewResponder(*iface)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	server, err := NewServer("br0", "0.0.0.0", responder, etcd)
	if err != nil {
		log.Printf("server error: %s", err)
		os.Exit(1)
	}
	log.Printf("server started: %v", server)
	err = server.Serve()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
