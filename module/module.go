// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jan-26 16:18 (EST)
// Function: modules

package module

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
)

type MASer interface {
	VM() *goja.Runtime
	NetIOHeavy() (func(), error)
	NetIOLight() (func(), error)
	NetIOErr()
	IsDryRun() bool
	NetTimeout() time.Duration
	Logf(string, ...interface{})
	Diagf(string, ...interface{})
	Error(error)
	Fatal(error)
	TraceInfo() string
}

type Installer func(MASer, *goja.Runtime, []interface{}) interface{}

var registry = make(map[string]Installer)

// called at init from the go modules
func Register(mod string, f Installer) bool {
	if _, ok := registry[mod]; ok {
		panic("duplicate module name")
	}
	registry[mod] = f
	return true
}

// add the module() function to js
func InstallModule(as MASer, vm *goja.Runtime) {
	vm.Set("module", func(name string, args ...interface{}) interface{} { return jsModule(as, name, vm, args) })
}

// in js code:
//  var foo = module("foo") or module("foo", args...)
func jsModule(as MASer, name string, vm *goja.Runtime, args []interface{}) interface{} {
	f := registry[name]

	if f == nil {
		as.VM().Interrupt(fmt.Errorf("module not fount: '%s'", name))
	}

	args = append([]interface{}{name}, args...) // add name to args
	return f(as, vm, args)
}
