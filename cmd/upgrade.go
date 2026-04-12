package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jinto/ina/config"
	"github.com/jinto/ina/daemon"
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

		// Sync cached version files so HUD doesn't show stale upgrade hints.
		writeLocalVersion(latest)
		writeLatestVersion(latest)

		restartDaemonAfterUpgrade()

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

// restartDaemonAfterUpgrade restarts the ina daemon so it picks up the new binary.
// Uses launchctl if managed by launchd, otherwise stops and re-launches directly.
func restartDaemonAfterUpgrade() {
	// Always stop any running daemon first (including manually launched ones)
	// to free the hook port before launchd starts a new instance.
	daemon.StopRunning() // ignore error — may not be running

	if _, err := os.Stat(plistPath()); err == nil {
		time.Sleep(500 * time.Millisecond)
		// launchd-managed: kickstart restarts the service.
		out, err := exec.Command("launchctl", "kickstart",
			fmt.Sprintf("gui/%d/%s", os.Getuid(), plistLabel)).CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: daemon restart failed: %s\n", strings.TrimSpace(string(out)))
			return
		}
		fmt.Println("Daemon restarted via launchd.")
		return
	}

	// Not launchd-managed: try stop + re-launch.
	if err := daemon.StopRunning(); err != nil {
		// No daemon running — nothing to restart.
		return
	}

	// Brief wait for the old process to exit.
	time.Sleep(500 * time.Millisecond)

	binPath, err := os.Executable()
	if err != nil {
		fmt.Println("Daemon stopped. Restart manually: ina daemon &")
		return
	}
	binPath, _ = filepath.EvalSymlinks(binPath)

	cmd := exec.Command(binPath, "daemon")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		fmt.Printf("Daemon stopped but restart failed: %v\nRestart manually: ina daemon &\n", err)
		return
	}
	cmd.Process.Release()
	fmt.Printf("Daemon restarted (pid=%d).\n", cmd.Process.Pid)
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
