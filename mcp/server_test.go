package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestToolsRegistered(t *testing.T) {
	tools := Tools()
	if len(tools) == 0 {
		t.Fatal("no tools registered")
	}
	// Verify sorted order.
	for i := 1; i < len(tools); i++ {
		if tools[i].Name < tools[i-1].Name {
			t.Errorf("tools not sorted: %s before %s", tools[i-1].Name, tools[i].Name)
		}
	}
}

func TestToolByName(t *testing.T) {
	tool, ok := ToolByName("helm_project_start")
	if !ok {
		t.Fatal("helm_project_start not found")
	}
	if tool.Description == "" {
		t.Error("tool has no description")
	}
}

func TestToolByNameMissing(t *testing.T) {
	_, ok := ToolByName("helm.does.not.exist")
	if ok {
		t.Error("expected not found")
	}
}

func TestProjectStartBuild(t *testing.T) {
	tool, _ := ToolByName("helm_project_start")
	args, err := tool.Build(map[string]interface{}{
		"issue": float64(42),
		"repo":  "maxbeizer/copilot-atc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.Join(args, " ")
	if !strings.Contains(got, "--issue 42") {
		t.Errorf("expected --issue 42, got: %s", got)
	}
	if !strings.Contains(got, "--repo maxbeizer/copilot-atc") {
		t.Errorf("expected --repo, got: %s", got)
	}
}

func TestProjectStartBuildMissingIssue(t *testing.T) {
	tool, _ := ToolByName("helm_project_start")
	_, err := tool.Build(map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing issue")
	}
}

func TestManagerPrepBuild(t *testing.T) {
	tool, _ := ToolByName("helm_manager_prep")
	args, err := tool.Build(map[string]interface{}{
		"handle": "sarah",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.Join(args, " ")
	if got != "manager prep sarah" {
		t.Errorf("expected 'manager prep sarah', got: %s", got)
	}
}

func TestServeInitialize(t *testing.T) {
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	reqBytes, _ := json.Marshal(req)

	var out bytes.Buffer
	err := Serve(bytes.NewReader(reqBytes), &out, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("serve error: %v", err)
	}

	var resp rpcResponse
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("unexpected protocol version: %v", result["protocolVersion"])
	}
}

func TestServeToolsList(t *testing.T) {
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	reqBytes, _ := json.Marshal(req)

	var out bytes.Buffer
	err := Serve(bytes.NewReader(reqBytes), &out, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("serve error: %v", err)
	}

	var resp rpcResponse
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	tools, ok := result["tools"].([]interface{})
	if !ok || len(tools) == 0 {
		t.Fatal("expected tools list")
	}
}

func TestAllToolsBuildable(t *testing.T) {
	// Ensure every tool's Build function works with empty args (may error, but shouldn't panic).
	for _, tool := range Tools() {
		t.Run(tool.Name, func(t *testing.T) {
			// Don't check error — some tools require args — just ensure no panic.
			tool.Build(map[string]interface{}{})
		})
	}
}
