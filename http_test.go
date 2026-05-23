package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// resetParseState clears globals that getHTTP reads/writes so each subtest
// starts from a clean slate.
func resetParseState() {
	jsonmap = make(map[string]interface{})
	form = false
	tmp := false
	isjson = &tmp
}

// runArgInChild re-execs the test binary so that a log.Fatal inside getHTTP
// doesn't kill the parent test process. The child runs only the named
// subtest of TestParseArgChild and returns combined output + exit code.
func runArgInChild(t *testing.T, name string, env ...string) (string, int) {
	t.Helper()
	// Use a substring match on the unique trailing portion of the name to
	// avoid the regexp escaping headaches that come with `/`, `=`, `@`, etc.
	// in subtest names.
	cmd := exec.Command(os.Args[0], "-test.run=TestParseArgChild", "-test.v")
	cmd.Env = append(append([]string{}, os.Environ()...), env...)
	cmd.Env = append(cmd.Env, "GURL_PARSE_CHILD=1", "GURL_PARSE_CHILD_NAME="+name)
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		code = -1
	}
	return string(out), code
}

// --- The actual table-driven argument parser tests --------------------------

type parseCase struct {
	name   string
	method string
	form   bool
	isjson bool
	args   []string
	// Expected outcome
	headers map[string]string      // subset of request headers asserted
	params  map[string]string      // full params map (only checked if non-nil)
	files   map[string]string      // full files map (only checked if non-nil)
	jsonmap map[string]interface{} // full jsonmap (only checked if non-nil)
	// runInChild=true => the case might call log.Fatal; run it under a
	// subprocess so the parent test survives.
	runInChild bool
	// wantExitCode is the expected exit code of the child (0 = success).
	// Only consulted when runInChild is true.
	wantExitCode int
	// wantOutputSubstr, when non-empty, requires the child's combined
	// output to contain this substring. Useful to assert *which* failure
	// path was taken (parse-time vs. marshal-time).
	wantOutputSubstr string
}

