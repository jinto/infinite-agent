package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jinto/ina/config"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure Claude Code hooks and MCP server for ina integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
		settingsPath := filepath.Join(home, ".claude", "settings.json")

		settings := make(map[string]interface{})
		if data, err := os.ReadFile(settingsPath); err == nil {
			if err := json.Unmarshal(data, &settings); err != nil {
				return fmt.Errorf("parse existing %s: %w", settingsPath, err)
			}
		}

		port := cfg.Daemon.GetHookPort()
		base := fmt.Sprintf("http://127.0.0.1:%d", port)

		// Build hooks config — each entry maps an event to an HTTP hook endpoint.
		hookEntry := func(matcher, path string, timeout int) []map[string]interface{} {
			return []map[string]interface{}{
				{"matcher": matcher, "hooks": []map[string]interface{}{
					{"type": "http", "url": base + path, "timeout": timeout},
				}},
			}
		}

		hooks := map[string]interface{}{
			"SessionStart": hookEntry("", "/hooks/session-start", 5),
			"SessionEnd":   hookEntry("", "/hooks/session-end", 5),
			"Stop":         hookEntry("", "/hooks/stop", 5),
			"PostToolUse":  hookEntry(".*", "/hooks/post-tool-use", 2),
		}

		// Merge hooks into existing settings
		existingHooks, _ := settings["hooks"].(map[string]interface{})
		if existingHooks == nil {
			existingHooks = make(map[string]interface{})
		}
		for k, v := range hooks {
			existingHooks[k] = v
		}
		settings["hooks"] = existingHooks

		// Statusline — ina hud
		inaPath := findIna()
		if inaPath != "" {
			settings["statusLine"] = map[string]interface{}{
				"type":    "command",
				"command": inaPath + " hud",
			}
			fmt.Printf("Statusline: %s hud\n", inaPath)
		}

		// Find ina-mcp binary
		mcpPath := findInfaMCP()
		if mcpPath != "" {
			mcpServers, _ := settings["mcpServers"].(map[string]interface{})
			if mcpServers == nil {
				mcpServers = make(map[string]interface{})
			}
			mcpServers["ina"] = map[string]interface{}{
				"command": mcpPath,
			}
			settings["mcpServers"] = mcpServers
			fmt.Printf("MCP server: %s\n", mcpPath)
		}

		if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
			return fmt.Errorf("create settings directory: %w", err)
		}
		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(settingsPath, data, 0600); err != nil {
			return err
		}

		fmt.Printf("Hooks configured → %s\n", settingsPath)
		fmt.Printf("Hook endpoint: %s/hooks/*\n", base)

		// Install Context7 MCP if not already configured
		setupContext7()

		// Install pre-push hook for LLM-Judge eval
		installPrePushHook()

		fmt.Println("\nRun 'ina daemon' to start receiving events.")
		return nil
	},
}

func findIna() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe
	}
	return real
}

func findInfaMCP() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(exe), "ina-mcp")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	candidate = filepath.Join(config.DataDir(), "bin", "ina-mcp")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func setupContext7() {
	// Check if context7 is already configured
	out, err := exec.Command("claude", "mcp", "list").CombinedOutput()
	if err == nil && strings.Contains(string(out), "context7") {
		fmt.Println("Context7 MCP: already configured")
		return
	}

	// Install Context7 (no API key required)
	fmt.Print("Installing Context7 MCP (library docs)... ")
	if err := exec.Command("claude", "mcp", "add", "context7", "--", "npx", "-y", "@upstash/context7-mcp").Run(); err != nil {
		fmt.Printf("skipped (%v)\n", err)
		return
	}
	fmt.Println("done")
}


func installPrePushHook() {
	// Find project root (git top-level)
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		fmt.Println("Pre-push hook: skipped (not a git repo)")
		return
	}
	root := strings.TrimSpace(string(out))

	src := filepath.Join(root, "scripts", "pre-push.sh")
	if _, err := os.Stat(src); err != nil {
		fmt.Println("Pre-push hook: skipped (scripts/pre-push.sh not found)")
		return
	}

	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		fmt.Printf("Pre-push hook: skipped (%v)\n", err)
		return
	}

	dst := filepath.Join(hooksDir, "pre-push")

	// Backup existing hook if it's not ours
	if data, err := os.ReadFile(dst); err == nil {
		if !strings.Contains(string(data), "ina eval") {
			backup := dst + ".backup"
			os.WriteFile(backup, data, 0o755)
			fmt.Printf("Pre-push hook: backed up existing hook to %s\n", backup)
		}
	}

	// Copy hook script
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Printf("Pre-push hook: skipped (%v)\n", err)
		return
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		fmt.Printf("Pre-push hook: skipped (%v)\n", err)
		return
	}

	fmt.Printf("Pre-push hook: installed → %s\n", dst)
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
