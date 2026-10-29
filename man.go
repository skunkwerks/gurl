// Copyright 2025-2026 SkunkWerks, GmbH.
// Licensed under the Apache License, Version 2.0; see LICENSE for details.

package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

// manOption describes one command-line option for the manual page.
//
// The short and long spellings are recorded together so the synopsis and
// the OPTIONS list stay accurate. names holds the flag names that map to
// this option (e.g. "pretty" and "p"); the parser test walks the real
// flag.FlagSet and asserts this table covers, and only covers, exactly the
// registered flags. That is what keeps the page honest as flags change.
type manOption struct {
	shorts []string // e.g. "p"
	longs  []string // e.g. "pretty"
	arg    string   // argument placeholder, empty for boolean flags
	def    string   // default value, empty if none
	desc   string   // human description (must not end with a period)
}

// manOptions lists every flag the program accepts, grouped by logical
// option. Order here is the order they appear in the OPTIONS section.
//
// NOTE: keep this list and gurl.go's init() in lock-step. The test
// man_test.go fails if they drift.
var manOptions = []manOption{
	{shorts: []string{"4"}, desc: "Resolve the URL hostname to IPv4 only"},
	{shorts: []string{"6"}, desc: "Resolve the URL hostname to IPv6 only"},
	{shorts: []string{"a"}, longs: []string{"auth"}, arg: "USER[:PASS]", desc: "HTTP basic authentication username and password"},
	{shorts: []string{"b"}, longs: []string{"bench"}, desc: "Run a benchmark against the URL"},
	{longs: []string{"b.C"}, arg: "N", def: "100", desc: "Number of concurrent benchmark requests"},
	{longs: []string{"b.N"}, arg: "N", def: "1000", desc: "Number of benchmark requests to run"},
	{longs: []string{"body"}, arg: "DATA", desc: "Send DATA verbatim as the request body"},
	{shorts: []string{"d"}, longs: []string{"download"}, desc: "Download the response body to a file"},
	{shorts: []string{"f"}, longs: []string{"form"}, desc: "Serialize request items as a form instead of JSON"},
	{longs: []string{"hmac"}, arg: "ENV", desc: "Name of an environment variable holding the HMAC signing details"},
	{shorts: []string{"i"}, longs: []string{"insecure"}, desc: "Allow connections to hosts with invalid TLS certificates"},
	{shorts: []string{"j"}, longs: []string{"json"}, arg: "BOOL", def: "true", desc: "Send request data as a JSON object"},
	{longs: []string{"method"}, arg: "METHOD", def: "GET", desc: "HTTP method to use"},
	{shorts: []string{"p"}, longs: []string{"pretty"}, arg: "BOOL", def: "true", desc: "Print JSON responses indented"},
	{longs: []string{"print"}, arg: "STRING", def: "A", desc: "Select which parts of the request and response to print"},
	{longs: []string{"proxy"}, arg: "URL", desc: "Proxy to send requests through"},
	{longs: []string{"url"}, arg: "URL", desc: "HTTP request URL (alternative to the positional argument)"},
	{shorts: []string{"v"}, longs: []string{"version"}, desc: "Print the version number and exit"},
}

// escapeMacroTraps escapes a dot that would otherwise begin one of the
// legacy single-letter font macros (.N .C .B .I .R .P) that mandoc still
// honours for historical reasons. Without this, a flag like "b.C" is broken
// mid-token into ".Fl b" plus a stray ".C" and the -tag column mis-reads.
func escapeMacroTraps(s string) string {
	return strings.ReplaceAll(s, ".", "\\&.")
}

// manFlagNames returns the sorted set of flag names the manual page claims
// to document. Used by the test to reconcile against flag.CommandLine.
func manFlagNames() []string {
	var names []string
	for _, o := range manOptions {
		names = append(names, o.shorts...)
		names = append(names, o.longs...)
	}
	sort.Strings(names)
	return names
}

