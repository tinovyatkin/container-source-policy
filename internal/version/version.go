package version

import "fmt"

var (
	version = "dev"
	commit  = "unknown"
)

// Version returns the current version string
func Version() string {
	if commit != "unknown" && len(commit) > 7 {
		return version + " (" + commit[:7] + ")"
	}
	return version
}

// Commit returns the git commit hash
func Commit() string {
	return commit
}

// UserAgent returns a User-Agent string for HTTP requests
// Format matches BuildKit's convention: "container-source-policy/{version}"
func UserAgent() string {
	return fmt.Sprintf("container-source-policy/%s", version)
}
