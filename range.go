package main

import (
	"errors"
	"time"
)

type Lease struct {
	IP         string    `json:"ip"`
	MAC        string    `json:"mac"`
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

func (r *Range) GetLeaseForMAC(mac string) *Lease {
	//TODO: validate mac
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
		r.currentIP = r.IPFrom
	} else {
		r.incrementCurrentIP()
	}
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
