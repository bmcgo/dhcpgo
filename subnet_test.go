package main

import (
	"log"
	"testing"
)

func assertTrue(t *testing.T, b bool) {
	if !b {
		log.Println("not true")
		t.Fail()
	}
}

func assertEqual(t *testing.T, expected interface{}, actual interface{}) {
	if expected != actual {
		log.Printf("%v != %v", expected, actual)
		t.Fail()
	}
}

func TestNewRange_GetLeaseForMAC(t *testing.T) {
	s := &Subnet{Subnet: "10.1.1.0/24", RangeFrom: "10.1.1.1", RangeTo: "10.1.1.3"}
	_, err := InitializeSubnet(s)
	assertTrue(t, err == nil)
	l1 := s.GetLeaseForMAC("00:00:00:00:00:01")
	assertEqual(t, l1.IP, "10.1.1.1")
	l2 := s.GetLeaseForMAC("00:00:00:00:00:02")
	assertEqual(t, l2.IP, "10.1.1.2")
	l3 := s.GetLeaseForMAC("00:00:00:00:00:03")
	assertEqual(t, l3.IP, "10.1.1.3")
	l3 = s.GetLeaseForMAC("00:00:00:00:00:03")
	assertEqual(t, l3.IP, "10.1.1.3")
	l4 := s.GetLeaseForMAC("00:00:00:00:00:04")
	assertTrue(t, l4 == nil)
}