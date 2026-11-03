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
	Store  bool   `json:"store,omitempty" jsonschema:"add a sqlstore package: SQL Repository using sqlx with context and WithTx"`
	Cache  bool   `json:"cache,omitempty" jsonschema:"add a cache package: in-memory Cache behind an interface"`
	HTTP   bool   `json:"http,omitempty" jsonschema:"add handler/httpapi: net/http routes, JSON bodies, RFC 9457 problem+json errors"`
	Queue  string `json:"queue,omitempty" jsonschema:"add a message consumer under handler/: nsq, kafka, nats (JetStream) or rabbitmq; omit for none"`
	Cron   bool   `json:"cron,omitempty" jsonschema:"add handler/cronjob: scheduled jobs that call the service"`
	DryRun bool   `json:"dry_run,omitempty" jsonschema:"only report the files that would be created"`
}

const toolDescription = `Scaffold a DDD-style Go domain package in one deterministic step: entity, domain errors and Service interface; a service implementation with table-driven tests; optional sqlstore (sqlx) and cache packages; and inbound handlers under handler/: httpapi (net/http, RFC 9457 errors), a message consumer (NSQ, Kafka, NATS JetStream or RabbitMQ) and cronjob (scheduled jobs).
Use it instead of hand-writing boilerplate when adding a new domain to a Go service. The output is gofmt'ed, compiles, passes its own tests, never overwrites an existing directory, and includes an AGENTS.md with the layer rules to follow when adding business logic.
After generating: run "go get" for any needs_deps, run "go test ./...", then build the feature following <name>/AGENTS.md.`

func newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "domain-generator", Version: version}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "generate_domain", Description: toolDescription},
		func(ctx context.Context, req *mcp.CallToolRequest, in generateInput) (*mcp.CallToolResult, generator.Result, error) {
			if !filepath.IsAbs(in.Dir) {
				return nil, generator.Result{}, errors.New("dir must be an absolute path")
			}
			res, err := generator.Generate(generator.Options{
				Name: in.Name, Dir: in.Dir, Module: in.Module,
				Store: in.Store, Cache: in.Cache, HTTP: in.HTTP, Queue: in.Queue, Cron: in.Cron, DryRun: in.DryRun,
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
