package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-helm",
	Short: "gh-helm — autonomous developer agents backed by GitHub",
}

func Execute(ctx context.Context) error {
	rootCmd.SetContext(ctx)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(managerCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(versionCmd)
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "Output JSON")
	rootCmd.PersistentFlags().String("jq", "", "Filter JSON output with jq")
}
