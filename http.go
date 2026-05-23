// Copyright 2020-2026 SkunkWerks, GmbH.
// Licensed under the Apache License, Version 2.0; see LICENSE for details.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/skunkwerks/gurl/httplib"
)

var defaultSetting = httplib.BeegoHttpSettings{
	ShowDebug:        true,
	UserAgent:        "gurl/" + version,
	ConnectTimeout:   60 * time.Second,
	ReadWriteTimeout: 60 * time.Second,
	Gzip:             true,
	DumpBody:         true,
}

// argKind classifies a CLI argument by the separator that links its key
// and value. See README "Request items".
type argKind int

const (
	argUnknown argKind = iota
	argData            // key=value
	argHeader          // key:value
	argJSON            // key:=value  (raw JSON; value may begin with @ for file)
	argFile            // key@path    (form file upload)
)

// splitArg parses a single CLI argument into (key, value, kind).
//
// The grammar is HTTPie-like:
//
//	key=value     -> data field / query parameter
//	key:value     -> header
//	key:=value    -> raw JSON value (file with leading @)
//	key@path      -> form file upload
//	key=@path     -> data field whose value is read from a file
//	key:=@path    -> raw JSON value read from a file
//
// The rule is "earliest unescaped separator wins"; when two separators
// start at the same offset the longer one wins (so `:=` beats `:`).
// This avoids miscategorising values that legitimately contain separator
// characters - e.g. `url=http://example.com` (an `=` data field whose
// value happens to contain `:`) used to be parsed as a header.
//
// A backslash before a separator character (`=`, `:`, `@`, `\`) escapes
// it: that byte is taken literally and the backslash is stripped from
// the emitted key/value. A backslash before any other byte is kept
// verbatim, so values like `path=C:\Users\me` survive untouched. A
// doubled backslash always collapses to a single literal backslash.
//
// fromFile is true when the value started with an unescaped `@`, in
// which case it has been stripped and the remaining bytes name a file
// to read. For argFile the leading `@` is the separator itself, so
// fromFile is always true. An escaped leading `\@` yields fromFile=false
// and the literal `@` is kept in the value.
func splitArg(s string) (key, value string, kind argKind, fromFile bool) {
	sepIdx, sepLen, k := findSeparator(s)
	if sepIdx < 0 {
		return unescape(s), "", argUnknown, false
	}
	rawKey := s[:sepIdx]
	rawVal := s[sepIdx+sepLen:]

	// For argFile the separator itself is the `@`, so by definition the
	// remainder is a file path. For argData/argJSON, an unescaped leading
	// `@` in the value means "read this from a file".
	switch k {
	case argFile:
		return unescape(rawKey), unescape(rawVal), k, true
	case argData, argJSON:
		if strings.HasPrefix(rawVal, "@") {
			return unescape(rawKey), unescape(rawVal[1:]), k, true
		}
	}
	return unescape(rawKey), unescape(rawVal), k, false
}

// findSeparator scans s left-to-right and returns the byte offset, length
// and kind of the chosen separator, or -1 if none is found. Backslash
// escapes are honored: an escape sequence is skipped wholesale (two
// bytes) and the separator character it shields is invisible to the
// scan.
//
// At a given position the longer of the candidate separators wins, so
// `:=` is preferred over `:` when both start at the same index.
func findSeparator(s string) (idx, length int, k argKind) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			// Only separator-class bytes participate in escaping; for
			// any other follower the backslash is literal and we keep
			// scanning from the next byte. This means `\\` (literal
			// backslash) is just "skip both bytes".
			n := s[i+1]
			if isSeparatorByte(n) {
				i++ // skip the escaped byte; outer i++ advances past it
				continue
			}
			// Non-separator follower: backslash is literal; continue
			// scanning at the next byte (i.e. don't skip it).
			continue
		}
		switch c {
		case ':':
			if i+1 < len(s) && s[i+1] == '=' {
				return i, 2, argJSON
			}
			return i, 1, argHeader
		case '=':
			return i, 1, argData
		case '@':
			return i, 1, argFile
		}
	}
	return -1, 0, argUnknown
}

func isSeparatorByte(b byte) bool {
	return b == '=' || b == ':' || b == '@' || b == '\\'
}

// unescape removes backslash escapes from a parsed key or value. Only
// separator-class bytes are recognised as escapable; a backslash before
// anything else is kept literally so non-separator content (e.g.
// Windows-style paths) round-trips unchanged.
func unescape(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) && isSeparatorByte(s[i+1]) {
			b.WriteByte(s[i+1])
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func getHTTP(method string, url string, args []string) (r *httplib.BeegoHttpRequest) {
	r = httplib.NewBeegoRequest(url, method)
	r.Setting(defaultSetting)
	r.Header("Accept-Encoding", "gzip, deflate")
	if *isjson {
		r.Header("Accept", "application/json")
		r.Header("Content-Type", "application/json")
	} else if form || method == "GET" {
		r.Header("Accept", "*/*")
	} else {
		r.Header("Accept", "application/json")
	}
	for i := range args {
		key, val, kind, fromFile := splitArg(args[i])
		switch kind {
		case argJSON: // key:=value (raw JSON)
			if fromFile {
				content, err := os.ReadFile(val)
				if err != nil {
					log.Fatalf("read JSON file %q: %v", val, err)
				}
				var j interface{}
				if err := json.Unmarshal(content, &j); err != nil {
					log.Fatalf("unmarshal JSON file %q: %v", val, err)
				}
				jsonmap[key] = j
				continue
			}
			if !json.Valid([]byte(val)) {
				log.Fatalf("invalid JSON for field %q: %s", key, val)
			}
			jsonmap[key] = json.RawMessage(val)
		case argData: // key=value (data field or query param)
			if fromFile {
				content, err := os.ReadFile(val)
				if err != nil {
					log.Fatalf("read data file %q: %v", val, err)
				}
				val = string(content)
			}
			if form || method == "GET" {
				r.Param(key, val)
			} else {
				jsonmap[key] = val
			}
		case argHeader: // key:value
			if key == "Host" {
				r.SetHost(val)
			}
			r.Header(key, val)
		case argFile: // key@path (form file)
			if *isjson {
				log.Fatalf("file upload not supported with JSON: %s", args[i])
			}
			if !form {
				log.Fatalf("file upload only supported in forms style: -f=true (got %q)", args[i])
			}
			r.PostFile(key, val)
		default:
			log.Fatalf("could not parse argument %q (expected key=value, key:value, key:=json, or key@file)", args[i])
		}
	}
	if !form && len(jsonmap) > 0 {
		_, err := r.JsonBody(jsonmap)
		if err != nil {
			log.Fatal("fail to marshal JSON: ", err)
		}
	}
	return
}

func formatResponseBody(res *http.Response, httpreq *httplib.BeegoHttpRequest, pretty bool) string {
	body, err := httpreq.Bytes()
	if err != nil {
		log.Fatalln("can't get the url", err)
	}
	fmt.Println("")
	match, err := regexp.MatchString(contentJsonRegex, res.Header.Get("Content-Type"))
	if err != nil {
		log.Fatalln("failed to compile regex", err)
	}
	if pretty && match {
		var output bytes.Buffer
		err := json.Indent(&output, body, "", "  ")
		if err != nil {
			log.Fatal("Response JSON Indent: ", err)
		}

		return output.String()
	}

	return string(body)
}
