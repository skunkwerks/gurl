// Copyright 2015-2026 SkunkWerks, GmbH.
// Licensed under the Apache License, Version 2.0; see LICENSE for details.

package main

import (
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skunkwerks/gurl/hamac"
)

func TestRequestGETWithParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
	}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/", "GET").Setting(Settings{})
	req.Param("a", "1").Param("b", "two words")
	if _, err := req.Response(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "a=1") || !strings.Contains(gotQuery, "b=two+words") {
		t.Fatalf("query = %q, want a=1 and b=two+words", gotQuery)
	}
}

func TestRequestPOSTJSON(t *testing.T) {
	var gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/", "POST").Setting(Settings{})
	if _, err := req.JsonBody(map[string]interface{}{"x": 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := req.Response(); err != nil {
		t.Fatal(err)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if parsed["x"].(float64) != 1 {
		t.Errorf("body x = %v, want 1", parsed["x"])
	}
}

func TestRequestPOSTForm(t *testing.T) {
	var gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/", "POST").Setting(Settings{})
	req.Param("name", "John Doe").Param("age", "30")
	if _, err := req.Response(); err != nil {
		t.Fatal(err)
	}
	if gotCT != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", gotCT)
	}
	bodyStr := string(gotBody)
	if !strings.Contains(bodyStr, "name=John+Doe") || !strings.Contains(bodyStr, "age=30") {
		t.Errorf("body = %q", bodyStr)
	}
}

func TestRequestPOSTMultipart(t *testing.T) {
	tmpDir := t.TempDir()
	upPath := filepath.Join(tmpDir, "hello.txt")
	if err := os.WriteFile(upPath, []byte("hello multipart"), 0644); err != nil {
		t.Fatal(err)
	}

	var gotCT string
	var gotFileContents string
	var gotFileName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		_, params, err := mime.ParseMediaType(gotCT)
		if err != nil {
			t.Errorf("ParseMediaType: %v", err)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("NextPart: %v", err)
				return
			}
			b, _ := io.ReadAll(p)
			if p.FormName() == "upload" {
				gotFileContents = string(b)
				gotFileName = p.FileName()
			}
		}
	}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/", "POST").Setting(Settings{})
	req.PostFile("upload", upPath)
	if _, err := req.Response(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotCT, "multipart/form-data;") {
		t.Errorf("Content-Type = %q", gotCT)
	}
	if gotFileContents != "hello multipart" {
		t.Errorf("file contents = %q", gotFileContents)
	}
	if gotFileName != "hello.txt" {
		t.Errorf("file name = %q, want hello.txt", gotFileName)
	}
}

func TestRequestCustomHost(t *testing.T) {
	var gotHost string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
	}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/", "GET").Setting(Settings{}).SetHost("example.invalid")
	if _, err := req.Response(); err != nil {
		t.Fatal(err)
	}
	if gotHost != "example.invalid" {
		t.Errorf("Host = %q, want example.invalid", gotHost)
	}
}

func TestRequestBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "alice" || p != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/", "GET").Setting(Settings{})
	req.GetRequest().SetBasicAuth("alice", "secret")
	res, err := req.Response()
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
}

func TestRequestGzipDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		gz.Write([]byte("zipped content"))
		gz.Close()
	}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/", "GET").Setting(Settings{Gzip: true})
	body, err := req.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "zipped content" {
		t.Errorf("body = %q", body)
	}
}

func TestRequestDumpRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/some/path", "GET").Setting(Settings{ShowDebug: true})
	if _, err := req.Response(); err != nil {
		t.Fatal(err)
	}
	dump := req.DumpRequest()
	if len(dump) == 0 {
		t.Fatal("dump is empty")
	}
	s := string(dump)
	if !strings.Contains(s, "GET ") {
		t.Errorf("dump missing method: %q", s)
	}
	if !strings.Contains(s, "/some/path") {
		t.Errorf("dump missing path: %q", s)
	}
}

func TestRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	req := NewRequest(srv.URL+"/", "GET").Setting(Settings{Timeout: 20 * time.Millisecond})
	if _, err := req.Response(); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestRequestTLSMinVersion(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	// Force a max version below 1.2 client-side; the default MinVersion
	// in SendOut should still be set to 1.2, which is impossible with
	// the cap, so the handshake must fail.
	req := NewRequest(srv.URL+"/", "GET").Setting(Settings{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MaxVersion:         tls.VersionTLS11,
		},
	})
	if _, err := req.Response(); err == nil {
		t.Fatal("expected TLS handshake error with MaxVersion < MinVersion")
	}
}

func TestRequestSignBody(t *testing.T) {
	var gotBody []byte
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotSig = r.Header.Get("x-test-sig")
	}))
	defer srv.Close()

	mac := hamac.New("sha256:x-test-sig:topsecret")
	if !mac.Enabled {
		t.Fatal("hmac not enabled")
	}

	req := NewRequest(srv.URL+"/", "POST").Setting(Settings{})
	req.Body("payload bytes")
	req.SignBody(mac)
	if _, err := req.Response(); err != nil {
		t.Fatal(err)
	}
	if string(gotBody) != "payload bytes" {
		t.Errorf("body = %q, want %q", gotBody, "payload bytes")
	}
	if gotSig == "" {
		t.Error("missing signature header")
	}
	if !strings.HasPrefix(gotSig, "sha256=") {
		t.Errorf("signature = %q, want sha256= prefix", gotSig)
	}
}
