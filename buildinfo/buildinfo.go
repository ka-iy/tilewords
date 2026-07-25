// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package buildinfo

import (
	"strings"
)

// This file contains variables and helper functions for build-related info which
// is injected into built binaries at link time. See the Makefile for how this
// info is injected into the built binaries at build time.

// Variables to hold the build information injected into the binary during builds.
// These are injected into the binary at link time via the "ldflags" argument to
// "go build", for example:
//
//	go build -ldflags="-X common.BuildTimestamp=`date -u +'%Y-%m-%d_%H:%M:%S_%Z'`
//
// Per the Go memory model, these strings are safe for concurrent access AS LONG
// as they are not modified at runtime. Which, since they're injected into the
// binary at link time, they should not be. This is why we have the helper
// functions which only return the appropriate string.

var (
	// buildVersion will contain the git information for the current state of
	// the codebase. This is usually set using:
	//    git describe --tags --long --dirty --always
	buildVersion string

	// buildType will hold the type of the build - whether production, debug,
	// dev, instrumentation, etc.
	buildType string

	// buildTimestamp contains the UTC build timestamp of when the binary was built.
	// The timestamp is formatted as a single string word with the date and time
	// (and timezone, which in this case is "UTC") separated by underscores.
	buildTimestamp string
)

func BuildVersion() string {
	return buildVersion
}

// BuildType returns the lower-cased value of buildType.
//
// It can be used to set certain functionality on or off depending upon the
// build type. For example, production builds can have log levels finer than
// INFO turned off to avoid leaking sensitive information. Instrumentation builds
// can enable pprof or OTEL code instrumentation without having said code
// instrumentation affect runtime behavior for production builds.
//
// For available build type strings, see the Makefile for this project.
func BuildType() string {
	return strings.ToLower(buildType)
}

// BuildTimestamp returns the build timestamp with each component-separator
// underscore character replaced by a single space.
func BuildTimestamp() string {
	return strings.ReplaceAll(buildTimestamp, "_", " ")
}

// BuildInfoAsStringSlice is useful when printing the build info in multiple
// lines, for example from a command-line invocation using the "--version"/"-v"
// argument.
func BuildInfoAsStringSlice() []string {
	return []string{
		"Build Version: " + buildVersion,
		"Build Type: " + BuildType(),
		"Build Timestamp: " + BuildTimestamp(),
	}
}

// DumpBuildInfo returns the build info as a single newline-separated string.
func DumpBuildInfo() string {
	return strings.Join(BuildInfoAsStringSlice(), "\n")
}

// IsProductionBuild returns true if BuildType returns "production".
func IsProductionBuild() bool {
	return BuildType() == "production"
}

// SetBuildTypeForTest overrides the link-time build type and returns a function
// that restores the previous value. It exists solely so packages whose behavior
// branches on BuildType()/IsProductionBuild() — e.g. the logger's Trace/Debug
// production gating — can be unit-tested without link-time build metadata.
// Production code must never call this; use it only from tests, paired with a
// deferred call to the returned restore func.
func SetBuildTypeForTest(bt string) (restore func()) {
	prev := buildType
	buildType = bt
	return func() { buildType = prev }
}
