package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLeaseTime = 14400 //4 hours
)

type Lease struct {
	MAC       string   `json:"mac"`
	IP        string   `json:"ip"`
	NetMask   string   `json:"netMask"`
	Gateway   string   `json:"gateway,omitempty"`
	DNS       []string `json:"dns,omitempty"`
	Options   []Option `json:"options,omitempty"`
	LeaseTime int      `json:"leaseTime,omitempty"`

	Ack        bool      `json:"ack"`
	LastUpdate time.Time `json:"lastUpdate"`
}

type Subnet struct {
	Subnet    string   `json:"subnet"`
	RangeFrom string   `json:"rangeFrom"`
	RangeTo   string   `json:"rangeTo"`
	Gateway   string   `json:"gateway"`
	DNS       []string `json:"dns"`
	Options   []Option `json:"options"`

	iPFrom     IPv4
	iPTo       IPv4
	leaseTime  time.Duration
	currentIP  IPv4
	leaseCache map[string]*Lease
	netMask    string
}

func InitializeSubnet(subnet *Subnet) (*Subnet, error) {
	var err error
	subnet.iPFrom, err = ParseIPv4(subnet.RangeFrom)
	if err != nil {
		return nil, err
	}
	subnet.iPTo, err = ParseIPv4(subnet.RangeTo)
	if err != nil {
		return nil, err
	}
	if subnet.iPFrom > subnet.iPTo {
		return nil, errors.New("from > to")
	}
	subnet.leaseCache = make(map[string]*Lease)
	if subnet.leaseTime == 0 {
		subnet.leaseTime = defaultLeaseTime
	}
	sn := strings.Split(subnet.Subnet, "/")
	if len(sn) != 2 {
		return nil, fmt.Errorf("invalid subnet %q (%v)", subnet.Subnet, subnet)
	}
	prefixLength, err := strconv.ParseInt(sn[1], 10, 8)
	if err != nil {
		return nil, err
	}
	subnet.netMask = net.IP(net.CIDRMask(int(prefixLength), 32)).String()
	return subnet, nil
}

func (r *Subnet) incrementCurrentIP() {
	r.currentIP.Inc()
	if r.currentIP > r.iPTo {
		r.currentIP = r.iPFrom
	}
}

func (r *Subnet) GetLeaseForMAC(mac string) *Lease {
	var (
		lease       *Lease
		oldestLease *Lease
		ok          bool
	)

	lease, ok = r.leaseCache[mac]
	if ok {
		return lease
	}

	if r.currentIP == 0 {
		r.currentIP = r.iPFrom
	} else {
		r.incrementCurrentIP()
	}
	expiredTime := time.Now().Add(-r.leaseTime)
	firstIp := r.currentIP
	for {
		lease, ok = r.leaseCache[r.currentIP.String()]
		if !ok {
			lease = &Lease{
				IP:         r.currentIP.String(),
				LastUpdate: time.Now(),
				Options:    r.Options,
				NetMask:    r.netMask,
			}
			r.leaseCache[lease.IP] = lease
			r.leaseCache[mac] = lease
			return lease
		}
		//TODO: check for ACK
		if lease.LastUpdate.Before(expiredTime) {
			if oldestLease == nil {
				oldestLease = lease
			} else {
				if oldestLease.LastUpdate.After(lease.LastUpdate) {
					oldestLease = lease
				}
			}
		}
		r.incrementCurrentIP()
		if firstIp == r.currentIP {
			if oldestLease != nil {
				oldestLease.Ack = false
				return oldestLease
			} else {
				return nil
			}
		}
	}
}
