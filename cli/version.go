package cli

import "runtime/debug"

// version can be set by the linker.
var version string

// Version returns a version string.  If the version variable in this package
// was set to a non-empty string by the linker, Version returns that.
// Otherwise, if build information is available via [debug.ReadBuildInfo],
// Version returns [debug.Buildinfo].Main.Version.  Otherwise, Version returns
// "unknown".
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		// This will be "(devel)" for binaries not built by
		// "go install PACKAGE@VERSION".
		return info.Main.Version
	}
	return "unknown"
}
