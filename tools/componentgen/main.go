// Command componentgen writes what `viva add` installs from the reference
// components, so the installable recipes cannot drift from the package.
//
//	go run ./tools/componentgen          # from the repository root
//	go run ./tools/componentgen -check   # fail if the generated files are stale
//
// Every non-test file of the components package except doc.go is a recipe. Its
// installable source is that file with the package clause replaced by
// "package {{PACKAGE}}", written to cmd/viva/templates/<file>.go.txt. The
// recipes a file calls into (a package-level identifier defined in another file)
// are found by type checking the package and written to cmd/viva/components_gen.go,
// which `viva add` uses to install a recipe together with what it needs.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type recipe struct {
	name     string // the name given to viva add, e.g. "metric-card"
	file     string // components/metric_card.go
	requires []string
}

func main() {
	root := flag.String("root", ".", "repository root")
	check := flag.Bool("check", false, "report stale generated files instead of writing them")
	flag.Parse()
	if err := run(*root, *check); err != nil {
		fmt.Fprintln(os.Stderr, "componentgen:", err)
		os.Exit(1)
	}
}

func run(root string, check bool) error {
	if err := os.Chdir(root); err != nil {
		return err
	}
	recipes, sources, err := analyze("components")
	if err != nil {
		return err
	}
	want := map[string][]byte{}
	for _, r := range recipes {
		source := sources[r.file]
		if !bytes.Contains(source, []byte("package components\n")) {
			return fmt.Errorf("%s has no package clause to replace", r.file)
		}
		want[filepath.Join("cmd", "viva", "templates", r.file+".txt")] =
			bytes.Replace(source, []byte("package components\n"), []byte("package {{PACKAGE}}\n"), 1)
	}
	table, err := table(recipes)
	if err != nil {
		return err
	}
	want[filepath.Join("cmd", "viva", "components_gen.go")] = table

	stale := compare(want, filepath.Join("cmd", "viva", "templates"))
	if check {
		if len(stale) > 0 {
			return fmt.Errorf("generated files are stale; run `go run ./tools/componentgen`:\n  %s", strings.Join(stale, "\n  "))
		}
		return nil
	}
	for _, path := range stale {
		if data, ok := want[path]; ok {
			if err := os.WriteFile(path, data, 0o644); err != nil {
				return err
			}
			continue
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

// compare returns the paths in want whose content on disk differs, plus files in
// templates that no recipe produces any more.
func compare(want map[string][]byte, templates string) []string {
	var stale []string
	for path, data := range want {
		if have, err := os.ReadFile(path); err != nil || !bytes.Equal(have, data) {
			stale = append(stale, path)
		}
	}
	if entries, err := os.ReadDir(templates); err == nil {
		for _, entry := range entries {
			path := filepath.Join(templates, entry.Name())
			if _, ok := want[path]; !ok && strings.HasSuffix(entry.Name(), ".go.txt") {
				stale = append(stale, path)
			}
		}
	}
	sort.Strings(stale)
	return stale
}

// analyze type checks the package in dir and returns one recipe per file, with
// the files it depends on.
func analyze(dir string) ([]recipe, map[string][]byte, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, nil, err
	}
	fset := token.NewFileSet()
	var files []*ast.File
	sources := map[string][]byte{}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		file, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, file)
		sources[filepath.Base(path)] = source
	}
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
	config := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	pkg, err := config.Check("github.com/vivaui/viva/components", fset, files, info)
	if err != nil {
		return nil, nil, err
	}

	fileOf := func(pos token.Pos) string { return filepath.Base(fset.Position(pos).Filename) }
	needs := map[string]map[string]bool{}
	for ident, object := range info.Uses {
		if object.Pkg() != pkg || object.Parent() != pkg.Scope() {
			continue
		}
		from, to := fileOf(ident.Pos()), fileOf(object.Pos())
		if from == to {
			continue
		}
		if needs[from] == nil {
			needs[from] = map[string]bool{}
		}
		needs[from][to] = true
	}

	var recipes []recipe
	for file := range sources {
		if file == "doc.go" {
			continue
		}
		r := recipe{name: recipeName(file), file: file}
		for to := range needs[file] {
			if to != "doc.go" {
				r.requires = append(r.requires, recipeName(to))
			}
		}
		sort.Strings(r.requires)
		recipes = append(recipes, r)
	}
	sort.Slice(recipes, func(i, j int) bool { return recipes[i].name < recipes[j].name })
	return recipes, sources, nil
}

func recipeName(file string) string {
	return strings.ReplaceAll(strings.TrimSuffix(file, ".go"), "_", "-")
}

func table(recipes []recipe) ([]byte, error) {
	var out bytes.Buffer
	out.WriteString("// Code generated by tools/componentgen; DO NOT EDIT.\n\npackage main\n\n")
	out.WriteString("// componentFiles maps the name given to `viva add` to its template.\n")
	out.WriteString("var componentFiles = map[string]string{\n")
	for _, r := range recipes {
		fmt.Fprintf(&out, "\t%q: %q,\n", r.name, r.file+".txt")
	}
	out.WriteString("}\n\n")
	out.WriteString("// componentRequires lists the recipes a recipe's source calls into. `viva add`\n")
	out.WriteString("// installs them along with it, so a recipe added on its own compiles.\n")
	out.WriteString("var componentRequires = map[string][]string{\n")
	for _, r := range recipes {
		if len(r.requires) == 0 {
			continue
		}
		quoted := make([]string, len(r.requires))
		for i, name := range r.requires {
			quoted[i] = fmt.Sprintf("%q", name)
		}
		fmt.Fprintf(&out, "\t%q: {%s},\n", r.name, strings.Join(quoted, ", "))
	}
	out.WriteString("}\n")
	return format.Source(out.Bytes())
}
