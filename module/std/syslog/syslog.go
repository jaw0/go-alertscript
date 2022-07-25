// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jul-23 18:31 (EDT)
// Function: send syslog messages (RFC 5424)

package stdsyslog

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/dop251/goja"
	"github.com/jaw0/go-alertscript/module"
	"github.com/jaw0/go-syslog"
	"github.com/pcktdmp/cef/cefevent"
)

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

type Message struct {
	time     time.Time
	Legacy   bool                 `json:"legacy"` // legacy bsd format
	Severity string               `json:"severity"`
	Facility string               `json:"facility"`
	Hostname string               `json:"hostname"`
	AppName  string               `json:"appname"`
	Message  string               `json:"message"`
	SdData   []*syslog.Structured `json:"sd_data"`
}

type CefEvent struct {
	Version            int               `json:"cef_version"`
	DeviceVendor       string            `json:"vendor"`
	DeviceProduct      string            `json:"product"`
	DeviceVersion      string            `json:"version"`
	DeviceEventClassId string            `json:"classid"`
	Name               string            `json:"name"`
	Severity           string            `json:"severity"`
	Extensions         map[string]string `json:"extensions"`
}

func (m *modSyslog) Send(dst string, msg *Message) error {

	if msg.Hostname == "" {
		msg.Hostname, _ = os.Hostname()
	}

	proto, addr, err := parseDst(dst)
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

	sev, err := syslog.Severity(msg.Severity)
	if err != nil {
		return fmt.Errorf("invalid severity '%s'", msg.Severity)
	}

	opts := []syslog.OptFunc{
		syslog.WithDst(proto, addr),
		syslog.WithTimeout(m.as.NetTimeout()),
		syslog.WithDialer(&net.Dialer{Timeout: m.as.NetTimeout()}),
		syslog.WithHostname(msg.Hostname),
		syslog.WithAppName(msg.AppName),
		syslog.WithFacilityName(msg.Facility),
	}

	if msg.Legacy {
		opts = append(opts, syslog.WithLegacyFormat())
	}
	slog, err := syslog.New(opts...)

	if err != nil {
		return fmt.Errorf("cannot send syslog: %v", err)
	}

	defer slog.Close()
	err = slog.Send(sev, syslog.Message{
		SData:   msg.SdData,
		Message: msg.Message,
	})

	return err
}

func parseDst(dst string) (string, string, error) {
	u, err := url.Parse(dst)

	if err != nil {
		return "", "", err
	}

	return u.Scheme, u.Host, nil
}

func (m *modSyslog) Cef(cc *CefEvent) (string, error) {

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
