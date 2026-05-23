// Copyright 2015-2026 SkunkWerks, GmbH.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Originally based on github.com/astaxie/beego/httplib (Apache-2.0).

package main

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/skunkwerks/gurl/hamac"
)

// Settings groups the per-request knobs.
type Settings struct {
	UserAgent       string
	TLSClientConfig *tls.Config
	Proxy           func(*http.Request) (*url.URL, error)
	Timeout         time.Duration // 0 = no timeout
	DumpBody        bool
	ShowDebug       bool
	Gzip            bool // decode gzip responses
}

// Request is a fluent builder for a single HTTP request.
type Request struct {
	url     string
	method  string
	req     *http.Request
	params  map[string]string
	files   map[string]string
	setting Settings
	resp    *http.Response
	body    []byte
	dump    []byte
}

// NewRequest builds a Request for the given URL and method.
func NewRequest(rawurl, method string) *Request {
	u, err := url.Parse(rawurl)
	if err != nil {
		log.Fatal(err)
	}
	req := &http.Request{
		URL:        u,
		Method:     method,
		Header:     make(http.Header),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	return &Request{
		url:    rawurl,
		method: method,
		req:    req,
		params: map[string]string{},
		files:  map[string]string{},
	}
}

// Setting replaces the per-request settings.
func (r *Request) Setting(s Settings) *Request {
	r.setting = s
	return r
}

// Header sets a single header key/value pair.
func (r *Request) Header(k, v string) *Request {
	r.req.Header.Set(k, v)
	return r
}

// SetHost overrides the request's Host header.
func (r *Request) SetHost(host string) *Request {
	r.req.Host = host
	return r
}

// SetTLSClientConfig swaps in a custom TLS config (e.g. for -insecure).
func (r *Request) SetTLSClientConfig(c *tls.Config) *Request {
	r.setting.TLSClientConfig = c
	return r
}

// SetProxy installs a proxy resolver.
func (r *Request) SetProxy(p func(*http.Request) (*url.URL, error)) *Request {
	r.setting.Proxy = p
	return r
}

// GetRequest returns the underlying *http.Request so callers can use
// stdlib helpers like SetBasicAuth or ProxyFromEnvironment.
func (r *Request) GetRequest() *http.Request {
	return r.req
}

// Body installs a raw request body. Accepts string or []byte.
func (r *Request) Body(data interface{}) *Request {
	switch t := data.(type) {
	case string:
		buf := bytes.NewBufferString(t)
		r.req.Body = io.NopCloser(buf)
		r.req.ContentLength = int64(len(t))
		raw := []byte(t)
		r.req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(raw)), nil
		}
	case []byte:
		buf := bytes.NewBuffer(t)
		r.req.Body = io.NopCloser(buf)
		r.req.ContentLength = int64(len(t))
		raw := append([]byte(nil), t...)
		r.req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(raw)), nil
		}
	}
	return r
}

// JsonBody installs a JSON-encoded body, matching the old guard: only
// applied if no body has yet been set and obj is non-nil.
func (r *Request) JsonBody(obj interface{}) (*Request, error) {
	if r.req.Body == nil && obj != nil {
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		if err := enc.Encode(obj); err != nil {
			return r, err
		}
		raw := append([]byte(nil), buf.Bytes()...)
		r.req.Body = io.NopCloser(bytes.NewReader(raw))
		r.req.ContentLength = int64(len(raw))
		r.req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(raw)), nil
		}
		r.req.Header.Set("Content-Type", "application/json")
	}
	return r, nil
}

// Param adds a query/form parameter.
func (r *Request) Param(k, v string) *Request {
	r.params[k] = v
	return r
}

// PostFile registers a form file upload.
func (r *Request) PostFile(formname, filename string) *Request {
	r.files[formname] = filename
	return r
}

// Params returns a read-only copy of the accumulated parameters.
func (r *Request) Params() map[string]string {
	out := make(map[string]string, len(r.params))
	for k, v := range r.params {
		out[k] = v
	}
	return out
}

// Files returns a read-only copy of the form file uploads.
func (r *Request) Files() map[string]string {
	out := make(map[string]string, len(r.files))
	for k, v := range r.files {
		out[k] = v
	}
	return out
}

