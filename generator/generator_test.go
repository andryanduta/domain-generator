package generator

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	for _, name := range []string{"order", "user_profile", "v2orders", "a"} {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"", "Order", "my-domain", "../escaped", "a/b", "1order", "_x", "func", "type", "main", "internal", "with space"} {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", name)
		}
	}
}

func TestModulePath(t *testing.T) {
	cases := map[string]string{
		"module example.com/app\n\ngo 1.22\n":     "example.com/app",
		"// comment\nmodule \"example.com/q\"\n":  "example.com/q",
		"module example.com/c // trailing note\n": "example.com/c",
		"go 1.22\n":                             "",
		"modules example.com/not-a-directive\n": "",
	}
	for in, want := range cases {
		if got := modulePath([]byte(in)); got != want {
			t.Errorf("modulePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetectModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.22\n")
	nested := filepath.Join(root, "internal", "domains")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	for dir, want := range map[string]string{root: "example.com/app", nested: "example.com/app/internal/domains"} {
		got, err := DetectModule(dir)
		if err != nil || got != want {
			t.Errorf("DetectModule(%s) = %q, %v; want %q", dir, got, err, want)
		}
	}

	if _, err := DetectModule(t.TempDir()); !errors.Is(err, ErrNoModule) {
		t.Errorf("DetectModule(no go.mod) error = %v, want ErrNoModule", err)
	}
}

func TestGenerateFiles(t *testing.T) {
	cases := []struct {
		name string
		opts Options
		want []string
	}{
		{"core only", Options{}, []string{"AGENTS.md", "order.go", "service/service.go"}},
		{"all layers", Options{Store: true, Cache: true, HTTP: true}, []string{
			"AGENTS.md", "handler/http/error.go", "handler/http/foo_handler.go", "handler/http/http.go",
			"handler/http/util.go", "order.go", "service/service.go", "store/cachestore.go", "store/store.go",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			opts := tc.opts
			opts.Name, opts.Dir, opts.Module = "order", dir, "example.com/app"
			res, err := Generate(opts)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(res.Files, tc.want) {
				t.Fatalf("files = %v, want %v", res.Files, tc.want)
			}
			if res.ImportPath != "example.com/app/order" {
				t.Errorf("import path = %q", res.ImportPath)
			}
			for _, f := range tc.want {
				if _, err := os.Stat(filepath.Join(dir, "order", f)); err != nil {
					t.Errorf("missing %s: %v", f, err)
				}
			}
			assertNoLeftovers(t, dir)
		})
	}
}

func TestGenerateDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	res, err := Generate(Options{Name: "order", Dir: dir, Module: "example.com/app", Store: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || len(res.Files) == 0 {
		t.Fatalf("unexpected result %+v", res)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("dry run wrote %d entries", len(entries))
	}
}

func TestGenerateRefusesExistingDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "order"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Generate(Options{Name: "order", Dir: dir, Module: "example.com/app"})
	if !errors.Is(err, ErrExists) {
		t.Fatalf("error = %v, want ErrExists", err)
	}
}

func TestGenerateRejectsTraversal(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "sub")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(Options{Name: "../escaped", Dir: dir, Module: "example.com/app"}); err == nil {
		t.Fatal("expected an error for a path-traversal name")
	}
	if _, err := os.Stat(filepath.Join(parent, "escaped")); err == nil {
		t.Fatal("files were written outside the output directory")
	}
}

func TestRenderedGoIsFormattedAndHasNoPlaceholders(t *testing.T) {
	out, err := Render(Options{Name: "order", Store: true, Cache: true, HTTP: true}, "example.com/app/order")
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range out {
		s := string(content)
		for _, bad := range []string{"yourreponame", "<%=", "{{", "\r\n"} {
			if strings.Contains(s, bad) {
				t.Errorf("%s contains %q", name, bad)
			}
		}
	}
}

// TestGeneratedCodeCompiles generates every layer combination into a throwaway module and
// runs `go vet` on it. sqlx is replaced by a local stub so the test works offline.
func TestGeneratedCodeCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles generated code with the go tool")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool not found")
	}

	for _, combo := range layerCombos() {
		t.Run(combo.label(), func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "go.mod"), strings.Join([]string{
				"module example.com/app", "", "go 1.22", "",
				"require github.com/jmoiron/sqlx v1.4.0", "",
				"replace github.com/jmoiron/sqlx => ./_stub/sqlx", "",
			}, "\n"))
			writeFile(t, filepath.Join(root, "_stub", "sqlx", "go.mod"), "module github.com/jmoiron/sqlx\n\ngo 1.22\n")
			writeFile(t, filepath.Join(root, "_stub", "sqlx", "sqlx.go"),
				"package sqlx\n\nimport \"database/sql\"\n\n// Tx stubs sqlx.Tx.\ntype Tx struct{ *sql.Tx }\n")

			opts := combo.opts
			opts.Name, opts.Dir = "order", root
			if _, err := Generate(opts); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(goBin, "vet", "./order/...")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("generated code does not compile:\n%s", out)
			}
		})
	}
}

type combo struct{ opts Options }

func (c combo) label() string {
	var parts []string
	if c.opts.Store {
		parts = append(parts, "store")
	}
	if c.opts.Cache {
		parts = append(parts, "cache")
	}
	if c.opts.HTTP {
		parts = append(parts, "http")
	}
	if len(parts) == 0 {
		return "core"
	}
	return strings.Join(parts, "+")
}

func layerCombos() []combo {
	var out []combo
	for i := 0; i < 8; i++ {
		out = append(out, combo{Options{Store: i&1 != 0, Cache: i&2 != 0, HTTP: i&4 != 0}})
	}
	return out
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertNoLeftovers(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			t.Errorf("temporary directory left behind: %s", e.Name())
		}
	}
}
