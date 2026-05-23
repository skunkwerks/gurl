// Copyright 2015-2026 SkunkWerks, GmbH.
// Licensed under the Apache License, Version 2.0; see LICENSE for details.

// Package version exposes the build-time version string.
//
// Override at link time via:
//
//	go build -ldflags "-X github.com/skunkwerks/gurl/version.Version=1.2.3"
//
// The baked-in default is used when no override is supplied (e.g.
// `go install` from a tarball).
package version

// Version is the semantic release version, without any leading "v".
var Version = "1.0.0"
