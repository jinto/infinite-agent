package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jinto/ina/config"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X github.com/jinto/ina/cmd.Version=v1.3.0"
var Version = "dev"

const (
	repoOwner = "jinto"
	repoName  = "infinite-angel"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade ina to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		current := currentVersion()
		fmt.Printf("Current version: %s\n", current)

		latest, err := fetchLatestVersion()
		if err != nil {
			return fmt.Errorf("check latest version: %w", err)
		}
		fmt.Printf("Latest version:  %s\n", latest)

		if current == latest {
			fmt.Println("Already up to date.")
			return nil
		}

		fmt.Printf("Upgrading %s → %s...\n", current, latest)
		if err := runInstallScript(); err != nil {
			return fmt.Errorf("upgrade failed: %w", err)
		}

		fmt.Println("Upgrade complete.")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current ina version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(currentVersion())
	},
}

// currentVersion returns the build-time version, falling back to the file-based version.
func currentVersion() string {
	if Version != "dev" {
		return Version
	}
	path := filepath.Join(config.DataDir(), "version")
	data, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

func writeLocalVersion(version string) {
	path := filepath.Join(config.DataDir(), "version")
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, []byte(version+"\n"), 0600)
}

func fetchLatestVersion() (string, error) {
	return fetchLatestVersionTimeout(10 * time.Second)
}

func fetchLatestVersionTimeout(timeout time.Duration) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

// CheckForUpdate checks if a newer version is available and prints a message.
// Uses a short timeout to avoid blocking interactive commands.
func CheckForUpdate() {
	current := currentVersion()
	if current == "unknown" {
		return
	}
	latest, err := fetchLatestVersionTimeout(2 * time.Second)
	if err != nil {
		return
	}
	writeLatestVersion(latest)
	if latest == current {
		return
	}
	fmt.Printf("\nina %s available (current: %s). Run 'ina upgrade' to update.\n", latest, current)
}

func writeLatestVersion(version string) {
	path := filepath.Join(config.DataDir(), "latest_version")
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, []byte(version+"\n"), 0600)
}

func runInstallScript() error {
	script := fmt.Sprintf("curl -sSL https://raw.githubusercontent.com/%s/%s/main/install.sh | sh", repoOwner, repoName)
	cmd := exec.Command("sh", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(versionCmd)
}
