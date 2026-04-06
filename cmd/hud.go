package cmd

import (
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

func init() {
	rootCmd.AddCommand(hudCmd)
}
