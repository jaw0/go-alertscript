// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Apr-05 18:28 (EDT)
// Function: simple in memory key-value store (for testing)

package modstd

import (
	"encoding/json"

	"github.com/deduce-com/go-alertscript/module"
	"github.com/dop251/goja"
)

var _ = module.Register("std/store", installStore)

type modStore struct {
	as  module.MASer
	bkt string
	kvs map[string][]byte
}

type Result struct {
	Value interface{}
	Found bool
}

func installStore(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	m := &modStore{
		as:  aser,
		bkt: aser.Federation(),
		kvs: make(map[string][]byte),
	}
	return m
}

func (m *modStore) Put(key string, value interface{}) error {

	buf, err := json.Marshal(value)
	if err != nil {
		return err
	}

	m.kvs[key] = buf
	return nil
}

func (m *modStore) Get(key string) (*Result, error) {

	buf, found := m.kvs[key]

	if !found {
		return &Result{}, nil
	}

	var value interface{}
	json.Unmarshal(buf, &value)

	return &Result{value, found}, nil
}
