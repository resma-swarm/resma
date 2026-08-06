package version

import "fmt"

// Version is the semantic version of the CLI, set via ldflags.
var Version = "dev"

// BuildDate is the build timestamp, set via ldflags.
var BuildDate = "unknown"

// Commit is the git commit hash, set via ldflags.
var Commit = "none"

// String returns a formatted version string.
func String() string {
	return fmt.Sprintf("resma %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
