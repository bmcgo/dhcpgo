package main

import (
	"testing"
)

func assertTrue(t *testing.T, b bool) {
	if !b {
		t.Fail()
	}
}

func TestIPv4_Parse_Next_Inc(t *testing.T) {
	i := IPv4(1)
	assertTrue(t, "0.0.0.1" == i.String())
	i.Inc()
	assertTrue(t, "0.0.0.2" == i.String())
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
