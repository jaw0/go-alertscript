// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jul-23 18:31 (EDT)
// Function: send syslog messages (RFC 5424)

package stdsyslog

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jaw0/go-alertscript/module"
	"github.com/dop251/goja"
	"github.com/pcktdmp/cef/cefevent"
)

// RFC 5424 - base
// RFC 6587 - tcp
// RFC 5426 - udp
// RFC 5425 - tls

const rfc5424time = "2006-01-02T15:04:05.999999Z07:00"
const maxSize = 8192

var _ = module.Register("std/syslog", installSyslog)

type modSyslog struct {
	as module.MASer
}

func installSyslog(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	m := &modSyslog{
		as: aser,
	}
	return m
}

type Structured struct {
	Name       string            `json:"name"`
	Enterprise string            `json:"enterprise"` // can be a dotted number / snmp OID
	Param      map[string]string `json:"param"`
}

// NB. Loggly's Ingestion Token : token @ 41058

type Message struct {
	time     time.Time
	Legacy   bool          `json:"legacy"` // legacy bsd format
	Severity string        `json:"severity"`
	Facility string        `json:"facility"`
	Hostname string        `json:"hostname"`
	AppName  string        `json:"appname"`
	Message  string        `json:"message"`
	SdData   []*Structured `json:"sd_data"`
	CEF      *CefEvent     `json:"cef"`
	//LEEF     LEEFMsg `json:"leef"`
}

type CefEvent struct {
	Version            int               `json:"version"`
	DeviceVendor       string            `json:"vendor"`
	DeviceProduct      string            `json:"product"`
	DeviceVersion      string            `json:"device_version"`
	DeviceEventClassId string            `json:"classid"`
	Name               string            `json:"name"`
	Severity           string            `json:"severity"`
	Extensions         map[string]string `json:"extensions"`
}

func (m *modSyslog) Send(dst string, msg *Message) error {

	if msg.Hostname == "" {
		msg.Hostname, _ = os.Hostname()
	}

	msg.time = time.Now()

	pkt, err := msg.buildMessage()
	if err != nil {
		return err
	}

	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return err
	}

	// for debugging
	m.as.Diagf("sending syslog to: %s", dst)
	if m.as.IsDryRun() {
		return nil
	}

	err = m.sendPkt(dst, pkt)
	return err
}

var defaultPort = map[string]string{
	"udp": "514",
	"tcp": "514", // 6587 3.3 - This protocol has no standardized port assignment
	"tls": "6514",
}

func (m *modSyslog) sendPkt(dst string, pkt string) error {

	proto, addr, err := parseDst(dst)
	proto, addr = proto, addr

	if err != nil {
		return fmt.Errorf("invalid dst '%s', use format 'tls://127.0.0.1:6541", dst)
	}

	var conn net.Conn
	var withLen bool

	// dial
	switch proto {
	case "udp":
		conn, err = net.Dial("udp", addr)
	case "tcp":
		conn, err = net.Dial("tcp", addr)
		withLen = true
	case "tls":
		conn, err = tls.Dial("tcp", addr, nil)
		withLen = true
	default:
		return fmt.Errorf("invalid protocol '%s', use udp, tcp. tls", proto)
	}

	if err != nil {
		return fmt.Errorf("cannot connect to '%s': %v", dst, err)
	}

	defer conn.Close()
	conn.SetDeadline(time.Now().Add(m.as.NetTimeout()))

	if withLen {
		fmt.Fprintf(conn, "%d %s", len(pkt), pkt)
	} else {
		fmt.Fprintf(conn, "%s", pkt)
	}

	return nil
}

func parseDst(dst string) (string, string, error) {
	u, err := url.Parse(dst)

	if err != nil {
		return "", "", err
	}

	if u.Port() == "" {
		port := defaultPort[u.Scheme]
		u.Host = net.JoinHostPort(u.Host, port)
	}

	return u.Scheme, u.Host, nil
}

func (m *Message) buildMessage() (string, error) {

	prio, err := m.priority()
	if err != nil {
		return "", err
	}

	if cc := m.CEF; cc != nil {
		m.Message, err = cc.Marshal()
		if err != nil {
			return "", fmt.Errorf("invalid CEF data: %v", err)
		}
	}

	ts := m.time.UTC().Format(rfc5424time)
	var msg string

	if m.Legacy {
		// rfc 3164
		msg = fmt.Sprintf("<%d> %s %s %s: %s", prio, ts, m.Hostname,
			m.AppName, m.cleanMessage())
	} else {
		msg = fmt.Sprintf("<%d>1 %s %s %s - - %s %s", prio, ts, m.Hostname,
			m.AppName, m.structuredData(), m.cleanMessage())
	}

	if maxSize != 0 && len(msg) > maxSize {
		msg = msg[0:maxSize]
	}

	return msg, nil
}

func (cc *CefEvent) Marshal() (string, error) {
	c := cefevent.CefEvent{
		Version:            cc.Version,
		DeviceVendor:       cc.DeviceVendor,
		DeviceProduct:      cc.DeviceProduct,
		DeviceVersion:      cc.DeviceVersion,
		DeviceEventClassId: cc.DeviceEventClassId,
		Name:               cc.Name,
		Severity:           cc.Severity,
		Extensions:         cc.Extensions,
	}

	return c.Generate()
}

func (m *Message) priority() (int, error) {

	sev, err := Severity(m.Severity)
	if err != nil {
		return 0, fmt.Errorf("invalid severity '%s'", m.Severity)
	}
	fac, err := Facility(m.Facility)
	if err != nil {
		return 0, fmt.Errorf("invalid facility '%s'", m.Facility)
	}

	return int((fac << 3) | sev), nil
}

func (m *Message) cleanMessage() string {

	s := strings.Map(func(ch rune) rune {
		if ch == '\n' {
			return ' '
		}
		if ch < ' ' {
			return -1
		}
		return ch
	}, m.Message)

	return s
}

func (m *Message) structuredData() string {
	res := bytes.NewBuffer(nil)

	for _, sd := range m.SdData {
		if sd.Enterprise == "" {
			fmt.Fprintf(res, "[%s", sd.Name)
		} else {
			fmt.Fprintf(res, "[%s@%s", sd.Name, sd.Enterprise)
		}

		for k, v := range sd.Param {
			fmt.Fprintf(res, ` %s="%s"`, k, sdValue(v))
		}
		fmt.Fprintf(res, "]")
	}

	if res.Len() == 0 {
		return "-"
	}

	return res.String()
}

func sdValue(s string) string {
	// 5424 6.3.3
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `]`, `\]`)
	return s
}
