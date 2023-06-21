package provisioner

import (
	"embed"
	"html/template"
	"io/fs"
)

var (
	//go:embed tmpl/*
	files embed.FS
)

func loadTerraformTemplates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	tmplFiles, err := fs.ReadDir(files, "tmpl")
	if err != nil {
		return templates, err
	}

	for _, tmpl := range tmplFiles {
		if tmpl.IsDir() {
			continue
		}

		pt, err := template.ParseFS(files, "tmpl/"+tmpl.Name())
		if err != nil {
			return templates, err
		}

		templates[tmpl.Name()] = pt
	}

	return templates, nil
}
