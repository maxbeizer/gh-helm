package cmd

import (
	"fmt"
	"time"

	"github.com/maxbeizer/max-ops/internal/agent"
	"github.com/maxbeizer/max-ops/internal/config"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var projectDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Continuous project agent loop",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("max-ops.yaml")
		if err != nil {
			return err
		}

		intervalText, _ := cmd.Flags().GetString("interval")
		interval, err := time.ParseDuration(intervalText)
		if err != nil {
			interval = 30 * time.Second
		}
		maxPerHour, _ := cmd.Flags().GetInt("max-per-hour")
		status, _ := cmd.Flags().GetString("status")
		label, _ := cmd.Flags().GetString("label")
		projectNumber, _ := cmd.Flags().GetInt("project")
		projectOwner, _ := cmd.Flags().GetString("owner")
		codespace, _ := cmd.Flags().GetBool("codespace")
		machine, _ := cmd.Flags().GetString("codespace-machine")
		idleTimeout, _ := cmd.Flags().GetString("codespace-idle-timeout")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if maxPerHour == 0 {
			maxPerHour = cfg.Agent.MaxPerHour
		}
		if maxPerHour == 0 {
			maxPerHour = 3
		}

		opts := agent.DaemonOpts{
			Interval:             interval,
			MaxPerHour:           maxPerHour,
			Status:               status,
			Label:                label,
			Codespace:            codespace,
			CodespaceMachine:     machine,
			CodespaceIdleTimeout: idleTimeout,
			DryRun:               dryRun,
			ProjectOwner:         projectOwner,
			ProjectNumber:        projectNumber,
		}

		jsonFlag, _ := cmd.Flags().GetBool("json")
		jqExpr, _ := cmd.Flags().GetString("jq")
		if jsonFlag || jqExpr != "" {
			out := output.New(cmd)
			opts.Logger = jsonLogger{out: out}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "project daemon started (interval: %s)\n", interval)
		}
		return agent.RunDaemon(cmd.Context(), cfg.Project, opts)
	},
}

type jsonLogger struct {
	out *output.Output
}

func (j jsonLogger) Printf(format string, args ...any) {
	payload := map[string]any{
		"time":    time.Now().Format(time.RFC3339),
		"message": fmt.Sprintf(format, args...),
	}
	_ = j.out.Print(payload)
}

func init() {
	projectDaemonCmd.Flags().String("interval", "30s", "Poll interval")
	projectDaemonCmd.Flags().Int("max-per-hour", 3, "Guardrail: max issues per hour")
	projectDaemonCmd.Flags().String("status", "Ready", "Only claim items in this status")
	projectDaemonCmd.Flags().String("label", "", "Only claim items with this label")
	projectDaemonCmd.Flags().Int("project", 0, "Project number (override config)")
	projectDaemonCmd.Flags().String("owner", "", "Project owner (override config)")
	projectDaemonCmd.Flags().Bool("codespace", false, "Create a codespace on each PR branch")
	projectDaemonCmd.Flags().String("codespace-machine", "basicLinux32gb", "Codespace machine type")
	projectDaemonCmd.Flags().String("codespace-idle-timeout", "30m", "Codespace idle timeout")
	projectDaemonCmd.Flags().Bool("dry-run", false, "Log actions without executing")
	projectCmd.AddCommand(projectDaemonCmd)
}
