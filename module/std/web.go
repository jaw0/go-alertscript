// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Jan-26 16:46 (EST)
// Function: make web requests

package modstd

import (
	"bytes"
	"crypto/tls"
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/jaw0/go-alertscript/module"
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
	tls     *tls.ConnectionState
}

// QQQ - provide more / less?
type TLSResult struct {
	Version            uint16 `json:"version"`
	CipherSuite        uint16 `json:"cipher_suite"`
	NegotiatedProtocol string `json:"negotiated_protocol"`
	ServerName         string `json:"server_name"`
	Certs              int    `json:"n_certs"`
}

type TLSCert struct {
	Signature             []byte     `json:"signature"`
	SignatureAlgorithm    string     `json:"signature_alg"`
	PublicKeyAlgorithm    string     `json:"signature_alg"`
	Version               int        `json:"version"`
	SerialNumber          string     `json:"serial_number"`
	Issuer                pkix.Name  `json:"issuer"`
	Subject               pkix.Name  `json:"subject"`
	NotBefore             int64      `json:"not_before"` // converted to js units
	NotAfter              int64      `json:"not_after"`  // converted to js units
	KeyUsage              int32      `json:"key_usage"`
	SubjectKeyId          []byte     `json:"subject_key_id"`
	AuthorityKeyId        []byte     `json:"authority_key_id"`
	DNSNames              []string   `json:"dns_names"`
	EmailAddresses        []string   `json:"email_addresses"`
	IPAddresses           []net.IP   `json:"ip_addresses"`
	URIs                  []*url.URL `json:"uris"`
	CRLDistributionPoints []string   `json:"crl_distribution_points"`
}

func installWeb(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	wg, _ := vm.RunProgram(webGet)
	wp, _ := vm.RunProgram(webPost)
	wj, _ := vm.RunProgram(webPostJson)
	wu, _ := vm.RunProgram(webPostUrlEnc)
	m := &modWeb{aser, wg, wp, wj, wu}
	return m
}

func NewWeb(aser module.MASer) *modWeb {
	return &modWeb{aser, nil, nil, nil, nil}
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
		return &WebResult{Code: 200, Message: "not tried"}, nil
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
		return &WebResult{Code: 500, Message: "Request Failed", Body: err.Error()}, nil
	}

	if resp.Status[0] != '2' {
		m.as.NetIOErr()
		m.as.Logf("Request Failed: %s", resp.Status)
	} else {
		m.as.Diagf("%v", resp.Status)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	ret := &WebResult{Code: resp.StatusCode, Message: resp.Status[4:], Body: string(body), Header: resp.Header, tls: resp.TLS}

	return ret, nil

}

// get the tls details from the result
func (wr *WebResult) Tls() *TLSResult {
	tls := wr.tls

	if tls == nil {
		return nil
	}
	return &TLSResult{
		Version:            tls.Version,
		CipherSuite:        tls.CipherSuite,
		NegotiatedProtocol: tls.NegotiatedProtocol,
		ServerName:         tls.ServerName,
		Certs:              len(tls.PeerCertificates),
	}

}

// get the tls cert details from the result
func (wr *WebResult) Cert(n int) *TLSCert {
	tls := wr.tls

	if tls == nil || n >= len(tls.PeerCertificates) {
		return nil
	}

	c := tls.PeerCertificates[n]

	return &TLSCert{
		Signature:             c.Signature,
		SignatureAlgorithm:    c.SignatureAlgorithm.String(),
		PublicKeyAlgorithm:    c.PublicKeyAlgorithm.String(),
		Version:               c.Version,
		SerialNumber:          c.SerialNumber.String(),
		Issuer:                c.Issuer,
		Subject:               c.Subject,
		KeyUsage:              int32(c.KeyUsage),
		SubjectKeyId:          c.SubjectKeyId,
		AuthorityKeyId:        c.AuthorityKeyId,
		DNSNames:              c.DNSNames,
		EmailAddresses:        c.EmailAddresses,
		IPAddresses:           c.IPAddresses,
		URIs:                  c.URIs,
		CRLDistributionPoints: c.CRLDistributionPoints,
		NotBefore:             c.NotBefore.UTC().UnixNano() / 1e6,
		NotAfter:              c.NotAfter.UTC().UnixNano() / 1e6,
	}
	return nil
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
