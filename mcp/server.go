package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/maxbeizer/gh-helm/internal/version"
)

// JSON-RPC types

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// MCP content types

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Serve starts the MCP server over stdio.
func Serve(in io.Reader, out io.Writer, errOut io.Writer) error {
	s := &server{
		in:  in,
		out: out,
		err: errOut,
	}
	return s.run()
}

type server struct {
	in  io.Reader
	out io.Writer
	err io.Writer
}

func (s *server) run() error {
	dec := json.NewDecoder(bufio.NewReader(s.in))
	enc := json.NewEncoder(s.out)

	for {
		var req rpcRequest
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("decode request: %w", err)
		}

		slog.Debug("mcp: received request", "method", req.Method, "id", req.ID)
		resp := s.handleRequest(&req)
		if resp != nil {
			if err := enc.Encode(resp); err != nil {
				return fmt.Errorf("encode response: %w", err)
			}
		}
	}
}

func (s *server) handleRequest(req *rpcRequest) *rpcResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		// Client acknowledgment — no response needed.
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return errorResponse(req.ID, -32601, "method not found", req.Method)
	}
}

func (s *server) handleInitialize(req *rpcRequest) *rpcResponse {
	return &rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{
					"listChanged": false,
				},
			},
			"serverInfo": map[string]any{
				"name":    "gh-helm",
				"version": version.Version,
			},
		},
	}
}

func (s *server) handleToolsList(req *rpcRequest) *rpcResponse {
	defs := Tools()
	tools := make([]map[string]any, len(defs))
	for i, t := range defs {
		tools[i] = map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		}
	}
	return &rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]any{"tools": tools},
	}
}

func (s *server) handleToolsCall(req *rpcRequest) *rpcResponse {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "invalid params", err.Error())
	}

	tool, ok := ToolByName(params.Name)
	if !ok {
		return errorResponse(req.ID, -32602, "unknown tool", params.Name)
	}

	slog.Debug("mcp: calling tool", "name", params.Name, "args", params.Arguments)

	args, err := tool.Build(params.Arguments)
	if err != nil {
		return &rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: toolResult{
				Content: []textContent{{Type: "text", Text: fmt.Sprintf("Error building command: %s", err)}},
				IsError: true,
			},
		}
	}

	// Always request JSON output.
	args = append(args, "--json")

	output, err := runGhHelm(args)
	if err != nil {
		slog.Debug("mcp: tool error", "name", params.Name, "error", err)
		return &rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: toolResult{
				Content: []textContent{{Type: "text", Text: fmt.Sprintf("Error: %s\nOutput: %s", err, string(output))}},
				IsError: true,
			},
		}
	}

	// Try to return structured JSON; fall back to plain text.
	text := strings.TrimSpace(string(output))
	slog.Debug("mcp: tool success", "name", params.Name, "outputLen", len(text))

	return &rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: toolResult{
			Content: []textContent{{Type: "text", Text: text}},
		},
	}
}

func runGhHelm(args []string) ([]byte, error) {
	ghArgs := append([]string{"helm"}, args...)
	cmd := exec.Command("gh", ghArgs...)
	cmd.Env = os.Environ()
	return cmd.CombinedOutput()
}

func errorResponse(id any, code int, message string, data any) *rpcResponse {
	return &rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message, Data: data},
	}
}
