package cmd

import (
	"fmt"

	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := output.New(cmd)
		return out.Print(map[string]string{"version": version})
	},
}
