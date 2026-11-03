package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/andryanduta/domain-generator/generator"
)

// generateInput is the generate_domain tool input. Field docs become the JSON schema the
// agent sees, so they are written for an LLM reader.
type generateInput struct {
	Name   string `json:"name" jsonschema:"domain name; must be a lowercase Go identifier such as order or user_profile"`
	Dir    string `json:"dir" jsonschema:"absolute path of the parent directory to create the domain in, usually inside a Go module"`
	Module string `json:"module,omitempty" jsonschema:"import path of dir; leave empty to detect it from the nearest go.mod"`
	Store  bool   `json:"store,omitempty" jsonschema:"add a SQL store layer (sqlx)"`
	Cache  bool   `json:"cache,omitempty" jsonschema:"add a cache store layer"`
	HTTP   bool   `json:"http,omitempty" jsonschema:"add an HTTP handler layer"`
	DryRun bool   `json:"dry_run,omitempty" jsonschema:"only report the files that would be created"`
}

const toolDescription = `Scaffold a DDD-style Go domain package (domain contract, service, and optional store, cache and HTTP layers) in one deterministic step.
Use it instead of hand-writing boilerplate when adding a new domain to a Go service. The output is gofmt'ed, compiles, never overwrites an existing directory, and includes an AGENTS.md describing the layer rules to follow when filling in business logic.
After generating: run "go get" for any needs_deps, then implement the TODOs.`

func newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "domain-generator", Version: version}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "generate_domain", Description: toolDescription},
		func(ctx context.Context, req *mcp.CallToolRequest, in generateInput) (*mcp.CallToolResult, generator.Result, error) {
			if !filepath.IsAbs(in.Dir) {
				return nil, generator.Result{}, errors.New("dir must be an absolute path")
			}
			res, err := generator.Generate(generator.Options{
				Name: in.Name, Dir: in.Dir, Module: in.Module,
				Store: in.Store, Cache: in.Cache, HTTP: in.HTTP, DryRun: in.DryRun,
			})
			if err != nil {
				return nil, res, fmt.Errorf("generate_domain: %w", err)
			}
			return nil, res, nil
		})
	return server
}

func runMCP(ctx context.Context) error {
	return newMCPServer().Run(ctx, &mcp.StdioTransport{})
}
