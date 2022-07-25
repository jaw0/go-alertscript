// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jul-24 10:41 (EDT)
// Function:

package stdsyslog

import (
	"fmt"
	"testing"
)

func TestSyslog(t *testing.T) {

	proto, addr, err := parseDst("tls://127.0.0.1:4321")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if proto != "tls" {
		t.Fatalf("proto: %s", proto)
	}
	if addr != "127.0.0.1:4321" {
		t.Fatalf("addr: %s", addr)
	}

}

func TestCEF(t *testing.T) {
	m := &modSyslog{}
	c := &CefEvent{
		DeviceVendor:       "acme",
		DeviceProduct:      "scrapple",
		DeviceVersion:      "1.414",
		DeviceEventClassId: "A+",
		Name:               "Acme Scrapple - Large",
		Severity:           "high",
		Extensions: map[string]string{
			"foo": "bar",
		},
	}

	p, err := m.CEF(c)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if p != "CEF:0|acme|scrapple|1.414|A+|Acme Scrapple - Large|high|foo=bar" {
		fmt.Printf(">> %s\n", p)
		t.Fail()
	}
}
