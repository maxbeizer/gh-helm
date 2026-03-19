package mcp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ToolDefinition describes an MCP tool backed by a gh helm CLI command.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Build       func(args map[string]interface{}) ([]string, error)
}

// Tools returns all registered tool definitions, sorted by name.
func Tools() []ToolDefinition {
	out := make([]ToolDefinition, len(tools))
	copy(out, tools)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ToolByName looks up a tool by its MCP name.
func ToolByName(name string) (*ToolDefinition, bool) {
	for i := range tools {
		if tools[i].Name == name {
			return &tools[i], true
		}
	}
	return nil, false
}

// --- Tool definitions ---

var tools = []ToolDefinition{
	// Project agent tools
	{
		Name:        "helm_project_start",
		Description: "Claim a GitHub issue and generate a draft PR with AI-written code. The agent reads the issue, generates a plan, writes code, and pushes a draft PR.",
		InputSchema: objectSchema(map[string]interface{}{
			"issue":     intProp("Issue number to work on"),
			"repo":      stringProp("Repository owner/name (e.g. maxbeizer/copilot-atc)"),
			"model":     stringProp("AI model to use (default: from config)"),
			"dry-run":   boolProp("Show plan without executing"),
			"codespace": boolProp("Create a Codespace on the PR branch"),
		}, "issue"),
		Build: func(args map[string]interface{}) ([]string, error) {
			cmd := []string{"project", "start"}
			issue, ok := getInt(args, "issue")
			if !ok {
				return nil, fmt.Errorf("issue number is required")
			}
			cmd = append(cmd, "--issue", strconv.Itoa(issue))
			return appendFlags(cmd, args, []flagDef{
				{key: "repo", flag: "--repo"},
				{key: "model", flag: "--model"},
				{key: "dry-run", flag: "--dry-run", isBool: true},
				{key: "codespace", flag: "--codespace", isBool: true},
			}), nil
		},
	},
	{
		Name:        "helm_project_status",
		Description: "Show what the project agent is currently working on, including active session, issue, and PR details.",
		InputSchema: objectSchema(map[string]interface{}{}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			return []string{"project", "status"}, nil
		},
	},
	{
		Name:        "helm_project_suggest",
		Description: "Suggest work based on developer profile — recommends issues that match the developer's skills and growth areas.",
		InputSchema: objectSchema(map[string]interface{}{
			"repo":         stringProp("Repository owner/name"),
			"profile-repo": stringProp("1-1 repo containing developer-profile.toml (e.g. owner/dev-1-1)"),
		}, "profile-repo"),
		Build: func(args map[string]interface{}) ([]string, error) {
			cmd := []string{"project", "suggest"}
			profileRepo, ok := getString(args, "profile-repo")
			if !ok {
				return nil, fmt.Errorf("profile-repo is required")
			}
			cmd = append(cmd, "--profile-repo", profileRepo)
			return appendFlags(cmd, args, []flagDef{
				{key: "repo", flag: "--repo"},
			}), nil
		},
	},
	{
		Name:        "helm_project_sot",
		Description: "View or propose updates to the project's source of truth document.",
		InputSchema: objectSchema(map[string]interface{}{
			"propose": stringProp("Propose an update to the source of truth"),
		}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			cmd := []string{"project", "sot"}
			return appendFlags(cmd, args, []flagDef{
				{key: "propose", flag: "--propose"},
			}), nil
		},
	},
	{
		Name:        "helm_project_daemon",
		Description: "Start or query the continuous agent daemon that automatically picks up issues from the project board.",
		InputSchema: objectSchema(map[string]interface{}{
			"status":       stringProp("Board status to watch (e.g. Ready, Todo)"),
			"max-per-hour": intProp("Maximum issues to process per hour"),
			"dry-run":      boolProp("Show what would be processed without executing"),
		}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			cmd := []string{"project", "daemon"}
			return appendFlags(cmd, args, []flagDef{
				{key: "status", flag: "--status"},
				{key: "max-per-hour", flag: "--max-per-hour", isInt: true},
				{key: "dry-run", flag: "--dry-run", isBool: true},
			}), nil
		},
	},

	// Manager agent tools
	{
		Name:        "helm_manager_pulse",
		Description: "Team health overview — shows velocity, blockers, and activity patterns across the team.",
		InputSchema: objectSchema(map[string]interface{}{}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			return []string{"manager", "pulse"}, nil
		},
	},
	{
		Name:        "helm_manager_prep",
		Description: "Generate 1-1 meeting prep for a team member — recent activity, contributions mapped to pillars, talking points.",
		InputSchema: objectSchema(map[string]interface{}{
			"handle": stringProp("GitHub handle of the team member"),
		}, "handle"),
		Build: func(args map[string]interface{}) ([]string, error) {
			handle, ok := getString(args, "handle")
			if !ok {
				return nil, fmt.Errorf("handle is required")
			}
			return []string{"manager", "prep", handle}, nil
		},
	},
	{
		Name:        "helm_manager_observe",
		Description: "Generate weekly observations for the team — maps contributions to performance pillars, posts to 1-1 repos.",
		InputSchema: objectSchema(map[string]interface{}{
			"dry-run": boolProp("Preview observations without posting"),
		}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			cmd := []string{"manager", "observe"}
			return appendFlags(cmd, args, []flagDef{
				{key: "dry-run", flag: "--dry-run", isBool: true},
			}), nil
		},
	},
	{
		Name:        "helm_manager_stats",
		Description: "Team and individual statistics — velocity, cycle time, bus factor, contribution distribution.",
		InputSchema: objectSchema(map[string]interface{}{
			"handle": stringProp("GitHub handle (omit for full team)"),
		}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			cmd := []string{"manager", "stats"}
			return appendFlags(cmd, args, []flagDef{
				{key: "handle", flag: "--handle"},
			}), nil
		},
	},
	{
		Name:        "helm_manager_report",
		Description: "Full report card for a team member — pillar impact, growth trajectory, contributions summary.",
		InputSchema: objectSchema(map[string]interface{}{
			"handle": stringProp("GitHub handle of the team member"),
		}, "handle"),
		Build: func(args map[string]interface{}) ([]string, error) {
			handle, ok := getString(args, "handle")
			if !ok {
				return nil, fmt.Errorf("handle is required")
			}
			return []string{"manager", "report", handle}, nil
		},
	},
	{
		Name:        "helm_manager_pillars",
		Description: "Show configured performance pillar definitions and signal mappings.",
		InputSchema: objectSchema(map[string]interface{}{}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			return []string{"manager", "pillars"}, nil
		},
	},

	// Config and operations tools
	{
		Name:        "helm_config_show",
		Description: "Display the current helm.toml or helm-manager.toml configuration.",
		InputSchema: objectSchema(map[string]interface{}{}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			return []string{"config", "show"}, nil
		},
	},
	{
		Name:        "helm_doctor",
		Description: "Validate project setup — checks config, source of truth, board access, labels, notifications, and auth.",
		InputSchema: objectSchema(map[string]interface{}{}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			return []string{"doctor"}, nil
		},
	},
	{
		Name:        "helm_upgrade",
		Description: "Auto-fix issues found by doctor — creates missing labels, scaffolds devcontainer, initializes state directory.",
		InputSchema: objectSchema(map[string]interface{}{}, ""),
		Build: func(args map[string]interface{}) ([]string, error) {
			return []string{"upgrade"}, nil
		},
	},
}

// --- Schema helpers ---

func objectSchema(props map[string]interface{}, required ...string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
	// Filter empty strings from required.
	var reqs []string
	for _, r := range required {
		if r != "" {
			reqs = append(reqs, r)
		}
	}
	if len(reqs) > 0 {
		schema["required"] = reqs
	}
	return schema
}

func stringProp(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
}

func intProp(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": desc}
}

func boolProp(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "boolean", "description": desc}
}

// --- Flag building ---

type flagDef struct {
	key    string
	flag   string
	isBool bool
	isInt  bool
}

func appendFlags(cmd []string, args map[string]interface{}, defs []flagDef) []string {
	for _, d := range defs {
		val, ok := args[d.key]
		if !ok || val == nil {
			continue
		}
		if d.isBool {
			if b, ok := val.(bool); ok && b {
				cmd = append(cmd, d.flag)
			}
			continue
		}
		if d.isInt {
			if n, ok := getInt(args, d.key); ok {
				cmd = append(cmd, d.flag, strconv.Itoa(n))
			}
			continue
		}
		if s, ok := val.(string); ok && s != "" {
			cmd = append(cmd, d.flag, s)
		}
	}
	return cmd
}

func getString(args map[string]interface{}, key string) (string, bool) {
	v, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func getInt(args map[string]interface{}, key string) (int, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case string:
		i, err := strconv.Atoi(n)
		return i, err == nil
	}
	return 0, false
}

// suppress unused import warning
var _ = strings.TrimSpace
