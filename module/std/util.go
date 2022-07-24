// Copyright (c) 2021
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2021-Jan-14 10:54 (EST)
// Function: useful helkper functions

package modstd

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"hash"

	"github.com/jaw0/go-alertscript/module"
	"github.com/dop251/goja"
)

var _ = module.Register("std/encoding/base64", installBase64)
var _ = module.Register("std/encoding/base32", installBase32)
var _ = module.Register("std/encoding/hex", installHex)
var _ = module.Register("std/crypto/hash", installHash)
var _ = module.Register("std/crypto/hmac", installHmac)

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
	Md5           func(string) []byte `json:"md5"`
	Sha1          func(string) []byte `json:"sha1"`
	Sha256        func(string) []byte `json:"sha256"`
	Sha512        func(string) []byte `json:"sha512"`
	Md5_hex       func(string) string `json:"md5_hex"`
	Md5_base64    func(string) string `json:"md5_base64"`
	Sha1_hex      func(string) string `json:"sha1_hex"`
	Sha1_base64   func(string) string `json:"sha1_base64"`
	Sha256_hex    func(string) string `json:"sha256_hex"`
	Sha256_base64 func(string) string `json:"sha256_base64"`
	Sha512_hex    func(string) string `json:"sha512_hex"`
	Sha512_base64 func(string) string `json:"sha512_base64"`
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
	Md5:           newHash(md5.New),
	Sha1:          newHash(sha1.New),
	Sha256:        newHash(sha256.New),
	Sha512:        newHash(sha512.New),
	Md5_hex:       newEncHash(md5.New, hexEncode),
	Md5_base64:    newEncHash(md5.New, base64Vars.UrlSafeNoPadding.Encode),
	Sha1_hex:      newEncHash(sha1.New, hexEncode),
	Sha1_base64:   newEncHash(sha1.New, base64Vars.UrlSafeNoPadding.Encode),
	Sha256_hex:    newEncHash(sha256.New, hexEncode),
	Sha256_base64: newEncHash(sha256.New, base64Vars.UrlSafeNoPadding.Encode),
	Sha512_hex:    newEncHash(sha512.New, hexEncode),
	Sha512_base64: newEncHash(sha512.New, base64Vars.UrlSafeNoPadding.Encode),
}

var hmacVars = &hmacFuncs{
	Md5:    newHmac(md5.New),
	Sha1:   newHmac(sha1.New),
	Sha256: newHmac(sha256.New),
	Sha512: newHmac(sha512.New),
}

// ################################################################

// install useful helper functions

func installBase32(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	return base32Vars
}

func installBase64(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	return base64Vars
}

func installHex(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	return hexVars
}

func installHash(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	return hashVars
}

func installHmac(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	return hmacVars
}

// ################################################################

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

func newEncHash(hf func() hash.Hash, enc func([]byte) string) func(string) string {
	return func(text string) string {
		h := hf()
		h.Write([]byte(text))
		return enc(h.Sum(nil))
	}
}

func newHmac(hf func() hash.Hash) func(string, string) []byte {
	return func(key, text string) []byte {
		h := hmac.New(hf, []byte(key))
		h.Write([]byte(text))
		return h.Sum(nil)
	}
}
