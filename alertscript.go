// Copyright (c) 2021
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2021-Jan-13 11:01 (EST)
// Function: alert script

package alertscript

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
)

type Conf struct {
	Script     string
	DataName   string
	Data       interface{}
	Timeout    time.Duration
	WebTimeout time.Duration
	WebMax     int
	WebMock    bool
	Diag       func(string)
}

type AS struct {
	cf      *Conf
	vm      *goja.Runtime
	Output  string
	Result  goja.Value
	WebReqs int
	WebErrs int
}

const (
	defaultTimeout    = 2 * time.Second
	defaultWebTimeout = 1 * time.Second
)

func Run(cf *Conf) (*AS, error) {

	vm := goja.New()
	as := &AS{cf: cf, vm: vm}
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", false))

	if cf.WebTimeout == 0 {
		cf.WebTimeout = defaultWebTimeout
	}
	if cf.Timeout == 0 {
		cf.Timeout = defaultTimeout
	}

	// wire up console.log output
	logger := func(c goja.FunctionCall) goja.Value {
		as.Log(joinJsArgs(c))
		return nil
	}

	vm.Set("console", map[string]interface{}{
		"log":   logger,
		"warn":  logger,
		"error": logger,
	})

	// provide useful functions and data
	vm.Set("webRequest", func(c goja.FunctionCall) goja.Value {
		res, err := as.webRequest(c)
		if err != nil {
			vm.Interrupt(err)
		}
		return vm.ToValue(res)
	})

	installUtilFuncs(vm)
	// RSN - more functions

	if cf.DataName != "" {
		vm.Set(cf.DataName, cf.Data)
	}

	// run common runtime code to set up more functions and data
	_, err := vm.RunProgram(scriptRuntime)
	if err != nil {
		return nil, err
	}

	// enforce maximum runtime
	timer := time.AfterFunc(cf.Timeout, func() {
		vm.Interrupt("timeout - maximum runtime exceeded!")
	})
	defer timer.Stop()

	// run the script
	res, err := vm.RunString(cf.Script)
	as.Result = res

	return as, err
}

func (as *AS) Log(s string) {
	as.Output += s + "\n"
	as.Diag(s)
}
func (as *AS) Logf(s string, args ...interface{}) {
	as.Log(fmt.Sprintf(s, args...))
}

func (as *AS) Diag(s string) {
	if as.cf.Diag != nil {
		as.cf.Diag(s)
	}
}

func (as *AS) Diagf(s string, args ...interface{}) {
	as.Diag(fmt.Sprintf(s, args...))
}

func joinJsArgs(c goja.FunctionCall) string {
	out := ""
	for _, a := range c.Arguments {
		l := fmt.Sprintf("%+v", a.Export())
		if out != "" {
			out += " "
		}
		out += l
	}
	return out
}

// ################################################################

var scriptRuntime = goja.MustCompile("runtime", `
var web = {
    request: webRequest,
    get:  function(url){ return webRequest(url, 'GET', null, null) },
    post: function(url, hdrs, body){ return webRequest(url, 'POST', hdrs, body) },
    post_json: function(url, hdrs, data){
	if( !hdrs ) hdrs = {}
        hdrs['Content-Type'] = 'application/json'
        var body = JSON.stringify( data )
        return webRequest(url, 'POST', hdrs, body)
    },
    post_urlencoded: function(url, hdrs, data){
	if( !hdrs ) hdrs = {}
        hdrs['Content-Type'] = 'application/x-www-form-urlencoded'
        var k, args=[]
        for (k in data){
            args.push(encodeURIComponent(k) + "=" + encodeURIComponent(data[k]))
        }
        return webRequest(url, 'POST', hdrs, args.join("&"))
    }
}
`, false)
