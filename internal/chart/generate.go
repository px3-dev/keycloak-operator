package chart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Generate writes a complete Helm chart to outputDir from parsed upstream data.
func Generate(u *Upstream, outputDir string, crdFiles []string) error {
	dirs := []string{
		filepath.Join(outputDir, "templates"),
		filepath.Join(outputDir, "crds"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	files := []struct {
		path string
		tmpl string
	}{
		{"Chart.yaml", chartYAMLTmpl},
		{"values.yaml", valuesYAMLTmpl},
		{".helmignore", helmignoreContent},
		{"templates/_helpers.tpl", helpersContent},
		{"templates/NOTES.txt", notesContent},
		{"templates/serviceaccount.yaml", serviceAccountTmpl},
		{"templates/deployment.yaml", deploymentTmpl},
		{"templates/service.yaml", serviceContent},
		{"templates/clusterrole.yaml", clusterRoleTmpl},
		{"templates/clusterrolebinding.yaml", clusterRoleBindingTmpl},
		{"templates/role.yaml", roleTmpl},
		{"templates/rolebinding.yaml", roleBindingTmpl},
	}

	funcMap := template.FuncMap{
		"indent": func(n int, s string) string {
			pad := strings.Repeat(" ", n)
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = pad + line
				}
			}
			return strings.Join(lines, "\n")
		},
	}

	for _, f := range files {
		if err := renderFile(filepath.Join(outputDir, f.path), f.tmpl, u, funcMap); err != nil {
			return fmt.Errorf("generating %s: %w", f.path, err)
		}
	}

	for _, src := range crdFiles {
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("reading CRD %s: %w", src, err)
		}
		dst := filepath.Join(outputDir, "crds", filepath.Base(src))
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("writing CRD %s: %w", dst, err)
		}
	}

	return nil
}

func renderFile(path, tmplStr string, data *Upstream, funcMap template.FuncMap) error {
	tmpl, err := template.New("").Delims("[[", "]]").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}
