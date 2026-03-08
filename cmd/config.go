package cmd

import (
	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config commands",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current helm.toml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("helm.toml")
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
