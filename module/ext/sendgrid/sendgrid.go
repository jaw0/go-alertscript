// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Feb-01 10:24 (EST)
// Function: send email via sendgrid

package modtsendgrid

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/deduce-com/go-alertscript/module"
	"github.com/dop251/goja"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/deduce-com/go-alertscript/module/std"
)

var _ = module.Register("ext/sendgrid", install)

type mod struct {
	as module.MASer
}

type Result struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

func install(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	m := &mod{aser}
	return m
}

// optionally pass in something resembling a SGMailV3
func (m *mod) Send(key string, msg *modstd.SmtpMsg, sgm *mail.SGMailV3) (*Result, error) {

	if msg == nil || key == "" {
		return nil, fmt.Errorf("sendgrid.send(key, message)")
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
	m.as.Diagf("sending to sendgrid %s", msg.To)
	if m.as.IsDryRun() {
		return &Result{200, "dry run", nil, ""}, nil
	}

	client := sendgrid.NewSendClient(key)

	if sgm == nil {
		sgm = mail.NewV3Mail()
	}

	sgm.SetFrom(mail.NewEmail(msg.FromName, msg.From))
	sgm.AddPersonalizations(&mail.Personalization{To: []*mail.Email{mail.NewEmail(msg.ToName, msg.To)}})
	sgm.Subject = msg.Subject

	if msg.Text != "" {
		sgm.AddContent(mail.NewContent("text/plain", msg.Text))
	}
	if msg.Html != "" {
		sgm.AddContent(mail.NewContent("text/html", msg.Html))
	}

	for _, a := range msg.Attach {
		encoded := base64.StdEncoding.EncodeToString([]byte(a.Content))

		sgm.AddAttachment(&mail.Attachment{
			Filename:    a.Name,
			Type:        a.Type,
			Content:     encoded,
			Disposition: "attachment",
		})
	}

	for k, a := range msg.Header {
		if len(a) > 1 {
			sgm.SetHeader(k, a[0]) // api supports only one
		}
	}

	ctx, _ := context.WithTimeout(context.Background(), m.as.NetTimeout())
	res, err := client.SendWithContext(ctx, sgm)

	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("sendgrid error %v", err)
		return &Result{500, err.Error(), nil, ""}, nil
	}

	sm := "OK"

	if res.StatusCode != 200 {
		sm = "?"
	}
	return &Result{res.StatusCode, sm, res.Headers, res.Body}, nil

}
