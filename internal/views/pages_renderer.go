package views

import (
	"html/template"

	"github.com/gin-gonic/gin/render"
)

// pageEntry binds a render name to the template set configured for it;
// full loads execute the shared "base" layout, htmx requests execute the
// page's "page:content" block.
type pageEntry struct {
	set      *template.Template
	execName string
}

// PagesRenderer serves per-page template sets, keeping the shared layout and
// partials DRY while isolating each page's page:title/page:content blocks.
// It implements gin.HTMLRender.
type PagesRenderer struct {
	shared  *template.Template
	entries map[string]pageEntry
}

// NewPagesRenderer clones shared for every page file and registers two render
// names per page: name (full page, executes BaseTemplate) and name + ":content"
// (htmx fragment, executes PageContentTemplate). Other names fall back to the
// shared set (partials).
func NewPagesRenderer(shared *template.Template, pages map[string]string) (*PagesRenderer, error) {
	r := &PagesRenderer{shared: shared, entries: make(map[string]pageEntry, len(pages)*2)}
	for name, file := range pages {
		set, err := shared.Clone()
		if err != nil {
			return nil, err
		}
		if _, err := set.ParseFiles(file); err != nil {
			return nil, err
		}
		r.entries[name] = pageEntry{set: set, execName: BaseTemplate}
		r.entries[name+":content"] = pageEntry{set: set, execName: PageContentTemplate}
	}
	return r, nil
}

// Instance implements gin.HTMLRender.
func (r *PagesRenderer) Instance(name string, data any) render.Render {
	if entry, ok := r.entries[name]; ok {
		return render.HTML{Template: entry.set, Name: entry.execName, Data: data}
	}
	return render.HTML{Template: r.shared, Name: name, Data: data}
}