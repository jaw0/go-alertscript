// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jul-24 10:41 (EDT)
// Function:

package stdsyslog

import (
	"fmt"
	"testing"
	"time"
)

func TestSyslog(t *testing.T) {

	m := Message{
		time:     time.Unix(1, 0),
		Hostname: "localhost.example.com",
		Severity: "info",
		Facility: "local2",
		AppName:  "test",
		Message:  "foo\nbar",
		SdData: []*Structured{
			{
				Name:       "Foo",
				Enterprise: "32473",
				Param: map[string]string{
					"girth": "foo\\bar]s",
				},
			},
		},
	}

	pkt, err := m.buildMessage()

	if err != nil {
		t.Fatalf("error: %v", err)
	}

	exp := `<150>1 1970-01-01T00:00:01Z localhost.example.com test - - [Foo@32473 girth="foo\\bar\]s"] foo bar`

	if pkt != exp {
		fmt.Printf(">> %s\n!= %s\n", pkt, exp)
		t.Fail()
	}

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

	c := CefEvent{
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

	p, err := c.Marshal()
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if p != "CEF:0|acme|scrapple|1.414|A+|Acme Scrapple - Large|high|foo=bar" {
		fmt.Printf(">> %s\n", p)
		t.Fail()
	}
}
