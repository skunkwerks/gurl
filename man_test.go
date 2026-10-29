// Copyright 2025-2026 SkunkWerks, GmbH.
// Licensed under the Apache License, Version 2.0; see LICENSE for details.

package main

import (
	"bytes"
	"flag"
	"os/exec"
	"strings"
	"testing"
)

// TestManCoversAllFlags asserts the manual page documents exactly the flags
// the program registers. If gurl.go gains or loses a flag and man.go is not
// updated in the same commit, this test fails.
func TestManCoversAllFlags(t *testing.T) {
	documented := map[string]bool{}
	for _, n := range manFlagNames() {
		documented[n] = true
	}

	var registered []string
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		registered = append(registered, f.Name)
	})

	// Meta-flags that are real but deliberately not listed in the OPTIONS
	// section because they are not request-shaping options. The -test.*
	// flags are injected by the testing package at run time and are not
	// part of gurl's own surface.
	meta := func(name string) bool {
		if name == "man" {
			return true
		}
		return strings.HasPrefix(name, "test.")
	}

	for _, name := range registered {
		if meta(name) {
			continue
		}
		if !documented[name] {
			t.Errorf("flag -%s is registered but not documented in man.go", name)
		}
	}

	// Reverse direction: every documented flag must actually exist, so we
	// never ship a page advertising a flag the binary rejects. -man itself
	// is a real flag and is intentionally excluded from manOptions (it is a
	// meta-flag, like -help), so account for it here.
	registeredSet := map[string]bool{}
	for _, name := range registered {
		registeredSet[name] = true
	}
	for _, name := range manFlagNames() {
		if !registeredSet[name] {
			t.Errorf("flag -%s is documented in man.go but not registered", name)
		}
	}
}

// TestManRenders runs the generator and, if mandoc(1) is available, lints
// the output with -Tlint. A failure here means the page has a markup error,
// not just a missing flag.
func TestManRenders(t *testing.T) {
	var buf bytes.Buffer
	writeMan(&buf)

	out := buf.String()
	// NAME section must be present and the NAME/Nd line load-bearing.
	if !strings.Contains(out, ".Sh NAME") {
		t.Fatal("missing NAME section")
	}
	if !strings.Contains(out, ".Nd") {
		t.Fatal("missing .Nd description")
	}

	// The first line must be the prologue in the correct order, with a
	// literal "Month Day, Year" date.
	lines := strings.Split(out, "\n")
	if len(lines) < 4 || !strings.HasPrefix(lines[0], ".Dd ") {
		t.Fatalf("prologue must start with .Dd, got %q", lines[0])
	}

	// Use exec.LookPath so the unit test is still meaningful on hosts
	// without mandoc (e.g. Linux CI); the FreeBSD port and this repo's
	// `make man` do the authoritative lint.
	mandoc, err := exec.LookPath("mandoc")
	if err != nil {
		t.Skip("mandoc not found; skipping lint")
	}
	cmd := exec.Command(mandoc, "-Tlint")
	cmd.Stdin = strings.NewReader(out)
	var lintErr bytes.Buffer
	cmd.Stderr = &lintErr
	_ = cmd.Run() // exit status is non-zero for STYLE notes too; inspect text
	stderr := lintErr.String()
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// A "referenced manual not found" STYLE note only means the target
		// page (curl, httpie) is not installed on this host, not that the
		// page is malformed. Anything else is a real problem.
		if strings.Contains(line, "STYLE: referenced manual not found") {
			continue
		}
		t.Errorf("mandoc -Tlint:\n%s", stderr)
		break
	}
}
