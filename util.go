// Copyright (c) 2021
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2021-Jan-14 10:54 (EST)
// Function: useful helkper functions

package alertscript

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	//"fmt"
	"hash"

	"github.com/dop251/goja"
)

type encodeDecode struct {
	Encode func([]byte) string `json:"encode"`
	Decode func(string) []byte `json:"decode"`
}

type base64Funcs struct {
	Std              encodeDecode `json:"std"`
	UrlSafe          encodeDecode `json:"urlsafe"`
	StdNoPadding     encodeDecode `json:"std_nopadding"`
	UrlSafeNoPadding encodeDecode `json:"urlsafe_nopadding"`
}
type base32Funcs struct {
	Std          encodeDecode `json:"std"`
	Hex          encodeDecode `json:"hex"`
	StdNoPadding encodeDecode `json:"std_nopadding"`
	HexNoPadding encodeDecode `json:"hex_nopadding"`
}

type hashFuncs struct {
	Md5    func(string) []byte `json:"md5"`
	Sha1   func(string) []byte `json:"sha1"`
	Sha256 func(string) []byte `json:"sha256"`
	Sha512 func(string) []byte `json:"sha512"`
}
type hmacFuncs struct {
	Md5    func(string, string) []byte `json:"md5"`
	Sha1   func(string, string) []byte `json:"sha1"`
	Sha256 func(string, string) []byte `json:"sha256"`
	Sha512 func(string, string) []byte `json:"sha512"`
}

var base64Vars = &base64Funcs{
	Std:              newEncodeDecode(base64.StdEncoding),
	StdNoPadding:     newEncodeDecode(base64.RawStdEncoding),
	UrlSafe:          newEncodeDecode(base64.URLEncoding),
	UrlSafeNoPadding: newEncodeDecode(base64.RawURLEncoding),
}

var base32Vars = &base32Funcs{
	Std:          newEncodeDecode(base32.StdEncoding),
	Hex:          newEncodeDecode(base32.HexEncoding),
	StdNoPadding: newEncodeDecode(base32.StdEncoding.WithPadding(base32.NoPadding)),
	HexNoPadding: newEncodeDecode(base32.HexEncoding.WithPadding(base32.NoPadding)),
}

var hexVars = &encodeDecode{
	Encode: hexEncode,
	Decode: hexDecode,
}

var hashVars = &hashFuncs{
	Md5:    newHash(md5.New),
	Sha1:   newHash(sha1.New),
	Sha256: newHash(sha256.New),
	Sha512: newHash(sha512.New),
}

var hmacVars = &hmacFuncs{
	Md5:    newHmac(md5.New),
	Sha1:   newHmac(sha1.New),
	Sha256: newHmac(sha256.New),
	Sha512: newHmac(sha512.New),
}

// ################################################################

// install useful helper functions
func installUtilFuncs(vm *goja.Runtime) {

	vm.Set("base64", base64Vars)
	vm.Set("base32", base32Vars)
	vm.Set("hex", hexVars)
	vm.Set("hash", hashVars)
	vm.Set("hmac", hmacVars)
}

func hexEncode(s []byte) string {
	return hex.EncodeToString(s)
}

func hexDecode(s string) []byte {
	d, _ := hex.DecodeString(s)
	return d
}

type encoder interface {
	DecodeString(s string) ([]byte, error)
	EncodeToString(src []byte) string
}

func newEncodeDecode(enc encoder) encodeDecode {

	return encodeDecode{
		Encode: func(s []byte) string {
			e := enc.EncodeToString(s)
			return string(e)
		},
		Decode: func(s string) []byte {
			d, _ := enc.DecodeString(s)
			return d
		},
	}
}

func newHash(hf func() hash.Hash) func(string) []byte {
	return func(text string) []byte {
		h := hf()
		h.Write([]byte(text))
		return h.Sum(nil)
	}
}

func newHmac(hf func() hash.Hash) func(string, string) []byte {
	return func(key, text string) []byte {
		h := hmac.New(hf, []byte(key))
		h.Write([]byte(text))
		return h.Sum(nil)
	}
}
