// Copyright 2014 beego Author. All Rights Reserved.
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

package httplib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Test server setup
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()
	
	// GET handlers
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"args": map[string]string{},
			"headers": r.Header,
			"origin": "127.0.0.1",
			"url": "http://localhost/get",
		})
	})

	mux.HandleFunc("/ip", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"origin": "127.0.0.1",
		})
	})

	// POST handler
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"args": map[string]string{},
			"data": "",
			"files": map[string]string{},
			"form": r.PostForm,
			"headers": r.Header,
			"json": nil,
			"origin": "127.0.0.1",
			"url": "http://localhost/post",
		})
	})

	// PUT handler
	mux.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"args": map[string]string{},
			"data": "",
			"files": map[string]string{},
			"form": map[string]string{},
			"headers": r.Header,
			"json": nil,
			"origin": "127.0.0.1",
			"url": "http://localhost/put",
		})
	})

	// DELETE handler
	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"args": map[string]string{},
			"data": "",
			"files": map[string]string{},
			"form": map[string]string{},
			"headers": r.Header,
			"json": nil,
			"origin": "127.0.0.1",
			"url": "http://localhost/delete",
		})
	})

	// Basic auth handler
	mux.HandleFunc("/basic-auth/", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "passwd" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"user": user,
		})
	})

	// Headers handler
	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"headers": r.Header,
		})
	})

	// Cookie handlers
	mux.HandleFunc("/cookies/set", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "k1",
			Value: r.URL.Query().Get("k1"),
		})
		http.Redirect(w, r, "/cookies", http.StatusFound)
	})

	mux.HandleFunc("/cookies", func(w http.ResponseWriter, r *http.Request) {
		cookies := map[string]string{}
		for _, cookie := range r.Cookies() {
			cookies[cookie.Name] = cookie.Value
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"cookies": cookies,
		})
	})

	return httptest.NewServer(mux)
}

var ts *httptest.Server

func TestMain(m *testing.M) {
	// Setup
	ts = setupTestServer()
	defer ts.Close()
	
	// Run tests
	code := m.Run()
	
	// Exit
	os.Exit(code)
}

func TestResponse(t *testing.T) {
	req := Get(ts.URL + "/get")
	resp, err := req.Response()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}

func TestGet(t *testing.T) {
	req := Get(ts.URL + "/get")
	b, err := req.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(b)

	s, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(s)

	if string(b) != s {
		t.Fatal("request data not match")
	}
}

func TestSimplePost(t *testing.T) {
	v := "smallfish"
	req := Post(ts.URL + "/post")
	req.Param("username", v)

	str, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in post")
	}
}

//func TestPostFile(t *testing.T) {
//	v := "smallfish"
//	req := Post("http://httpbin.org/post")
//	req.Debug(true)
//	req.Param("username", v)
//	req.PostFile("uploadfile", "httplib_test.go")

//	str, err := req.String()
//	if err != nil {
//		t.Fatal(err)
//	}
//	t.Log(str)

//	n := strings.Index(str, v)
//	if n == -1 {
//		t.Fatal(v + " not found in post")
//	}
//}

func TestSimplePut(t *testing.T) {
	str, err := Put(ts.URL + "/put").String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
}

func TestSimpleDelete(t *testing.T) {
	str, err := Delete(ts.URL + "/delete").String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
}

func TestWithCookie(t *testing.T) {
	v := "smallfish"
	str, err := Get(ts.URL + "/cookies/set?k1=" + v).SetEnableCookie(true).String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	str, err = Get(ts.URL + "/cookies").SetEnableCookie(true).String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in cookie")
	}
}

func TestWithBasicAuth(t *testing.T) {
	str, err := Get(ts.URL + "/basic-auth/user/passwd").SetBasicAuth("user", "passwd").String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
	n := strings.Index(str, "authenticated")
	if n == -1 {
		t.Fatal("authenticated not found in response")
	}
}

func TestWithUserAgent(t *testing.T) {
	v := "beego"
	str, err := Get(ts.URL + "/headers").SetUserAgent(v).String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in user-agent")
	}
}

func TestWithSetting(t *testing.T) {
	v := "beego"
	var setting BeegoHttpSettings
	setting.EnableCookie = true
	setting.UserAgent = v
	setting.Transport = nil
	SetDefaultSetting(setting)

	str, err := Get(ts.URL + "/get").String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in user-agent")
	}
}

func TestToJson(t *testing.T) {
	req := Get(ts.URL + "/ip")
	resp, err := req.Response()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)

	// httpbin will return http remote addr
	type Ip struct {
		Origin string `json:"origin"`
	}
	var ip Ip
	err = req.ToJson(&ip)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ip.Origin)

	if n := strings.Count(ip.Origin, "."); n != 3 {
		t.Fatal("response is not valid ip")
	}
}

func TestToFile(t *testing.T) {
	f := "beego_testfile"
	req := Get(ts.URL + "/ip")
	err := req.ToFile(f)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f)
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Index(string(b), "origin"); n == -1 {
		t.Fatal("response does not contain 'origin' field")
	}
}

func TestHeader(t *testing.T) {
	req := Get(ts.URL + "/headers")
	req.Header("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.57 Safari/537.36")
	str, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
}
