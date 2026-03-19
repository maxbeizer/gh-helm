package cmd

import (
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/maxbeizer/gh-helm/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := output.New(cmd)
		return out.Print(map[string]string{"version": version.Version})
	},
}
