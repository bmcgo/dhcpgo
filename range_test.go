package main

import (
	"log"
	"testing"
	"time"
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
	r, err := NewRange("10.1.1.1", "10.1.1.3", time.Second * 5)
	assertTrue(t, err == nil)
	l1 := r.GetLeaseForMAC("00:00:00:00:00:01")
	assertEqual(t, l1.IP, "10.1.1.1")
	l2 := r.GetLeaseForMAC("00:00:00:00:00:02")
	assertEqual(t, l2.IP,  "10.1.1.2")
	l3 := r.GetLeaseForMAC("00:00:00:00:00:03")
	assertEqual(t, l3.IP, "10.1.1.3")
	l3 = r.GetLeaseForMAC("00:00:00:00:00:03")
	assertEqual(t, l3.IP, "10.1.1.3")
	l4 := r.GetLeaseForMAC("00:00:00:00:00:04")
	assertTrue(t, l4 == nil)
}