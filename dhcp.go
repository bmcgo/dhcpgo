package main

import (
	"fmt"
	"net"
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
	ip = ip | 0xff0000 & uint32(n<<16)

	n, err = strconv.ParseUint(bits[2], 10, 8)
	if err != nil {
		return 0, err
	}
	ip = ip | 0xff00 & uint32(n<<8)

	n, err = strconv.ParseUint(bits[3], 10, 8)
	if err != nil {
		return 0, err
	}
	ip = ip | 0xff & uint32(n)

	return IPv4(ip), nil
}

func (i *IPv4) String() string {
	ip := uint32(*i)
	return fmt.Sprintf("%d.%d.%d.%d",
		ip >> 24 & 0xff,
		ip >> 16 & 0xff,
		ip >> 8 & 0xff,
		ip & 0xff)
}

func (i *IPv4) Inc() {
	*i++
}

func (i *IPv4) Next() *IPv4 {
	ip := *i+1
	return &ip
}

type Lease struct {
	IP         string    `json:"ip"`
	Ack        bool      `json:"ack"`
	LastUpdate time.Time `json:"lastUpdate"`
}

type Range struct {
	IPFrom net.IP
	IPTo net.IP
	lastIP net.IP
	leaseCache map[string]Lease
}

func NewRange(from string, to string) (*Range, error) {
	//TODO: validate IPFrom < IPTo
	return &Range{
		IPFrom: net.ParseIP(from),
		IPTo: net.ParseIP(to),
		lastIP: net.ParseIP(from),
		leaseCache: make(map[string]Lease),
	},
	nil
}

func (r *Range) GetFreeIP() *Lease {
	//var lease Lease
	var ok bool
	r.lastIP[0] += 1
	for {
		//lease, ok = r.leaseCache[r.lastIP.String()]
		if !ok {

		}
		return nil
	}
}