// SignBody reads the body fully, signs it with the given HMAC, sets the
// configured header, and re-installs the body so it can still be sent.
func (r *Request) SignBody(mac hamac.Hmac) *Request {
	if r.req.Body == nil {
		return r
	}
	body, _ := io.ReadAll(r.req.Body)
	signature := string(hamac.Sign(mac, body))
	r.req.Header.Set(mac.Header, signature)
	r.req.Body = io.NopCloser(bytes.NewBuffer(body))
	r.req.ContentLength = int64(len(body))
	raw := append([]byte(nil), body...)
	r.req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(raw)), nil
	}
	return r
}

// Debug toggles request dumping.
func (r *Request) Debug(on bool) *Request {
	r.setting.ShowDebug = on
	return r
}

// DumpBody toggles whether the request dump includes the body.
func (r *Request) DumpBody(on bool) *Request {
	r.setting.DumpBody = on
	return r
}

// DumpRequest returns the cached dump captured during SendOut (or an
// empty slice if the request has never been sent).
func (r *Request) DumpRequest() []byte {
	return r.dump
}

// buildBody finalises params/files into the URL query string or
// request body. Mirrors the old buildUrl semantics.
func (r *Request) buildBody(paramBody string) {
	if r.req.Method == "GET" && len(paramBody) > 0 {
		if strings.Contains(r.url, "?") {
			r.url += "&" + paramBody
		} else {
			r.url += "?" + paramBody
		}
		return
	}

	if (r.req.Method == "POST" || r.req.Method == "PUT" || r.req.Method == "PATCH") && r.req.Body == nil {
		// multipart upload
		if len(r.files) > 0 {
			pr, pw := io.Pipe()
			bodyWriter := multipart.NewWriter(pw)
			go func() {
				for formname, filename := range r.files {
					fileWriter, err := bodyWriter.CreateFormFile(formname, filename)
					if err != nil {
						log.Fatal(err)
					}
					fh, err := os.Open(filename)
					if err != nil {
						log.Fatal(err)
					}
					_, err = io.Copy(fileWriter, fh)
					fh.Close()
					if err != nil {
						log.Fatal(err)
					}
				}
				for k, v := range r.params {
					bodyWriter.WriteField(k, v)
				}
				bodyWriter.Close()
				pw.Close()
			}()
			r.Header("Content-Type", bodyWriter.FormDataContentType())
			r.req.Body = io.NopCloser(pr)
			return
		}

		// urlencoded form
		if len(paramBody) > 0 {
			r.Header("Content-Type", "application/x-www-form-urlencoded")
			r.Body(paramBody)
		}
	}
}

// SendOut performs the request once, without memoising the response.
// Used by the bench loop.
func (r *Request) SendOut() (*http.Response, error) {
	var paramBody string
	if len(r.params) > 0 {
		var buf bytes.Buffer
		for k, v := range r.params {
			buf.WriteString(url.QueryEscape(k))
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
			buf.WriteByte('&')
		}
		paramBody = buf.String()
		paramBody = paramBody[:len(paramBody)-1]
	}

	r.buildBody(paramBody)
	u, err := url.Parse(r.url)
	if err != nil {
		return nil, err
	}
	r.req.URL = u

	tlsCfg := r.setting.TLSClientConfig
	if tlsCfg == nil {
		tlsCfg = &tls.Config{}
	}
	if tlsCfg.MinVersion == 0 {
		tlsCfg.MinVersion = tls.VersionTLS12
	}
	tr := &http.Transport{
		TLSClientConfig:   tlsCfg,
		Proxy:             r.setting.Proxy,
		DialContext:       (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2: true,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   r.setting.Timeout,
	}

	if r.setting.UserAgent != "" && r.req.Header.Get("User-Agent") == "" {
		r.req.Header.Set("User-Agent", r.setting.UserAgent)
	}

	if r.setting.ShowDebug {
		dump, err := httputil.DumpRequestOut(r.req, r.setting.DumpBody)
		if err != nil {
			println(err.Error())
		}
		r.dump = dump
	}

	return client.Do(r.req)
}

// Response executes the request once and caches the response.
func (r *Request) Response() (*http.Response, error) {
	if r.resp != nil && r.resp.StatusCode != 0 {
		return r.resp, nil
	}
	resp, err := r.SendOut()
	if err != nil {
		return nil, err
	}
	r.resp = resp
	return resp, nil
}

// Bytes reads and caches the response body, transparently gunzipping
// when setting.Gzip is on and Content-Encoding is gzip.
func (r *Request) Bytes() ([]byte, error) {
	if r.body != nil {
		return r.body, nil
	}
	resp, err := r.Response()
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()
	if r.setting.Gzip && resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		r.body, err = io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		return r.body, nil
	}
	r.body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return r.body, nil
}
