package cmd

import (
	"context"
	"log/slog"
	"os"

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
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable debug logging")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		slog.SetDefault(slog.New(handler))
		return nil
	}
}
