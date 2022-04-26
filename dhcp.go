package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type IPv4 uint32

func ParseIPv4(s string) (IPv4, error) {
	bits := strings.Split(s, ".")
	if len(bits) != 4 {
		return 0, fmt.Errorf("invalid ipv4: %s", s)
	}
	var ip uint32
	var n uint64
	var err error

	n, err = strconv.ParseUint(bits[0], 10, 8)
	if err != nil {
		return 0, err
	}
	ip = 0xff000000 & uint32(n<<24)

	n, err = strconv.ParseUint(bits[1], 10, 8)
	if err != nil {
		return 0, err
	}
	ip = ip | 0xff0000&uint32(n<<16)

	n, err = strconv.ParseUint(bits[2], 10, 8)
	if err != nil {
		return 0, err
	}
	ip = ip | 0xff00&uint32(n<<8)

	n, err = strconv.ParseUint(bits[3], 10, 8)
	if err != nil {
		return 0, err
	}
	ip = ip | 0xff&uint32(n)

	return IPv4(ip), nil
}

func (i IPv4) String() string {
	ip := uint32(i)
	return fmt.Sprintf("%d.%d.%d.%d",
		ip>>24&0xff,
		ip>>16&0xff,
		ip>>8&0xff,
		ip&0xff)
}

func (i *IPv4) Inc() {
	*i++
}

func (i *IPv4) Next() *IPv4 {
	ip := *i + 1
	return &ip
}

type Lease struct {
	IP         string    `json:"ip"`
	Ack        bool      `json:"ack"`
	LastUpdate time.Time `json:"lastUpdate"`
}

type Range struct {
	IPFrom    IPv4
	IPTo      IPv4
	LeaseTime time.Duration

	currentIP  IPv4
	leaseCache map[string]*Lease
}

func NewRange(from string, to string, leaseTime time.Duration) (*Range, error) {
	ipfrom, err := ParseIPv4(from)
	if err != nil {
		return nil, err
	}
	ipto, err := ParseIPv4(to)
	if err != nil {
		return nil, err
	}
	if from > to {
		return nil, errors.New("from > to")
	}

	//TODO: validate IPFrom < IPTo
	return &Range{
			IPFrom:     ipfrom,
			IPTo:       ipto,
			LeaseTime:  leaseTime,
			leaseCache: make(map[string]*Lease),
		},
		nil
}

func (r *Range) incrementCurrentIP() {
	r.currentIP.Inc()
	if r.currentIP > r.IPTo {
		r.currentIP = r.IPFrom
	}
}

func (r *Range) FindFreeIP() *Lease {
	if r.currentIP == 0 {
		r.currentIP = r.IPFrom
	} else {
		r.incrementCurrentIP()
	}
	var lease *Lease
	var oldestLease *Lease
	var ok bool
	expiredTime := time.Now().Add(-r.LeaseTime)
	firstIp := r.currentIP
	for {
		lease, ok = r.leaseCache[r.currentIP.String()]
		if !ok {
			lease = &Lease{
				IP:         r.currentIP.String(),
				LastUpdate: time.Now(),
			}
			r.leaseCache[lease.IP] = lease
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
