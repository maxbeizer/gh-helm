package cmd

import (
	"github.com/maxbeizer/max-ops/internal/config"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config commands",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current max-ops.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("max-ops.yaml")
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(cfg)
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
}
