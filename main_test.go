package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/andryanduta/domain-generator/generator"
)

// Regression test: the old CLI created a new bufio.Reader per prompt group, so the first
// reader swallowed buffered answers and piped input never enabled the HTTP handler.
func TestAskLayersReadsAllPipedAnswers(t *testing.T) {
	var opts generator.Options
	askLayers(&opts, bufio.NewReader(strings.NewReader("y\nyes\nY\n")), &bytes.Buffer{})
	if !opts.Store || !opts.Cache || !opts.HTTP {
		t.Fatalf("got %+v, want all layers enabled", opts)
	}
}

func TestRunJSON(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{"-name", "order", "-dir", dir, "-module", "example.com/app", "-store", "-json"},
		strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	var res generator.Result
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
	}
	if res.ImportPath != "example.com/app/order" || len(res.NeedsDeps) != 1 {
		t.Fatalf("unexpected result %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "order", "store", "store.go")); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsInvalidName(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-name", "my-domain", "-dir", t.TempDir(), "-module", "x", "-no-prompt"},
		strings.NewReader(""), &stdout, &stderr); code == 0 {
		t.Fatal("expected a non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "invalid domain name") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestMCPGenerateDomain(t *testing.T) {
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	if _, err := newMCPServer().Connect(ctx, serverT, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 1 || tools.Tools[0].Name != "generate_domain" {
		t.Fatalf("ListTools = %+v, %v", tools, err)
	}

	dir := t.TempDir()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_domain",
		Arguments: map[string]any{"name": "order", "dir": dir, "module": "example.com/app", "http": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res.Content)
	}
	if _, err := os.Stat(filepath.Join(dir, "order", "handler", "http", "http.go")); err != nil {
		t.Fatal(err)
	}

	// Relative dirs are rejected: the server's working directory is not the agent's.
	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "generate_domain", Arguments: map[string]any{"name": "other", "dir": "relative/path"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected a tool error for a relative dir")
	}
}
