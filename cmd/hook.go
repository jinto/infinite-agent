package cmd

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"time"

	"github.com/jinto/ina/config"
	"github.com/jinto/ina/daemon"
	"github.com/spf13/cobra"
)

const defaultHookTimeout = 500 * time.Millisecond

var hookCmd = &cobra.Command{
	Use:    "hook [event]",
	Short:  "Forward a Claude Code hook event to the daemon (internal use)",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		event := args[0]
		// On session start, check if the binary is behind the plugin cache.
		if event == "session-start" {
			checkPluginVersion(os.Stdout)
		}
		// Always exit 0 — errors are silently swallowed so Claude Code
		// never surfaces "hook error" messages to the user.
		_ = forwardHook(config.SocketPath(), event, os.Stdin, defaultHookTimeout)
		return nil
	},
}

// forwardHook reads stdin and sends it to the daemon via unix socket.
// Returns nil on any failure — this function must never cause a non-zero exit.
func forwardHook(socketPath, event string, stdin io.Reader, timeout time.Duration) error {
	// Read stdin payload (Claude Code sends hook context as JSON).
	var body json.RawMessage
	if stdin != nil {
		data, _ := io.ReadAll(stdin)
		if len(data) > 0 {
			body = data
		}
	}

	payload, err := json.Marshal(struct {
		Event string          `json:"event"`
		Body  json.RawMessage `json:"body,omitempty"`
	}{Event: event, Body: body})
	if err != nil {
		return nil
	}

	cmd := daemon.Command{
		Action: daemon.ActionHook,
		Data:   payload,
	}

	// Connect with timeout.
	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		return nil
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		return nil
	}

	// Read response (best-effort, don't block long).
	var resp daemon.Response
	json.NewDecoder(conn).Decode(&resp)

	return nil
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
