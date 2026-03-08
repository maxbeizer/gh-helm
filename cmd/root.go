package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "max-ops",
	Short: "max-ops — autonomous developer agents backed by GitHub",
}

func Execute(ctx context.Context) error {
	rootCmd.SetContext(ctx)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(managerCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	return rootCmd.Execute()
}
