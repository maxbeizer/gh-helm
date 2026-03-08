package cmd

import (
	"fmt"
	"time"

	"github.com/maxbeizer/max-ops/internal/manager"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var managerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Run scheduled manager daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonFlag, _ := cmd.Flags().GetBool("json")
		jqExpr, _ := cmd.Flags().GetString("jq")
		if jsonFlag || jqExpr != "" {
			out := output.New(cmd)
			logger := managerJSONLogger{out: out}
			return manager.RunManagerDaemon(cmd.Context(), "manager-ops.yaml", logger)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "manager daemon started")
		return manager.RunManagerDaemon(cmd.Context(), "manager-ops.yaml", nil)
	},
}

type managerJSONLogger struct {
	out *output.Output
}

func (m managerJSONLogger) Printf(format string, args ...any) {
	payload := map[string]any{
		"time":    time.Now().Format(time.RFC3339),
		"message": fmt.Sprintf(format, args...),
	}
	_ = m.out.Print(payload)
}

func init() {
	managerCmd.AddCommand(managerStartCmd)
}
