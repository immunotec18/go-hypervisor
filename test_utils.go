//go:build darwin && arm64

package hypervisor

import "os"

// isCI returns true if running in GitHub Actions
func isCI() bool {
	return os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
}
