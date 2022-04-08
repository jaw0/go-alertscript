// Copyright (c) 2022
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2022-Feb-13 17:08 (EST)
// Function: s3 (+ compat) put/get

package mods3

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/deduce-com/go-alertscript/module"
	"github.com/dop251/goja"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const s3Host = "s3.amazonaws.com"

var _ = module.Register("ext/s3", install)

type mod struct {
	as module.MASer
}

func install(aser module.MASer, vm *goja.Runtime, args []interface{}) interface{} {
	m := &mod{aser}
	return m
}

type Creds struct {
	Hostname  string `json:"hostname"` // optional
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
}

type PutOpts struct {
	Metadata         map[string]string `json:"metadata"`
	Tags             map[string]string `json:"tags"`
	ContentType      string            `json:"content_type"`
	Encoding         string            `json:"encoding"`
	Disposition      string            `json:"disposition"`
	Language         string            `json:"language"`
	CacheControl     string            `json:"cache_control"`
	RedirectLocation string            `json:"redirect_location"`
	RetainUntil      time.Time         `json:"retain_until"`
}

type Result struct {
	Content     string
	ETag        string              `json:"etag"`
	ContentType string              `json:"content_type"` // A standard MIME type describing the format of the object data.
	Header      map[string][]string `json:"header"`
	Metadata    map[string]string   `json:"metadata"`
	Tags        map[string]string   `json:"tags"`
	Version     string              `json:"version"`
}

func (m *mod) Put(creds *Creds, bucket string, key string, data string, opts *PutOpts) (*Result, error) {

	if creds == nil {
		return nil, fmt.Errorf("s3.put where?")
	}

	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return nil, err
	}

	// for debugging
	m.as.Diagf("s3/put bucket %s, key %s", bucket, key)
	if m.as.IsDryRun() {
		return &Result{ETag: "1", Version: "dry-run"}, nil
	}

	if creds.Hostname == "" {
		creds.Hostname = s3Host
	}

	client, err := minio.New(creds.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKey, creds.SecretKey, ""),
		Secure: true,
		Region: creds.Region,
	})

	if err != nil {
		return nil, fmt.Errorf("s3 client failed: %v", err)
	}

	//client.SetAppInfo("myCloudApp", "1.0.0")

	ctx, _ := context.WithTimeout(context.Background(), m.as.NetTimeout())
	b := bytes.NewBufferString(data)

	putOpts := minio.PutObjectOptions{
		ContentType:             opts.ContentType,
		ContentEncoding:         opts.Encoding,
		ContentDisposition:      opts.Disposition,
		ContentLanguage:         opts.Language,
		CacheControl:            opts.CacheControl,
		UserMetadata:            opts.Metadata,
		UserTags:                opts.Tags,
		WebsiteRedirectLocation: opts.RedirectLocation,
		RetainUntilDate:         opts.RetainUntil,
	}

	info, err := client.PutObject(ctx, bucket, key, b, int64(len(data)), putOpts)

	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("s3 error %v", err)
		return nil, fmt.Errorf("s3.put failed: %v", err)
	}
	return &Result{
		ETag:    info.ETag,
		Version: info.VersionID,
	}, nil

}

func (m *mod) Get(creds *Creds, bucket string, key string) (*Result, error) {

	if creds == nil {
		return nil, fmt.Errorf("s3.put where?")
	}

	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return nil, err
	}

	// for debugging
	m.as.Diagf("s3/get bucket %s, key %s", bucket, key)
	if m.as.IsDryRun() {
		return &Result{}, nil
	}

	if creds.Hostname == "" {
		creds.Hostname = s3Host
	}

	client, err := minio.New(creds.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKey, creds.SecretKey, ""),
		Secure: true,
		Region: creds.Region,
	})

	if err != nil {
		return nil, fmt.Errorf("s3 client failed: %v", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), m.as.NetTimeout())

	obj, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})

	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("s3 error %v", err)
		return nil, fmt.Errorf("s3.get failed: %v", err)
	}

	info, _ := obj.Stat()
	body, _ := ioutil.ReadAll(obj)
	obj.Close()

	return &Result{
		Content:     string(body),
		Version:     info.VersionID,
		ETag:        info.ETag,
		ContentType: info.ContentType,
		Header:      info.Metadata,
		Metadata:    info.UserMetadata,
		Tags:        info.UserTags,
	}, nil

}

func (m *mod) Remove(creds *Creds, bucket string, key string, version string) (*Result, error) {

	if creds == nil {
		return nil, fmt.Errorf("s3.put where?")
	}

	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return nil, err
	}

	// for debugging
	m.as.Diagf("s3/put bucket %s, key %s", bucket, key)
	if m.as.IsDryRun() {
		return &Result{ETag: "1", Version: "dry-run"}, nil
	}

	if creds.Hostname == "" {
		creds.Hostname = s3Host
	}

	client, err := minio.New(creds.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKey, creds.SecretKey, ""),
		Secure: true,
		Region: creds.Region,
	})

	if err != nil {
		return nil, fmt.Errorf("s3 client failed: %v", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), m.as.NetTimeout())

	err = client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{
		VersionID: version},
	)

	if err != nil {
		m.as.NetIOErr()
		m.as.Logf("s3 error %v", err)
		return nil, fmt.Errorf("s3.put failed: %v", err)
	}
	return &Result{}, nil

}

func (m *mod) Newbucket(creds *Creds, bucket string) error {

	if creds == nil {
		return fmt.Errorf("s3.put where?")
	}

	closer, err := m.as.NetIOHeavy()
	if closer != nil {
		defer closer()
	}
	if err != nil {
		m.as.Fatal(err)
		return err
	}

	// for debugging
	m.as.Diagf("s3/new bucket %s", bucket)
	if m.as.IsDryRun() {
		return nil
	}

	if creds.Hostname == "" {
		creds.Hostname = s3Host
	}

	client, err := minio.New(creds.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKey, creds.SecretKey, ""),
		Secure: true,
		Region: creds.Region,
	})

	if err != nil {
		return fmt.Errorf("s3 client failed: %v", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), m.as.NetTimeout())
	err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: creds.Region})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := client.BucketExists(ctx, bucket)
		if errBucketExists == nil && exists {
			return nil
			m.as.Diagf("s3/new bucket %s - already exists", bucket)
		} else {
			return fmt.Errorf("s3/newbucket failed: %v", err)
		}
	}

	m.as.Diagf("s3/new bucket %s - created", bucket)
	return nil
}
