// Package version reports the ore build version for CLI use.
//
// Version is overwritten at release time via ldflags:
//
//	-ldflags "-X github.com/rsiota/ore/internal/version.Version=v0.1.0"
package version

import "runtime/debug"

// Version is the current ore version, set via ldflags at release time.
var Version = ""

// String returns a short version label like "ore v1.2.3".
func String() string {
	if Version != "" {
		return "ore " + Version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "ore (unknown)"
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		return "ore (devel)"
	}
	return "ore " + v
}
