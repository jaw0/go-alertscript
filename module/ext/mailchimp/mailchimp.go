// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Feb-01 19:08 (EST)
// Function: send email via mailchimp

package modmailchimp

import (
	"fmt"

	"github.com/deduce-com/go-alertscript/module"
	"github.com/deduce-com/go-alertscript/module/std"
	"github.com/dop251/goja"
)

const (
	host = "smtp.mandrillapp.com"
	port = 587
)

var _ = module.Register("ext/mailchimp", install)

type mod struct {
	as module.MASer
}

func install(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	m := &mod{aser}
	return m
}

func (m *mod) Send(creds *modstd.SmtpServer, msg *modstd.SmtpMsg) (*modstd.SmtpResult, error) {

	if creds == nil || msg == nil {
		return nil, fmt.Errorf("mailchimp.send(creds, message)")
	}
	creds.Hostname = host
	creds.Port = port

	smtp := modstd.NewSmtp(m.as)
	return smtp.Send(creds, msg)
}
