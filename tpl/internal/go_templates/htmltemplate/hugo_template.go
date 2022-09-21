package template

import (
	template "github.com/sunwei/hugo-playground/tpl/internal/go_templates/texttemplate"
)

var GoFuncs = funcMap

// Prepare returns a template ready for execution.
func (t *Template) Prepare() (*template.Template, error) {
	if err := t.escape(); err != nil {
		return nil, err
	}
	return t.text, nil
}

// See https://github.com/golang/go/issues/5884
func StripTags(html string) string {
	return stripTags(html)
}
