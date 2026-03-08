package cmd

import (
	"fmt"
	"strings"

	"github.com/maxbeizer/max-ops/internal/doctor"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check project health and max-ops compliance",
	RunE: func(cmd *cobra.Command, args []string) error {
		fix, _ := cmd.Flags().GetBool("fix")
		result, err := doctor.Run(cmd.Context(), doctor.Options{Fix: fix})
		if err != nil {
			return err
		}
		jsonFlag, _ := cmd.Flags().GetBool("json")
		jqExpr, _ := cmd.Flags().GetString("jq")
		if jsonFlag || jqExpr != "" {
			out := output.New(cmd)
			return out.Print(result)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "🏥 max-ops doctor — project health check")
		fmt.Fprintln(cmd.OutOrStdout(), "")
		for _, line := range formatDoctorChecks(result.Checks) {
			fmt.Fprintln(cmd.OutOrStdout(), line)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintf(cmd.OutOrStdout(), "  Result: %d passed, %d warnings, %d failures\n", result.Summary.Passed, result.Summary.Warnings, result.Summary.Failures)
		if result.Summary.Warnings > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "  Run 'max-ops upgrade' to fix warnings automatically.")
		}
		return nil
	},
}

func formatDoctorChecks(checks []doctor.CheckResult) []string {
	order := []string{"config", "source_of_truth", "project_board", "labels", "devcontainer", "notifications", "auth", "state"}
	labels := map[string]string{
		"config":         "Config",
		"source_of_truth": "Source of Truth",
		"project_board":  "Project Board",
		"labels":         "Labels",
		"devcontainer":   "DevContainer",
		"notifications":  "Notifications",
		"auth":           "Auth",
		"state":          "State",
	}
	lookup := map[string]doctor.CheckResult{}
	for _, check := range checks {
		lookup[check.Key] = check
	}
	lines := []string{}
	for _, key := range order {
		check, ok := lookup[key]
		if !ok {
			continue
		}
		status := iconForStatus(check.Status)
		label := labels[key]
		message := strings.TrimSpace(check.Message)
		if message != "" {
			lines = append(lines, fmt.Sprintf("  %s %s: %s", status, label, message))
		} else {
			lines = append(lines, fmt.Sprintf("  %s %s", status, label))
		}
	}
	return lines
}

func iconForStatus(status doctor.Status) string {
	switch status {
	case doctor.StatusPass:
		return "✅"
	case doctor.StatusWarn:
		return "⚠️ "
	case doctor.StatusFail:
		return "❌"
	case doctor.StatusInfo:
		return "ℹ️ "
	default:
		return "•"
	}
}

func init() {
	doctorCmd.Flags().Bool("fix", false, "Automatically fix warnings")
}
