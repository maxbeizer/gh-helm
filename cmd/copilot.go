package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maxbeizer/gh-helm/mcp"
	"github.com/spf13/cobra"
)

var copilotCmd = &cobra.Command{
	Use:   "copilot",
	Short: "Copilot integration — MCP server and skills",
}

var copilotServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server over stdio",
	Long:  "Starts a JSON-RPC MCP server on stdin/stdout for use by Copilot CLI, VS Code, or any MCP client.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcp.Serve(os.Stdin, os.Stdout, os.Stderr)
	},
}

var copilotToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		tools := mcp.Tools()
		for _, t := range tools {
			fmt.Printf("%-30s %s\n", t.Name, t.Description)
		}
		return nil
	},
}

var copilotSkillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "List available Copilot skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := listSkills()
		if err != nil {
			return err
		}
		for _, s := range skills {
			fmt.Println(s)
		}
		return nil
	},
}

var copilotTestCmd = &cobra.Command{
	Use:   "test <query>",
	Short: "Test which skill matches a natural language query",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		suggestion := suggestSkill(query)
		fmt.Printf("Skill:   %s\n", suggestion.Skill)
		fmt.Printf("Tool:    %s\n", suggestion.Tool)
		fmt.Printf("Reason:  %s\n", suggestion.Reason)
		return nil
	},
}

type skillSuggestion struct {
	Skill  string
	Tool   string
	Reason string
}

func suggestSkill(query string) skillSuggestion {
	q := strings.ToLower(query)
	switch {
	case strings.Contains(q, "start") || strings.Contains(q, "pick up") || strings.Contains(q, "work on") || strings.Contains(q, "claim"):
		return skillSuggestion{"pick-up-work", "helm.project.start", "work assignment query"}
	case strings.Contains(q, "status") || strings.Contains(q, "working on") || strings.Contains(q, "progress"):
		return skillSuggestion{"agent-status", "helm.project.status", "status query"}
	case strings.Contains(q, "suggest") || strings.Contains(q, "recommend") || strings.Contains(q, "what should"):
		return skillSuggestion{"pick-up-work", "helm.project.suggest", "work suggestion query"}
	case strings.Contains(q, "pulse") || strings.Contains(q, "team health") || strings.Contains(q, "how's the team"):
		return skillSuggestion{"team-pulse", "helm.manager.pulse", "team health query"}
	case strings.Contains(q, "1-1") || strings.Contains(q, "one on one") || strings.Contains(q, "prep"):
		return skillSuggestion{"one-on-one-prep", "helm.manager.prep", "1-1 prep query"}
	case strings.Contains(q, "observe") || strings.Contains(q, "observation"):
		return skillSuggestion{"team-pulse", "helm.manager.observe", "observation query"}
	case strings.Contains(q, "report") || strings.Contains(q, "report card"):
		return skillSuggestion{"one-on-one-prep", "helm.manager.report", "report card query"}
	case strings.Contains(q, "stats") || strings.Contains(q, "velocity") || strings.Contains(q, "cycle time"):
		return skillSuggestion{"team-pulse", "helm.manager.stats", "statistics query"}
	case strings.Contains(q, "doctor") || strings.Contains(q, "validate") || strings.Contains(q, "check setup"):
		return skillSuggestion{"agent-status", "helm.doctor", "setup validation query"}
	case strings.Contains(q, "daemon") || strings.Contains(q, "continuous") || strings.Contains(q, "auto"):
		return skillSuggestion{"pick-up-work", "helm.project.daemon", "daemon query"}
	case strings.Contains(q, "config") || strings.Contains(q, "settings"):
		return skillSuggestion{"agent-status", "helm.config.show", "config query"}
	case strings.Contains(q, "pillar") || strings.Contains(q, "performance"):
		return skillSuggestion{"team-pulse", "helm.manager.pillars", "pillar query"}
	default:
		return skillSuggestion{"agent-status", "helm.project.status", "default — agent status"}
	}
}

func listSkills() ([]string, error) {
	skillDir := filepath.Join("copilot-skills")
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return nil, fmt.Errorf("read copilot-skills/: %w (run from the gh-helm repo root)", err)
	}
	var skills []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		skills = append(skills, strings.TrimSuffix(entry.Name(), ".md"))
	}
	sort.Strings(skills)
	return skills, nil
}

func init() {
	copilotCmd.AddCommand(copilotServeCmd)
	copilotCmd.AddCommand(copilotToolsCmd)
	copilotCmd.AddCommand(copilotSkillsCmd)
	copilotCmd.AddCommand(copilotTestCmd)
}
