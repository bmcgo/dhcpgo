package dhcp

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

	Subnet     string    `json:"subnet"`
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
	ipNet      net.IPNet
	leaseTime  time.Duration
	currentIP  IPv4
	leaseCache map[string]*Lease
	netMask    string
}

func (s *Subnet) Contains(ip net.IP) bool {
	return s.ipNet.Contains(ip)
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
	ipAddr := net.ParseIP(sn[0])
	ipMask := net.CIDRMask(int(prefixLength), 32)
	subnet.ipNet = net.IPNet{
		IP:   ipAddr,
		Mask: ipMask,
	}
	subnet.netMask = net.IP(ipMask).String()
	return subnet, nil
}

func (s *Subnet) incrementCurrentIP() {
	s.currentIP.Inc()
	if s.currentIP > s.iPTo {
		s.currentIP = s.iPFrom
	}
}

func (s *Subnet) GetLeaseForMAC(mac string) *Lease {
	var (
		lease       *Lease
		oldestLease *Lease
		ok          bool
	)

	lease, ok = s.leaseCache[mac]
	if ok {
		return lease
	}

	if s.currentIP == 0 {
		s.currentIP = s.iPFrom
	} else {
		s.incrementCurrentIP()
	}
	expiredTime := time.Now().Add(-s.leaseTime)
	firstIp := s.currentIP
	for {
		lease, ok = s.leaseCache[s.currentIP.String()]
		if !ok {
			lease = &Lease{
				IP:         s.currentIP.String(),
				LastUpdate: time.Now(),
				Options:    s.Options,
				NetMask:    s.netMask,
				Gateway:    s.Gateway,
				DNS:        s.DNS,
				LeaseTime:  defaultLeaseTime, //TODO
			}
			s.leaseCache[lease.IP] = lease
			s.leaseCache[mac] = lease
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
		s.incrementCurrentIP()
		if firstIp == s.currentIP {
			if oldestLease != nil {
				return oldestLease
			} else {
				return nil
			}
		}
	}
}
