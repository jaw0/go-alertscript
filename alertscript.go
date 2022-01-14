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

type logger interface {
	Debug(string, ...interface{})
	Verbose(string, ...interface{})
}

type Conf struct {
	Script     string
	DataName   string
	Data       interface{}
	Timeout    time.Duration
	NetTimeout time.Duration
	NetMax     int
	NetMock    bool
	Init       func(*goja.Runtime)
	Logger     logger
}

type AS struct {
	cf      *Conf
	vm      *goja.Runtime
	Result  goja.Value
	NetReqs int
	NetErrs int
	NetTime time.Duration
}

const (
	defaultTimeout    = 2 * time.Second
	defaultWebTimeout = 1 * time.Second
)

func Run(cf *Conf) (*AS, error) {

	vm := goja.New()
	as := &AS{cf: cf, vm: vm}
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", false))

	if cf.NetTimeout == 0 {
		cf.NetTimeout = defaultWebTimeout
	}
	if cf.Timeout == 0 {
		cf.Timeout = defaultTimeout
	}

	// wire up console.log output
	logger := func(c goja.FunctionCall) goja.Value {
		as.Log(joinJsArgs(c))
		return nil
	}
	debugger := func(c goja.FunctionCall) goja.Value {
		as.Diag(joinJsArgs(c))
		return nil
	}

	vm.Set("console", map[string]interface{}{
		"log":   logger,
		"warn":  logger,
		"error": logger,
		"debug": debugger,
	})

	// provide useful functions and data
	vm.Set("webRequest", func(c goja.FunctionCall) goja.Value {
		res, err := as.webRequest(c)
		if err != nil {
			vm.Interrupt(err)
		}
		return vm.ToValue(res)
	})

	// install js functions
	installUtilFuncs(vm)
	if cf.Init != nil {
		cf.Init(vm)
	}
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

// rom console.log (et al)
func (as *AS) Log(s string) {
	as.Logf("%s", s)
}
func (as *AS) Logf(s string, args ...interface{}) {
	if as.cf.Logger != nil {
		as.cf.Logger.Verbose(s, args...)
	}
}

// from internals
func (as *AS) Diag(s string) {
	as.Diagf("%s", s)
}

func (as *AS) Diagf(s string, args ...interface{}) {
	if as.cf.Logger != nil {
		as.cf.Logger.Debug(s, args...)
	}
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
