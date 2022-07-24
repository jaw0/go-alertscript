// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jan-29 13:54 (EST)
// Function: send thing via twilio

package modtwilio

import (
	"fmt"

	"github.com/jaw0/go-alertscript/module"
	"github.com/dop251/goja"
	// these are awful. just awful. absolutely awful.
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

var _ = module.Register("ext/twilio", install)

const twilioUrl = "https://api.twilio.com/"

type mod struct {
	as module.MASer
}

type Creds struct {
	SID   string `json:"accound_sid"`
	Token string `json:"auth_token"`
}

type Result struct {
	Code     int         `json:"code"`
	Message  string      `json:"message"`
	Response interface{} `json:"response"` // QQQ?
}

func install(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	m := &mod{aser}
	return m
}

// optionally pass in something resembling CreateMessageParams
func (m *mod) Message(creds *Creds, to, from string, text string, params *openapi.CreateMessageParams) (*Result, error) {

	if creds == nil {
		return nil, fmt.Errorf("must supply twilio credentials")
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
	m.as.Diagf("sending to twilio (message) %s", to)
	if m.as.IsDryRun() {
		return &Result{200, "dry run", nil}, nil
	}

	client := twilio.NewRestClientWithParams(twilio.RestClientParams{
		Username: creds.SID,
		Password: creds.Token,
	})

	if params == nil {
		params = &openapi.CreateMessageParams{}
	}
	params.SetTo(to)
	params.SetFrom(from)
	params.SetBody(text)

	client.SetTimeout(m.as.NetTimeout())
	res, err := client.ApiV2010.CreateMessage(params)

	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("twilio error %v", err)
		return &Result{500, err.Error(), nil}, nil
	}

	return &Result{200, "OK", res}, nil
}

func (m *mod) Phone(creds *Creds, to, from, url string, params *openapi.CreateCallParams) (*Result, error) {

	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return nil, err
	}

	// for debugging
	m.as.Diagf("sending to twilio (phone) %s", to)
	if m.as.IsDryRun() {
		return &Result{200, "dry run", nil}, nil
	}

	client := twilio.NewRestClientWithParams(twilio.RestClientParams{
		Username: creds.SID,
		Password: creds.Token,
	})

	if params == nil {
		params = &openapi.CreateCallParams{}
	}
	params.SetTo(to)
	params.SetFrom(from)
	params.SetUrl(url)

	client.SetTimeout(m.as.NetTimeout())
	res, err := client.ApiV2010.CreateCall(params)

	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("twilio error %v", err)
		return &Result{500, err.Error(), nil}, nil
	}

	return &Result{200, "OK", res}, nil

}

// RSN - more things
