package main

import (
	"log"
	"testing"
	"time"
)

func assertTrue(t *testing.T, b bool) {
	if !b {
		t.Fail()
	}
}

func assertEqual(t *testing.T, expected interface{}, actual interface{}) {
	if expected != actual {
		log.Printf("%v != %v", expected, actual)
		t.Fail()
	}
}

func TestIPv4_Parse_String_Next_Inc(t *testing.T) {
	i, err := ParseIPv4("1.2.3.254")
	assertTrue(t, err == nil)
	assertTrue(t, "1.2.3.254" == i.String())
	i.Inc()
	assertTrue(t, "1.2.3.255" == i.String())
	i.Inc()
	assertTrue(t, "1.2.4.0" == i.String())
	assertTrue(t, "1.2.4.1" == i.Next().String())
	i, err = ParseIPv4("1.2.3.300")
	assertTrue(t, err != nil)
}

func TestNewRange_FindFreeIP(t *testing.T) {
	r, err := NewRange("10.1.1.1", "10.1.1.3", time.Second * 5)
	assertTrue(t, err == nil)

	l1 := r.FindFreeIP()
	assertEqual(t, l1.IP, "10.1.1.1")

	l2 := r.FindFreeIP()
	assertEqual(t, l2.IP,  "10.1.1.2")

	l3 := r.FindFreeIP()
	assertEqual(t, l3.IP, "10.1.1.3")

	l4 := r.FindFreeIP()
	assertTrue(t, l4 == nil)
}