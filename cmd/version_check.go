package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// pluginCacheDir returns the path to the ina plugin cache directory.
func pluginCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "plugins", "cache", "ina-marketplace", "ina")
}

// maxVersion returns the highest semver string and its parsed parts from a
// list of directory names. Non-semver entries are silently ignored.
func maxVersion(dirs []string) (string, [3]int, bool) {
	var best string
	var bestParts [3]int
	found := false

	for _, d := range dirs {
		parts, ok := parseSemver(d)
		if !ok {
			continue
		}
		if !found || compareParts(parts, bestParts) > 0 {
			best = d
			bestParts = parts
			found = true
		}
	}
	return best, bestParts, found
}

// parseSemver parses "X.Y.Z" into [3]int. Returns false for non-semver strings.
func parseSemver(s string) ([3]int, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var result [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		result[i] = n
	}
	return result, true
}

// compareParts returns >0 if a > b, <0 if a < b, 0 if equal.
func compareParts(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return 0
}

// checkPluginVersion compares the running binary version against the plugin
// cache and writes a warning to w if the binary is outdated.
// All errors are silently ignored — this must never disrupt session start.
func checkPluginVersion(w io.Writer) {
	checkPluginVersionIn(pluginCacheDir(), currentVersion(), w)
}

// checkPluginVersionIn is the testable core: given a cache directory path and
// the current version string, it scans for the highest cached version and
// writes a one-line warning to w if the binary is outdated.
func checkPluginVersionIn(cacheDir, current string, w io.Writer) {
	if cacheDir == "" {
		return
	}
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}

	latest, latestParts, ok := maxVersion(names)
	if !ok {
		return
	}

	current = strings.TrimPrefix(current, "v")
	if current == "" || current == "unknown" || current == "dev" {
		return
	}

	currentParts, ok := parseSemver(current)
	if !ok {
		return
	}

	if compareParts(latestParts, currentParts) > 0 {
		fmt.Fprintf(w, "ina %s available (binary: %s). Run 'ina upgrade' to update.\n", latest, current)
	}
}
