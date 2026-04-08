package cmd

import (
	"fmt"
	"os"

	"github.com/jinto/ina/config"
	"github.com/spf13/cobra"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "ina",
	Short: "Infinite Angel — coding angels that never stop",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		return config.EnsureDir()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to status
		return statusCmd.RunE(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
