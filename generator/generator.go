// Package generator scaffolds a DDD-style Go domain package: a domain contract, a service,
// and optional store, cache store and HTTP handler layers.
//
// Output is deterministic: the same Options always produce the same files, every generated
// .go file is gofmt'ed, and the generated code compiles. That makes it safe to call from
// scripts, CI and AI coding agents (see the mcp subcommand).
package generator

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

//go:embed all:template
var templateFS embed.FS

// Options describes the domain to generate.
type Options struct {
	// Name of the domain. It becomes the directory and the Go package name, so it must be a
	// lowercase Go identifier (for example "order" or "user_profile").
	Name string `json:"name"`
	// Dir is the parent directory the domain directory is created in. Defaults to ".".
	Dir string `json:"dir,omitempty"`
	// Module is the import path of Dir. When empty it is detected from the nearest go.mod.
	Module string `json:"module,omitempty"`
	// Store adds a SQL repository (sqlstore/) using sqlx.
	Store bool `json:"store,omitempty"`
	// Cache adds a cache (cache/).
	Cache bool `json:"cache,omitempty"`
	// HTTP adds an HTTP API (handler/httpapi/).
	HTTP bool `json:"http,omitempty"`
	// Queue adds a message consumer (handler/<queue>consumer/). One of Queues, or "" for none.
	Queue string `json:"queue,omitempty"`
	// Cron adds scheduled jobs (handler/cronjob/).
	Cron bool `json:"cron,omitempty"`
	// DryRun renders and validates everything but writes nothing.
	DryRun bool `json:"dry_run,omitempty"`
}

// Result reports what was (or, for a dry run, would be) generated.
type Result struct {
	Name       string   `json:"name"`
	Dir        string   `json:"dir"`
	ImportPath string   `json:"import_path"`
	Files      []string `json:"files"`
	DryRun     bool     `json:"dry_run"`
	// NeedsDeps lists modules the generated code imports that the caller may need to `go get`.
	NeedsDeps []string `json:"needs_deps,omitempty"`
}

