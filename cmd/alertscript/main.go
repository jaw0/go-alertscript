// Copyright (c) 2021
// Author: Jeff Weisberg
// Created: 2021-Jan-13 11:05 (EST)
// Function: for testing an alertscript

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/deduce-com/go-alertscript"
	_ "github.com/deduce-com/go-alertscript/module/std/memstore" // mock kvs
)

type event struct {
	Type    string `json:"type"`
	Ehls1   string `json:"ehls1"`
	IP_req  string `json:"request_ip"` // address of the original request
	IP_clk  string `json:"click_ip"`   // address of the current click action
	Device  string `json:"device"`
	AlertId string `json:"id"`
	RefId   string `json:"refid"`
	Auth    string `json:"auth"`
	IsTest  bool   `json:"testmode"`
	Country string `json:"country"`
}

type Logger struct{}

func main() {
	web_n := false
	var evtType string

	flag.BoolVar(&web_n, "n", false, "do not perform web requests")
	flag.StringVar(&evtType, "e", "yes", "event type")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Printf("usage: %s [opts] script\n", os.Args[0])
		os.Exit(1)
	}

	// simulated event
	data := &event{
		Type:    evtType,
		Ehls1:   "b84c4c03b2af4050ac2d3b105e58adf83fa5af05",
		IP_req:  "192.0.2.123",
		IP_clk:  "2001:db8:0:0:0:0:0:7b",
		AlertId: "aGVsbG8gd29ybGQK",
		RefId:   "f6b9743c-32a1-4d00-81bb-b8a62b947552",
		Auth:    "SSBsb3ZlIGJhc2U2NCBlbmNvZGVkIGF1dGggdG9rZW5z",
		Country: "US",
		IsTest:  true,
	}

	// read script
	file := args[0]
	script, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("cannot open file '%s': %v\n", file, err)
		os.Exit(1)
	}

	// run using typical settings, actual production servers may vary...
	_, err = alertscript.Run(&alertscript.Conf{
		Script:   string(script),
		Timeout:  time.Second,
		NetMock:  web_n,
		NetMax:   2,
		Logger:   Logger{},
		DataName: "event",
		Data:     data,
	})

	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}

func (l Logger) Verbose(s string, p ...interface{}) {
	fmt.Printf("%s\n", fmt.Sprintf(s, p...))
}
func (l Logger) Debug(s string, p ...interface{}) {
	fmt.Printf("[diag]> %s\n", fmt.Sprintf(s, p...))
}
func (l Logger) Error(err error) {
	fmt.Printf("[error]> %v\n", err)
}
