package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jinto/ina/hud"
	"github.com/spf13/cobra"
)

var hudCmd = &cobra.Command{
	Use:   "hud",
	Short: "Statusline for Claude Code — reads stdin JSON, outputs context bar",
	RunE: func(cmd *cobra.Command, args []string) error {
		return hud.RenderFromStdin()
	},
}

var hudOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Enable HUD statusline",
	RunE: func(cmd *cobra.Command, args []string) error {
		os.Remove(hud.DisabledFile)
		fmt.Println("HUD enabled")
		return nil
	},
}

var hudOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Disable HUD statusline",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll(filepath.Dir(hud.DisabledFile), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(hud.DisabledFile, nil, 0600); err != nil {
			return err
		}
		fmt.Println("HUD disabled")
		return nil
	},
}

func init() {
	hudCmd.AddCommand(hudOnCmd, hudOffCmd)
	rootCmd.AddCommand(hudCmd)
}