var (
	// ErrExists is returned when the domain directory already exists.
	ErrExists = errors.New("domain directory already exists")
	// ErrNoModule is returned when no import path is given and no go.mod is found.
	ErrNoModule = errors.New("no go.mod found: run inside a Go module or pass a module import path")

	nameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// ValidateName reports whether name can be used as a domain directory and Go package name.
func ValidateName(name string) error {
	switch {
	case name == "":
		return errors.New("domain name is required")
	case !nameRE.MatchString(name):
		return fmt.Errorf("invalid domain name %q: use lowercase letters, digits and underscores, starting with a letter", name)
	case token.IsKeyword(name):
		return fmt.Errorf("invalid domain name %q: it is a Go keyword", name)
	case name == "main" || name == "internal" || name == "vendor" || name == "testdata":
		return fmt.Errorf("invalid domain name %q: reserved by the Go toolchain", name)
	case reservedNames[name]:
		return fmt.Errorf("invalid domain name %q: it clashes with a package the generated code imports", name)
	}
	return nil
}

// reservedNames are package names imported by the generated code (or its own subpackages);
// a domain with one of these names would not compile.
var reservedNames = map[string]bool{
	"service": true, "sqlstore": true, "cache": true, "httpapi": true,
	"context": true, "errors": true, "fmt": true, "strings": true, "strconv": true, "sync": true,
	"sql": true, "sqlx": true, "json": true, "http": true, "httptest": true, "slog": true,
	"io": true, "testing": true, "time": true, "atomic": true, "handler": true, "cronjob": true,
	"nsq": true, "nsqconsumer": true, "kafka": true, "kafkaconsumer": true, "jetstream": true,
	"nats": true, "natsconsumer": true, "amqp": true, "rabbitconsumer": true,
}

// queueSpec describes a supported message queue.
type queueSpec struct {
	pkg string // generated package, under handler/
	dep string // client module the generated code imports
}

// queues are the supported values of Options.Queue.
var queues = map[string]queueSpec{
	"nsq":      {pkg: "nsqconsumer", dep: "github.com/nsqio/go-nsq"},
	"kafka":    {pkg: "kafkaconsumer", dep: "github.com/segmentio/kafka-go"},
	"nats":     {pkg: "natsconsumer", dep: "github.com/nats-io/nats.go"},
	"rabbitmq": {pkg: "rabbitconsumer", dep: "github.com/rabbitmq/amqp091-go"},
}

// Queues returns the supported message queue names, sorted.
func Queues() []string {
	out := make([]string, 0, len(queues))
	for q := range queues {
		out = append(out, q)
	}
	sort.Strings(out)
	return out
}

// validateOptions checks everything except the name.
func validateOptions(opts Options) error {
	if _, ok := queues[opts.Queue]; opts.Queue != "" && !ok {
		return fmt.Errorf("unknown queue %q: use one of %s", opts.Queue, strings.Join(Queues(), ", "))
	}
	return nil
}

// DetectModule returns the import path of dir by finding the nearest go.mod above it.
func DetectModule(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for cur := abs; ; cur = filepath.Dir(cur) {
		data, err := os.ReadFile(filepath.Join(cur, "go.mod"))
		if err == nil {
			modPath := modulePath(data)
			if modPath == "" {
				return "", fmt.Errorf("%s: no module directive", filepath.Join(cur, "go.mod"))
			}
			rel, err := filepath.Rel(cur, abs)
			if err != nil {
				return "", err
			}
			return path.Join(modPath, filepath.ToSlash(rel)), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if filepath.Dir(cur) == cur {
			return "", ErrNoModule
		}
	}
}

type templateData struct {
	Name       string // package name, e.g. user_profile
	Type       string // exported entity type, e.g. UserProfile
	Var        string // unexported identifier prefix, e.g. userProfile
	Route      string // URL path segment, e.g. user-profiles
	Table      string // SQL table name, e.g. user_profiles
	ImportPath string
	Store      bool
	Cache      bool
	HTTP       bool
	Queue      string // nsq, kafka, nats, rabbitmq or ""
	QueuePkg   string // e.g. kafkaconsumer
	Cron       bool
}

func newTemplateData(opts Options, importPath string) templateData {
	parts := strings.Split(opts.Name, "_")
	var typ strings.Builder
	for _, p := range parts {
		if p != "" {
			typ.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	t := typ.String()
	plural := pluralize(opts.Name)
	return templateData{
		Name:       opts.Name,
		Type:       t,
		Var:        strings.ToLower(t[:1]) + t[1:],
		Route:      strings.ReplaceAll(plural, "_", "-"),
		Table:      plural,
		ImportPath: importPath,
		Store:      opts.Store,
		Cache:      opts.Cache,
		HTTP:       opts.HTTP,
		Queue:      opts.Queue,
		QueuePkg:   queues[opts.Queue].pkg,
		Cron:       opts.Cron,
	}
}

// pluralize is a small English pluralizer for route and table names; rename them in the
// generated code if it guesses wrong.
func pluralize(s string) string {
	switch {
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "z"),
		strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"):
		return s + "es"
	case len(s) > 1 && strings.HasSuffix(s, "y") && !strings.ContainsRune("aeiou", rune(s[len(s)-2])):
		return s[:len(s)-1] + "ies"
	default:
		return s + "s"
	}
}

// modulePath returns the module path declared in go.mod content, or "" if there is none.
func modulePath(gomod []byte) string {
	for _, line := range strings.Split(string(gomod), "\n") {
		line = strings.TrimSpace(line)
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		rest, ok := strings.CutPrefix(line, "module")
		if !ok || (rest != "" && rest[0] != ' ' && rest[0] != '	') {
			continue
		}
		return strings.Trim(strings.TrimSpace(rest), "\"`")
	}
	return ""
}

type fileSpec struct {
	tmpl    string
	out     string
	include func(templateData) bool
}

func always(templateData) bool { return true }

var files = []fileSpec{
	{"template/domain.go.tmpl", "{{.Name}}.go", always},
	{"template/AGENTS.md.tmpl", "AGENTS.md", always},
	{"template/service/service.go.tmpl", "service/service.go", always},
	{"template/service/service_test.go.tmpl", "service/service_test.go", always},
	{"template/sqlstore/sqlstore.go.tmpl", "sqlstore/sqlstore.go", func(d templateData) bool { return d.Store }},
	{"template/cache/cache.go.tmpl", "cache/cache.go", func(d templateData) bool { return d.Cache }},
	{"template/handler/httpapi/routes.go.tmpl", "handler/httpapi/routes.go", isHTTP},
	{"template/handler/httpapi/handlers.go.tmpl", "handler/httpapi/handlers.go", isHTTP},
	{"template/handler/httpapi/respond.go.tmpl", "handler/httpapi/respond.go", isHTTP},
	{"template/handler/httpapi/handlers_test.go.tmpl", "handler/httpapi/handlers_test.go", isHTTP},
	{"template/handler/consumer/message.go.tmpl", "handler/{{.QueuePkg}}/message.go", hasQueue},
	{"template/handler/consumer/message_test.go.tmpl", "handler/{{.QueuePkg}}/message_test.go", hasQueue},
	{"template/handler/{{.QueuePkg}}/consumer.go.tmpl", "handler/{{.QueuePkg}}/consumer.go", hasQueue},
	{"template/handler/{{.QueuePkg}}/consumer_test.go.tmpl", "handler/{{.QueuePkg}}/consumer_test.go", hasQueue},
	{"template/handler/cronjob/cronjob.go.tmpl", "handler/cronjob/cronjob.go", isCron},
	{"template/handler/cronjob/cronjob_test.go.tmpl", "handler/cronjob/cronjob_test.go", isCron},
}

func isHTTP(d templateData) bool   { return d.HTTP }
func hasQueue(d templateData) bool { return d.Queue != "" }
func isCron(d templateData) bool   { return d.Cron }

// expandPath fills the {{.Name}} and {{.QueuePkg}} placeholders in a file spec path.
func expandPath(p string, d templateData) string {
	return strings.NewReplacer("{{.Name}}", d.Name, "{{.QueuePkg}}", d.QueuePkg).Replace(p)
}

// Render returns the generated files (slash-separated relative path -> content) without
// touching disk. importPath is the import path of the new domain package.
func Render(opts Options, importPath string) (map[string][]byte, error) {
	if err := ValidateName(opts.Name); err != nil {
		return nil, err
	}
	if err := validateOptions(opts); err != nil {
		return nil, err
	}
	data := newTemplateData(opts, importPath)

	out := make(map[string][]byte)
	for _, f := range files {
		if !f.include(data) {
			continue
		}
		tmplPath := expandPath(f.tmpl, data)
		src, err := templateFS.ReadFile(tmplPath)
		if err != nil {
			return nil, err
		}
		t, err := template.New(tmplPath).Option("missingkey=error").Parse(string(src))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", tmplPath, err)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("render %s: %w", tmplPath, err)
		}
		content := buf.Bytes()
		name := expandPath(f.out, data)
		if strings.HasSuffix(name, ".go") {
			if content, err = format.Source(content); err != nil {
				return nil, fmt.Errorf("generated %s is not valid Go (template bug): %w", name, err)
			}
		}
		out[name] = content
	}
	return out, nil
}

// Generate renders the domain and writes it to <Dir>/<Name>. It never overwrites an existing
// directory, and writes into a temporary directory first so a failure leaves nothing behind.
func Generate(opts Options) (Result, error) {
	if err := ValidateName(opts.Name); err != nil {
		return Result{}, err
	}
	if err := validateOptions(opts); err != nil {
		return Result{}, err
	}
	if opts.Dir == "" {
		opts.Dir = "."
	}
	parent, err := filepath.Abs(opts.Dir)
	if err != nil {
		return Result{}, err
	}
	if st, err := os.Stat(parent); err != nil || !st.IsDir() {
		return Result{}, fmt.Errorf("output directory %s does not exist", parent)
	}

	module := opts.Module
	if module == "" {
		if module, err = DetectModule(parent); err != nil {
			return Result{}, err
		}
	}
	importPath := path.Join(module, opts.Name)
	target := filepath.Join(parent, opts.Name)

	rendered, err := Render(opts, importPath)
	if err != nil {
		return Result{}, err
	}

	res := Result{Name: opts.Name, Dir: target, ImportPath: importPath, DryRun: opts.DryRun}
	for name := range rendered {
		res.Files = append(res.Files, name)
	}
	sort.Strings(res.Files)
	if opts.Store {
		res.NeedsDeps = append(res.NeedsDeps, "github.com/jmoiron/sqlx")
	}
	if q, ok := queues[opts.Queue]; ok {
		res.NeedsDeps = append(res.NeedsDeps, q.dep)
	}

	if _, err := os.Lstat(target); err == nil {
		return res, fmt.Errorf("%w: %s", ErrExists, target)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return res, err
	}
	if opts.DryRun {
		return res, nil
	}

	tmp, err := os.MkdirTemp(parent, "."+opts.Name+"-*")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(tmp) // no-op after a successful rename

	for _, name := range res.Files {
		p := filepath.Join(tmp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return res, err
		}
		if err := os.WriteFile(p, rendered[name], 0o644); err != nil {
			return res, err
		}
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return res, err
	}
	if err := os.Rename(tmp, target); err != nil {
		return res, fmt.Errorf("move generated domain into place: %w", err)
	}
	return res, nil
}