// writeMan renders the mdoc(7) manual page to w. It is the single source
// of truth for what the shipped gurl.1 contains; the Makefile pipes the
// built binary's own output to produce it, so the installed page and the
// running program can never disagree about flags or defaults.
func writeMan(w io.Writer) {
	// Prologue. The date is generated at build time so a freshly built page
	// always reflects when it was last produced.
	fmt.Fprintf(w, ".Dd %s\n", time.Now().Format("January 2, 2006"))
	fmt.Fprintln(w, ".Dt GURL 1")
	fmt.Fprintln(w, ".Os")
	fmt.Fprintln(w, ".Sh NAME")

	// NAME / Nd on a single physical line so makewhatis(8) indexes it.
	fmt.Fprintln(w, ".Nm gurl")
	fmt.Fprintln(w, ".Nd a human-friendly HTTP command-line client")

	fmt.Fprintln(w, ".Sh SYNOPSIS")
	fmt.Fprintln(w, ".Nm")
	fmt.Fprintln(w, ".Op Fl 4 | Fl 6")
	fmt.Fprintln(w, ".Op Fl method Ar method")
	fmt.Fprintln(w, ".Ar URL")
	fmt.Fprintln(w, ".Op Ar item ...")

	fmt.Fprintln(w, ".Sh DESCRIPTION")
	fmt.Fprintln(w, "The")
	fmt.Fprintln(w, ".Nm")
	fmt.Fprintln(w, "utility is a cURL-like tool for humans, for testing,")
	fmt.Fprintln(w, "debugging, and generally interacting with HTTP servers.")
	fmt.Fprintln(w, "It speaks JSON natively, makes forms and file uploads easy,")
	fmt.Fprintln(w, "and can sign request bodies with an HMAC.")
	fmt.Fprintln(w, ".Pp")
	fmt.Fprintln(w, "Request data may be read from standard input,")
	fmt.Fprintln(w, "and response bodies may be written to standard output")
	fmt.Fprintln(w, "or downloaded to a file.")
	fmt.Fprintln(w, ".Pp")
	fmt.Fprintln(w, "The sole required argument is")
	fmt.Fprintln(w, ".Ar URL ,")
	fmt.Fprintln(w, "which may omit the")
	fmt.Fprintln(w, ".Cm http://")
	fmt.Fprintln(w, "scheme.")
	fmt.Fprintln(w, ".Pp")
	fmt.Fprintln(w, "The arguments are as follows:")

	// Rendered width of each option tag, in plain text, to size the -tag
	// column. A dot must be escaped with \& so it is not read as one of the
	// legacy .N/.C/.B/.I/.R/.P font macros (which would also throw off the
	// width computation).
	tagText := func(o manOption) string {
		var b strings.Builder
		for i, s := range append(append([]string{}, o.shorts...), o.longs...) {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteByte('-')
			b.WriteString(s)
		}
		if o.arg != "" {
			b.WriteByte(' ')
			b.WriteString(o.arg)
		}
		return b.String()
	}
	var widest string
	for _, o := range manOptions {
		if t := tagText(o); len(t) > len(widest) {
			widest = t
		}
	}
	fmt.Fprintf(w, ".Bl -tag -width \"%s\"\n", escapeMacroTraps(widest))

	for _, o := range manOptions {
		var spellings []string
		spellings = append(spellings, o.shorts...)
		spellings = append(spellings, o.longs...)

		var line strings.Builder
		line.WriteString(".It")
		for i, s := range spellings {
			if i > 0 {
				line.WriteString(" ,")
			}
			line.WriteString(" Fl ")
			line.WriteString(escapeMacroTraps(s))
		}
		if o.arg != "" {
			line.WriteString(" Ar ")
			line.WriteString(o.arg)
		}
		fmt.Fprintln(w, line.String())

		if o.def != "" {
			fmt.Fprintf(w, "%s (default %s).\n", o.desc, o.def)
		} else {
			fmt.Fprintf(w, "%s.\n", o.desc)
		}
	}
	fmt.Fprintln(w, ".El")

	fmt.Fprintln(w, ".Sh REQUEST ITEMS")
	fmt.Fprintln(w, "Arguments after the URL are key/value pairs distinguished by")
	fmt.Fprintln(w, "the separator they use:")
	fmt.Fprintln(w, ".Bl -tag -width Ds")
	fmt.Fprintln(w, ".It Ar name : Ns Ar value")
	fmt.Fprintln(w, "Sets an arbitrary HTTP header.")
	fmt.Fprintln(w, ".It Ar field Ns = Ns Ar value")
	fmt.Fprintln(w, "Adds a request data field, serialized as JSON by default.")
	fmt.Fprintln(w, ".It Ar field Ns := Ns Ar json")
	fmt.Fprintln(w, "Adds a raw JSON field value, for booleans, numbers, arrays, and objects.")
	fmt.Fprintln(w, ".It Ar field Ns =@ Ns Ar file")
	fmt.Fprintln(w, "Reads the field value from a file.")
	fmt.Fprintln(w, ".It Ar field Ns :=@ Ns Ar file")
	fmt.Fprintln(w, "Reads a raw JSON field value from a file.")
	fmt.Fprintln(w, ".It Ar field Ns @ Ns Ar file")
	fmt.Fprintln(w, "Uploads a file as a form field (with -form only).")
	fmt.Fprintln(w, ".El")

	fmt.Fprintln(w, ".Sh ENVIRONMENT")
	fmt.Fprintln(w, ".Bl -tag -width Ds")
	fmt.Fprintln(w, ".It Ev HTTP_PROXY , HTTPS_PROXY , NO_PROXY")
	fmt.Fprintln(w, "Standard proxy configuration, honored when no -proxy is given.")
	fmt.Fprintln(w, ".El")
	fmt.Fprintln(w, ".Pp")
	fmt.Fprintln(w, "The")
	fmt.Fprintln(w, ".Fl Fl hmac")
	fmt.Fprintln(w, "option names an environment variable holding the signing")
	fmt.Fprintln(w, "details as algorithm, header name, and secret separated by colons,")
	fmt.Fprintln(w, "for example")
	fmt.Fprintln(w, ".Cm sha256:x-hub-signature:very_secret .")

	fmt.Fprintln(w, ".Sh EXIT STATUS")
	fmt.Fprintln(w, ".Ex -std")
	fmt.Fprintln(w, "A usage error exits with status 2.")

	fmt.Fprintln(w, ".Sh EXAMPLES")
	fmt.Fprintln(w, "Issue a GET request:")
	fmt.Fprintln(w, ".Dl $ gurl example.net")
	fmt.Fprintln(w, ".Pp")
	fmt.Fprintln(w, "PUT a JSON object with a custom header:")
	fmt.Fprintln(w, ".Dl $ gurl PUT example.org X-API-Token:123 name=John")
	fmt.Fprintln(w, ".Pp")
	fmt.Fprintln(w, "Send raw JSON from a file body:")
	fmt.Fprintln(w, ".Dl $ gurl POST example.org/person < person.json")
	fmt.Fprintln(w, ".Pp")
	fmt.Fprintln(w, "Download a file:")
	fmt.Fprintln(w, ".Dl $ gurl -download example.org/file")

	fmt.Fprintln(w, ".Sh SEE ALSO")
	fmt.Fprintln(w, ".Lk https://curl.se/ curl")
	fmt.Fprintln(w, ".Lk https://httpie.io/ httpie")

	fmt.Fprintln(w, ".Sh AUTHORS")
	fmt.Fprintln(w, ".An -nosplit")
	fmt.Fprintln(w, ".An Dave Cottlehuber Aq Mt dch@skunkwerks.at")

	fmt.Fprintln(w, ".Sh CAVEATS")
	fmt.Fprintln(w, "HMAC signatures cover only the request body, not headers or query")
	fmt.Fprintln(w, "parameters.")
	fmt.Fprintln(w, "Do not assume those are authenticated.")
}

// printMan writes the manual page to standard output and exits.
func printMan() {
	writeMan(os.Stdout)
}
