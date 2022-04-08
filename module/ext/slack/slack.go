// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jan-29 13:54 (EST)
// Function: post to slack

package modslack

import (
	"context"

	"github.com/deduce-com/go-alertscript/module"
	"github.com/dop251/goja"
	"github.com/slack-go/slack"
)

var _ = module.Register("ext/slack", install)

type mod struct {
	as module.MASer
}

type logger struct {
	as module.MASer
}

type Result struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func install(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	m := &mod{aser}
	return m
}

func (m *mod) Post(token string, channel string, msgs ...slack.Attachment) (*Result, error) {

	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return nil, err
	}

	// for debugging
	m.as.Diagf("posting to slack chan %s", channel)
	if m.as.IsDryRun() {
		return &Result{200, "dry run"}, nil
	}

	client := slack.New(token, slack.OptionDebug(true), slack.OptionLog(logger{m.as}))

	// QQQ - other options?
	ctx, _ := context.WithTimeout(context.Background(), m.as.NetTimeout())
	_, _, err = client.PostMessageContext(
		ctx, channel, slack.MsgOptionAttachments(msgs...),
	)
	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("slack error %v", err)
		return &Result{500, err.Error()}, nil
	}

	return &Result{200, "OK"}, nil
}

func (m logger) Output(l int, msg string) error {
	m.as.Diagf("%s", msg)
	return nil

}
