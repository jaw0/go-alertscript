// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jan-27 10:45 (EST)
// Function: send email via smtp

package modstd

import (
	"bytes"
	"fmt"
	"net"
	"net/smtp"

	"github.com/jaw0/go-alertscript/module"
	"github.com/domodwyer/mailyak"
	"github.com/dop251/goja"
)

var _ = module.Register("std/smtp", installSMTP)

// exported to js:
type modSMTP struct {
	as module.MASer
	// QQQ - provide helpers?
}

type SmtpServer struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type SmtpAttach struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // MIME-type
	Content string `json:"content"`
}

type SmtpMsg struct {
	To       string              `json:"to"`
	ToName   string              `json:"to_name"`
	From     string              `json:"from"`
	FromName string              `json:"from_name"`
	ReplyTo  string              `json:"reply_to"`
	Subject  string              `json:"subject"`
	Text     string              `json:"text"`
	Html     string              `json:"html"`
	Header   map[string][]string `json:"header"`
	Attach   []SmtpAttach        `json:"attach"`
}

type SmtpResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func installSMTP(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	m := &modSMTP{aser}
	return m
}

func NewSmtp(aser module.MASer) *modSMTP {
	return &modSMTP{aser}
}

func (m *modSMTP) Send(srv *SmtpServer, msg *SmtpMsg) (*SmtpResult, error) {

	if srv == nil || msg == nil {
		return nil, fmt.Errorf("smtp.send(server, message)")
	}
	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return nil, err
	}

	// for debugging
	m.as.Diagf("sending mail to: %s via: %s", msg.To, srv.Hostname)
	if m.as.IsDryRun() {
		return &SmtpResult{200, "not tried"}, nil
	}

	if srv.Port == 0 {
		srv.Port = 25
	}

	var mail *mailyak.MailYak

	if srv.Username != "" {
		mail = mailyak.New(net.JoinHostPort(srv.Hostname, fmt.Sprintf("%d", srv.Port)),
			smtp.PlainAuth("", srv.Username, srv.Password, srv.Hostname))
	} else {
		mail = mailyak.New(net.JoinHostPort(srv.Hostname, fmt.Sprintf("%d", srv.Port)), nil)
	}

	mail.To(msg.To)
	mail.From(msg.From)
	mail.FromName(msg.FromName)
	mail.Subject(msg.Subject)
	mail.ReplyTo(msg.ReplyTo)

	if msg.Text != "" {
		mail.Plain().Set(msg.Text)
	}
	if msg.Html != "" {
		mail.HTML().Set(msg.Html)
	}

	for k, a := range msg.Header {
		for _, v := range a {
			mail.AddHeader(k, v)
		}
	}

	// for troubleshooting
	ti := m.as.TraceInfo()
	if ti != "" {
		mail.AddHeader("X-Trace-Info", ti)
	}

	for i := range msg.Attach {
		a := &msg.Attach[i]
		b := bytes.NewBufferString(a.Content)
		mail.AttachWithMimeType(a.Name, b, a.Type)
	}

	err = mail.Send()
	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("smtp error %v", err)
		return &SmtpResult{500, err.Error()}, nil
	}

	return &SmtpResult{200, "OK"}, nil
}
