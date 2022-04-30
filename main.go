package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"
)

func getenv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is empty", key)
	}
	return value
}

func main() {
	var (
		config EtcdClientConfig
		endpoints string
	)

	endpoints = getenv("DHCPGO_ETCD_ENDPOINTS")
	config.keyPath = getenv("DHCPGO_ETCD_KEY")
	config.certPath = getenv("DHCPGO_ETCD_CERT")
	config.caCertPath = getenv("DHCPGO_ETCD_CACERT")
	config.prefix = getenv("DHCPGO_ETCD_PREFIX")

	config.endpoints = strings.Split(endpoints, ",")
	etcd, err := NewEtcdClient(context.TODO(), &config, time.Second * 10)
	if err != nil {
		log.Fatalf("unable to connect: %s", err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "configure":
			if len(os.Args) < 3 {
				log.Println("Usage configure: TODO")
				os.Exit(1)
			}
			tool := NewDhcpgoTool(context.Background(), etcd)
			err = tool.Configure(os.Args[2:])
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
		default:
			//TODO: Usage
			log.Println("Usage: TODO")
		}
		return
	}

	manager := NewServerManager(etcd)
	etcd.WatchConfig(context.Background(), manager)
	log.Printf("Exited")
}
