// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jan-26 16:46 (EST)
// Function: make web requests

package modstd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/deduce-com/go-alertscript/module"
	"github.com/dop251/goja"
)

var _ = module.Register("std/web", installWeb)

// exported to js:
type modWeb struct {
	as       module.MASer
	Get      goja.Value `json:"get"`
	Post     goja.Value `json:"post"`
	PostJSON goja.Value `json:"post_json"`
	PostUE   goja.Value `json:"post_urlencoded"`
}

// returned to user
type WebResult struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Body    string              `json:"body"`
	Header  map[string][]string `json:"header"`
}

func installWeb(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	wg, _ := vm.RunProgram(webGet)
	wp, _ := vm.RunProgram(webPost)
	wj, _ := vm.RunProgram(webPostJson)
	wu, _ := vm.RunProgram(webPostUrlEnc)
	m := &modWeb{aser, wg, wp, wj, wu}
	return m
}

func (m *modWeb) Request(url, method string, hdrs map[string][]string, content string) (*WebResult, error) {

	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return nil, err
	}

	// for debugging
	m.as.Diagf("web: %s %s\nheaders: %+v\nbody: %s\n", method, url, hdrs, content)
	if m.as.IsDryRun() {
		return &WebResult{200, "not tried", "", nil}, nil
	}

	// build request
	client := &http.Client{Timeout: m.as.NetTimeout()}
	req, err := http.NewRequest(method, url, bytes.NewReader([]byte(content)))
	if err != nil {
		m.as.Fatal(fmt.Errorf("webRequest: error %v", err))
		return nil, err
	}

	req.Header = hdrs

	// send request
	resp, err := client.Do(req)

	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("Request Failed: %v", err)
		return &WebResult{500, "Request Failed", err.Error(), nil}, nil
	}

	if resp.Status[0] != '2' {
		m.as.NetIOErr()
		m.as.Logf("Request Failed: %s", resp.Status)
	} else {
		m.as.Diagf("%v", resp.Status)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return &WebResult{resp.StatusCode, resp.Status[4:], string(body), resp.Header}, nil

}

var webGet = goja.MustCompile("runtime", `(function(url){ return this.request(url, 'GET') })`, false)
var webPost = goja.MustCompile("runtime", `
(function(url, hdrs, body){ return this.request(url, 'POST', hdrs, body) })`, false)

var webPostJson = goja.MustCompile("runtime", `
     (function(url, hdrs, data){
 	if( !hdrs ) hdrs = {}
         hdrs['Content-Type'] = ['application/json']
         var body = JSON.stringify( data )
         return this.request(url, 'POST', hdrs, body)
     })`, false)

var webPostUrlEnc = goja.MustCompile("runtime", `
     (function(url, hdrs, data){
 	if( !hdrs ) hdrs = {}
         hdrs['Content-Type'] = ['application/x-www-form-urlencoded']
         var k, args=[]
         for (k in data){
             args.push(encodeURIComponent(k) + "=" + encodeURIComponent(data[k]))
         }
         return this.request(url, 'POST', hdrs, args.join("&"))
     })`, false)
