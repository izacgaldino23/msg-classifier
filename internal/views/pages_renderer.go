package views

import (
	"html/template"
	"strings"

	"github.com/gin-gonic/gin/render"
)

// PagesRenderer provides per-page template sets so a shared set of layouts and
// partials stays DRY while each page's page:title/page:content blocks stay
// isolated. Implements gin.HTMLRender.
type PagesRenderer struct {
	shared *template.Template
	pages  map[string]*template.Template
}

// NewPagesRenderer clones the shared set once per page file. Render names:
// "page" executes the base layout on that page's set, "page:content" executes
// the page:content block (htmx fragment). Any other name renders from the
// shared set (partials).
func NewPagesRenderer(shared *template.Template, pages map[string]string) (*PagesRenderer, error) {
	r := &PagesRenderer{shared: shared, pages: make(map[string]*template.Template, len(pages))}
	for name, file := range pages {
		set, err := shared.Clone()
		if err != nil {
			return nil, err
		}
		if _, err := set.ParseFiles(file); err != nil {
			return nil, err
		}
		r.pages[name] = set
	}
	return r, nil
}

// Instance implements gin.HTMLRender.
func (r *PagesRenderer) Instance(name string, data any) render.Render {
	base := strings.TrimSuffix(name, ":content")
	if set, ok := r.pages[base]; ok {
		exec := BaseTemplate
		if name != base {
			exec = PageContentTemplate
		}
		return render.HTML{Template: set, Name: exec, Data: data}
	}
	return render.HTML{Template: r.shared, Name: name, Data: data}
}
