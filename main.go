// Command domain-generator scaffolds a DDD-style Go domain package.
//
//	domain-generator -name order -store -http        # non-interactive
//	domain-generator -name order                     # asks which layers to add
//	domain-generator -name order -all -dry-run -json # plan only, machine-readable
//	domain-generator mcp                             # run as an MCP server for AI agents
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andryanduta/domain-generator/generator"
)

// version is set at release time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "mcp":
			if err := runMCP(context.Background()); err != nil {
				fmt.Fprintln(stderr, "mcp:", err)
				return 1
			}
			return 0
		case "version", "-version", "--version":
			fmt.Fprintln(stdout, version)
			return 0
		}
	}

	fs := flag.NewFlagSet("domain-generator", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		opts     generator.Options
		all      bool
		noPrompt bool
		asJSON   bool
	)
	fs.StringVar(&opts.Name, "name", "", "domain name: lowercase Go identifier, e.g. order (required)")
	fs.StringVar(&opts.Dir, "dir", ".", "parent directory to create the domain in")
	fs.StringVar(&opts.Module, "module", "", "import path of -dir (default: detected from go.mod)")
	fs.BoolVar(&opts.Store, "store", false, "add a SQL store layer (sqlx)")
	fs.BoolVar(&opts.Cache, "cache", false, "add a cache store layer")
	fs.BoolVar(&opts.HTTP, "http", false, "add an HTTP handler layer")
	fs.BoolVar(&all, "all", false, "add every layer (same as -store -cache -http)")
	fs.BoolVar(&noPrompt, "no-prompt", false, "never ask; only generate the layers given by flags")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "show what would be generated without writing files")
	fs.BoolVar(&asJSON, "json", false, "print the result as JSON (implies -no-prompt)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage:\n  domain-generator -name <domain> [flags]\n  domain-generator mcp\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if all {
		opts.Store, opts.Cache, opts.HTTP = true, true, true
	}

	layerFlagSet := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "store", "cache", "http", "all":
			layerFlagSet = true
		}
	})
	if !layerFlagSet && !noPrompt && !asJSON {
		askLayers(&opts, bufio.NewReader(stdin), stdout)
	}

	res, err := generator.Generate(opts)
	if asJSON {
		out := struct {
			generator.Result
			Error string `json:"error,omitempty"`
		}{Result: res}
		if err != nil {
			out.Error = err.Error()
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		if err != nil {
			return 1
		}
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	verb := "Created"
	if res.DryRun {
		verb = "Would create"
	}
	fmt.Fprintf(stdout, "%s %s (%s):\n", verb, res.Dir, res.ImportPath)
	for _, f := range res.Files {
		fmt.Fprintln(stdout, "  "+f)
	}
	if len(res.NeedsDeps) > 0 && !res.DryRun {
		fmt.Fprintf(stdout, "\nNext: go get %s\n", strings.Join(res.NeedsDeps, " "))
	}
	return 0
}

// askLayers asks which optional layers to add. It reads every answer from one reader, so
// piped input such as `printf 'y\nn\ny\n' | domain-generator -name x` works.
func askLayers(opts *generator.Options, in *bufio.Reader, out io.Writer) {
	opts.Store = ask(in, out, "Add a SQL store layer? (y/N): ")
	opts.Cache = ask(in, out, "Add a cache store layer? (y/N): ")
	opts.HTTP = ask(in, out, "Add an HTTP handler layer? (y/N): ")
}

func ask(in *bufio.Reader, out io.Writer, question string) bool {
	fmt.Fprint(out, question)
	answer, _ := in.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
