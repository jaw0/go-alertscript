// Copyright (c) 2021
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2021-Jan-13 11:01 (EST)
// Function: alert script

package alertscript

import (
	"fmt"
	"time"

	"github.com/deduce-com/go-alertscript/module"
	"github.com/dop251/goja"

	_ "github.com/deduce-com/go-alertscript/module/std"
)

type logger interface {
	Debug(string, ...interface{})
	Verbose(string, ...interface{})
	Error(error)
}

type Conf struct {
	Script      string
	DataName    string
	Data        interface{}
	Timeout     time.Duration
	NetTimeout  time.Duration
	HardTimeout time.Duration
	NetMax      int
	NetMock     bool
	Init        func(*goja.Runtime)
	Logger      logger
	Trace       string
}

type AS struct {
	cf        *Conf
	vm        *goja.Runtime
	t0        time.Time
	t1        time.Time
	tacc      time.Duration
	timer     *time.Timer
	Result    goja.Value
	NetReqs   int
	LocalReqs int
	NetErrs   int
	NetTime   time.Duration
}

type mAS struct {
	as *AS
}

const (
	defaultTimeout    = 2 * time.Second
	defaultWebTimeout = 1 * time.Second
	defaultHard       = 30 * time.Second
)

const (
	CTL_PAUSE  = 0
	CTL_RESUME = 1
)

func Run(cf *Conf) (*AS, error) {

	vm := goja.New()
	as := &AS{cf: cf, vm: vm}
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	if cf.NetTimeout == 0 {
		cf.NetTimeout = defaultWebTimeout
	}
	if cf.Timeout == 0 {
		cf.Timeout = defaultTimeout
	}
	if cf.HardTimeout == 0 {
		cf.HardTimeout = defaultHard
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
	erroror := func(c goja.FunctionCall) goja.Value {
		cf.Logger.Error(fmt.Errorf("%s", joinJsArgs(c)))
		return nil
	}

	vm.Set("console", map[string]interface{}{
		"log":   logger,
		"warn":  logger,
		"error": erroror,
		"debug": debugger,
	})

	// provide useful functions and data
	module.InstallModule(mAS{as}, vm)
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
		cf.Logger.Error(err)
		return nil, err
	}

	// enforce maximum runtime
	as.t0 = time.Now()
	as.t1 = as.t0
	as.timer = time.AfterFunc(cf.Timeout, func() {
		vm.Interrupt("timeout - maximum runtime exceeded!")
	})
	defer as.timer.Stop()

	// run the script
	res, err := vm.RunString(cf.Script)
	as.Result = res

	if err != nil {
		cf.Logger.Error(err)
	}

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

func (as *AS) pauseTimer() {
	// update the state, move the timer to the hard limit
	t := time.Now()
	as.tacc += t.Sub(as.t1)
	as.t1 = t
	hard := as.t0.Add(as.cf.HardTimeout).Sub(t)
	as.timer.Reset(hard)
}
func (as *AS) resumeTimer() {
	// move the time to the std limit
	t := time.Now()
	as.t1 = t
	tick := as.cf.Timeout - as.tacc
	if tick < 0 {
		tick = 0
	}
	as.timer.Reset(tick)
}

// ****************************************************************
// available to modules

func (m mAS) Logf(s string, args ...interface{}) {
	m.as.Logf(s, args...)
}
func (m mAS) Diagf(s string, args ...interface{}) {
	m.as.Diagf(s, args...)
}

// for ordinary network requests
func (m mAS) NetIOHeavy() (func(), error) {
	m.as.NetReqs++

	if m.as.NetReqs > m.as.cf.NetMax {
		err := fmt.Errorf("Maximum number of web requests exceeded!")
		m.as.vm.Interrupt(err)
		return nil, err
	}

	m.as.pauseTimer()
	t0 := time.Now() // start timing

	return func() {
		// stop timing
		dur := time.Now().Sub(t0)
		m.as.NetTime += dur
		m.as.resumeTimer()
	}, nil
}

// for local (on-net) network requests
func (m mAS) NetIOLight() (func(), error) {
	m.as.LocalReqs++

	m.as.pauseTimer()
	t0 := time.Now() // start timing

	return func() {
		// stop timing
		dur := time.Now().Sub(t0)
		m.as.NetTime += dur
		m.as.resumeTimer()
	}, nil
}

func (m mAS) NetIOErr() {
	m.as.NetErrs++
}

func (m mAS) Fatal(err error) {
	m.as.NetErrs++
	m.as.cf.Logger.Error(err)
	m.as.vm.Interrupt(err)
}
func (m mAS) Error(err error) {
	m.as.NetErrs++
	m.as.cf.Logger.Error(err)
}

func (m mAS) IsDryRun() bool {
	return m.as.cf.NetMock
}

func (m mAS) NetTimeout() time.Duration {
	return m.as.cf.NetTimeout
}

func (m mAS) VM() *goja.Runtime {
	return m.as.vm
}

func (m mAS) TraceInfo() string {
	return m.as.cf.Trace
}

// ################################################################

var scriptRuntime = goja.MustCompile("runtime", `
var web = module('std/web')
`, false)