func parseCases(t *testing.T) []parseCase {
	t.Helper()
	tmpDir := t.TempDir()

	jsonFile := filepath.Join(tmpDir, "payload.json")
	if err := os.WriteFile(jsonFile, []byte(`{"nested":true,"n":3}`), 0644); err != nil {
		t.Fatal(err)
	}
	textFile := filepath.Join(tmpDir, "about.txt")
	if err := os.WriteFile(textFile, []byte("hello from file"), 0644); err != nil {
		t.Fatal(err)
	}
	weirdName := filepath.Join(tmpDir, "@weird.json")
	if err := os.WriteFile(weirdName, []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	_ = jsonFile
	_ = textFile
	_ = weirdName

	// Stash file paths in env so the child process can rebuild them.
	t.Setenv("GURL_TEST_JSON_FILE", jsonFile)
	t.Setenv("GURL_TEST_TEXT_FILE", textFile)
	t.Setenv("GURL_TEST_WEIRD_FILE", weirdName)
	t.Setenv("GURL_TEST_TMPDIR", tmpDir)

	return []parseCase{
		// ---- well-formed, currently-correct behavior ----------------------
		{
			name:    "simple JSON data field",
			method:  "POST",
			args:    []string{"name=John"},
			jsonmap: map[string]interface{}{"name": "John"},
		},
		{
			name:    "simple header",
			method:  "GET",
			args:    []string{"X-Token:abc123"},
			headers: map[string]string{"X-Token": "abc123"},
		},
		{
			name:    "header value containing colons",
			method:  "GET",
			args:    []string{"Authorization:Bearer abc:def:ghi"},
			headers: map[string]string{"Authorization": "Bearer abc:def:ghi"},
		},
		{
			name:   "raw JSON field with object value",
			method: "POST",
			args:   []string{`payload:={"a":"b","c":1}`},
			jsonmap: map[string]interface{}{
				"payload": json.RawMessage(`{"a":"b","c":1}`),
			},
		},
		{
			name:   "raw JSON field with array value",
			method: "POST",
			args:   []string{`hobbies:=["http","pies"]`},
			jsonmap: map[string]interface{}{
				"hobbies": json.RawMessage(`["http","pies"]`),
			},
		},
		{
			name:   "raw JSON field with boolean",
			method: "POST",
			args:   []string{`married:=false`},
			jsonmap: map[string]interface{}{
				"married": json.RawMessage(`false`),
			},
		},
		{
			name:   "raw JSON from file with :=@",
			method: "POST",
			args:   []string{"bookmarks:=@" + jsonFile},
			jsonmap: map[string]interface{}{
				"bookmarks": map[string]interface{}{
					"nested": true,
					"n":      float64(3),
				},
			},
		},
		{
			name:   "form file upload (form mode)",
			method: "POST",
			form:   true,
			args:   []string{"screenshot@" + textFile},
			files:  map[string]string{"screenshot": textFile},
		},

		// ---- edge cases that used to be parser bugs ----------------------

		{
			// Previously: `:` was checked before `=`, so the URL value was
			// split as a header. Now: data field with URL value works.
			name:    "data field whose value contains a URL",
			method:  "POST",
			args:    []string{"url=http://example.com/path"},
			jsonmap: map[string]interface{}{"url": "http://example.com/path"},
		},
		{
			name:    "data field with quoted value containing colon",
			method:  "POST",
			args:    []string{`name="John:Doe"`},
			jsonmap: map[string]interface{}{"name": `"John:Doe"`},
		},
		{
			name:   "GET query param whose value contains a colon",
			method: "GET",
			args:   []string{"redirect=https://example.com/cb"},
			params: map[string]string{"redirect": "https://example.com/cb"},
		},
		{
			// Previously: `=@file` outside form/json mode was caught by the
			// `@` branch first and triggered log.Fatal. Now it reads the file.
			name:    "=@ data field outside form/json mode",
			method:  "POST",
			args:    []string{"description=@" + textFile},
			jsonmap: map[string]interface{}{"description": "hello from file"},
		},
		{
			// Previously: form mode + `=value@something` was treated as a
			// file upload. Now: `=` wins, value is preserved verbatim.
			name:   "form data field whose value contains @",
			method: "POST",
			form:   true,
			args:   []string{"comment=look@me"},
			params: map[string]string{"comment": "look@me"},
		},
		{
			// Previously: strings.TrimLeft(s, "@") stripped all leading '@'
			// bytes, so a filename starting with '@' could not be referenced.
			// Now: strings.TrimPrefix only strips the separator '@'.
			name:    "raw JSON from file whose name begins with @",
			method:  "POST",
			args:    []string{"data:=@@weird.json"},
			jsonmap: map[string]interface{}{"data": map[string]interface{}{"ok": true}},
			// File is read from the working dir; child chdirs to tmpDir.
			runInChild: true, // chdir requires fresh process state
		},
		{
			// Malformed JSON in `:=` must fail fast with a clear message,
			// at parse time (before JsonBody marshal). The fix emits
			// "invalid JSON for field …" via log.Fatalf.
			name:             "malformed raw JSON is rejected at parse time",
			method:           "POST",
			args:             []string{`bad:={not valid json`},
			runInChild:       true,
			wantExitCode:     1,
			wantOutputSubstr: "invalid JSON",
		},

		// ---- backslash escaping ------------------------------------------
		// Rule: a backslash before a separator character (=, :, @, \) makes
		// that character literal and is itself stripped. A backslash before
		// anything else is kept verbatim, so Windows paths and similar
		// values don't need special handling. `\\` -> literal `\`.

		{
			// README's canonical example: foo\==bar -> data field "foo=" -> "bar".
			name:    "escape = in key produces literal = and uses next = as separator",
			method:  "POST",
			args:    []string{`foo\==bar`},
			jsonmap: map[string]interface{}{"foo=": "bar"},
		},
		{
			// Header whose name contains a literal colon: X\:Y:value.
			name:    "escape : in key produces header with literal :",
			method:  "GET",
			args:    []string{`X\:Y:value`},
			headers: map[string]string{"X:Y": "value"},
		},
		{
			// `weird\:=value`: the escape consumes the `:`. The next
			// unescaped separator is `=`, so this is a data field whose
			// key is literally `weird:` and value is `value`. (Without
			// the escape it would have been a raw JSON field.)
			name:    "escape := disambiguates from raw JSON",
			method:  "POST",
			args:    []string{`weird\:=value`},
			jsonmap: map[string]interface{}{"weird:": "value"},
		},
		{
			// Escape `@` in a key (form mode) so it's not a file-upload sep.
			name:   "escape @ in key keeps it as a data field",
			method: "POST",
			form:   true,
			args:   []string{`a\@b=c`},
			params: map[string]string{"a@b": "c"},
		},
		{
			// Backslash before a non-separator char is kept literally; this
			// is what lets Windows-style values pass through unchanged.
			name:    "backslash before non-separator survives in value",
			method:  "POST",
			args:    []string{`path=C:\Users\me`},
			jsonmap: map[string]interface{}{"path": `C:\Users\me`},
		},
		{
			// Doubled backslash -> literal single backslash.
			name:    "doubled backslash collapses to one in value",
			method:  "POST",
			args:    []string{`path=a\\b`},
			jsonmap: map[string]interface{}{"path": `a\b`},
		},
		{
			// Escape inside a value: literal `=` survives, no second split.
			name:    "escape = in value keeps it literal",
			method:  "POST",
			args:    []string{`equation=x\=y`},
			jsonmap: map[string]interface{}{"equation": "x=y"},
		},
		{
			// Escape `@` at the start of a file-load value so the file path
			// is parsed without TrimPrefix eating the marker. The fix:
			// `data=\@notafile` is a literal value, not a file load.
			name:    "escape leading @ in =@ value treats it as a literal value",
			method:  "POST",
			args:    []string{`data=\@notafile`},
			jsonmap: map[string]interface{}{"data": "@notafile"},
		},
		{
			// Trailing lone backslash is kept verbatim (no following char).
			name:    "trailing backslash in value is preserved",
			method:  "POST",
			args:    []string{`tail=hello\`},
			jsonmap: map[string]interface{}{"tail": `hello\`},
		},
	}
}

// TestParseArgChild is the worker invoked by the parent (TestGetHTTPArgParsing)
// for cases that may call log.Fatal. It runs only when GURL_PARSE_CHILD is set.
// The specific case to run is selected by GURL_PARSE_CHILD_NAME.
func TestParseArgChild(t *testing.T) {
	if os.Getenv("GURL_PARSE_CHILD") != "1" {
		t.Skip("child-only test")
	}
	if td := os.Getenv("GURL_TEST_TMPDIR"); td != "" {
		if err := os.Chdir(td); err != nil {
			t.Fatalf("child chdir: %v", err)
		}
	}
	target := os.Getenv("GURL_PARSE_CHILD_NAME")
	for _, tc := range childCases() {
		if tc.name != target {
			continue
		}
		runOneCase(t, tc)
		return
	}
	t.Fatalf("child: no case matched %q", target)
}

// childCases mirrors parseCases but reads file paths from environment so the
// child sees the same fixtures.
func childCases() []parseCase {
	_ = os.Getenv("GURL_TEST_TEXT_FILE")
	return []parseCase{
		{
			name:    "raw JSON from file whose name begins with @",
			method:  "POST",
			args:    []string{"data:=@@weird.json"},
			jsonmap: map[string]interface{}{"data": map[string]interface{}{"ok": true}},
		},
		{
			name:   "malformed raw JSON is rejected at parse time",
			method: "POST",
			args:   []string{`bad:={not valid json`},
			// Child is expected to log.Fatal here; we don't run runOneCase
			// assertions in that case (parser exits before returning).
		},
	}
}

func runOneCase(t *testing.T, tc parseCase) {
	t.Helper()
	resetParseState()
	form = tc.form
	tmp := tc.isjson
	isjson = &tmp

	req := getHTTP(tc.method, "http://example.test/", tc.args)

	for k, v := range tc.headers {
		if got := req.GetRequest().Header.Get(k); got != v {
			t.Errorf("header %q = %q, want %q", k, got, v)
		}
	}
	if tc.params != nil {
		if gotP := req.Params(); !reflect.DeepEqual(gotP, tc.params) {
			t.Errorf("params = %#v, want %#v", gotP, tc.params)
		}
	}
	if tc.files != nil {
		if gotF := req.Files(); !reflect.DeepEqual(gotF, tc.files) {
			t.Errorf("files = %#v, want %#v", gotF, tc.files)
		}
	}
	if tc.jsonmap != nil {
		if !jsonmapEqual(jsonmap, tc.jsonmap) {
			t.Errorf("jsonmap = %s, want %s",
				mustJSON(jsonmap), mustJSON(tc.jsonmap))
		}
	}
}

// TestGetHTTPArgParsing is the parent. It runs the cases that don't trigger
// log.Fatal directly, and re-execs the test binary for those that do.
func TestGetHTTPArgParsing(t *testing.T) {
	if os.Getenv("GURL_PARSE_CHILD") == "1" {
		t.Skip("parent-only test")
	}
	cases := parseCases(t)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.runInChild {
				out, code := runArgInChild(t, tc.name,
					"GURL_TEST_JSON_FILE="+os.Getenv("GURL_TEST_JSON_FILE"),
					"GURL_TEST_TEXT_FILE="+os.Getenv("GURL_TEST_TEXT_FILE"),
					"GURL_TEST_WEIRD_FILE="+os.Getenv("GURL_TEST_WEIRD_FILE"),
					"GURL_TEST_TMPDIR="+os.Getenv("GURL_TEST_TMPDIR"),
				)
				if code != tc.wantExitCode {
					t.Errorf("child exit code = %d, want %d\n%s",
						code, tc.wantExitCode, out)
				}
				if tc.wantOutputSubstr != "" && !strings.Contains(out, tc.wantOutputSubstr) {
					t.Errorf("child output missing %q\n%s",
						tc.wantOutputSubstr, out)
				}
				return
			}
			runOneCase(t, tc)
		})
	}
}

// jsonmapEqual compares two parsed-arg maps. json.RawMessage values are
// compared by their canonical JSON form so byte-identical input maps round
// trip correctly.
func jsonmapEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if ar, isRaw := av.(json.RawMessage); isRaw {
			br, ok := bv.(json.RawMessage)
			if !ok {
				return false
			}
			var ap, bp interface{}
			if json.Unmarshal(ar, &ap) != nil || json.Unmarshal(br, &bp) != nil {
				return false
			}
			if !reflect.DeepEqual(ap, bp) {
				return false
			}
			continue
		}
		if !reflect.DeepEqual(av, bv) {
			return false
		}
	}
	return true
}

func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<unmarshalable>"
	}
	return string(b)
}
