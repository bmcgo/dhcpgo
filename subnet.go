package main

import (
	"errors"
	"time"
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
	return subnet, nil
}

func (r *Subnet) incrementCurrentIP() {
	r.currentIP.Inc()
	if r.currentIP > r.iPTo {
		r.currentIP = r.iPFrom
	}
}

func (r *Subnet) GetLeaseForMAC(mac string) *Lease {
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
