package tplimpl

import (
	"errors"
	"github.com/sunwei/hugo-playground/tpl"
	htmltemplate "github.com/sunwei/hugo-playground/tpl/internal/go_templates/htmltemplate"
	"github.com/sunwei/hugo-playground/tpl/internal/go_templates/texttemplate"
	"github.com/sunwei/hugo-playground/tpl/internal/go_templates/texttemplate/parse"
)

const (
	templateUndefined templateType = iota
	templateShortcode
	templatePartial
)

func findTemplateIn(name string, in tpl.Template) (tpl.Template, bool) {
	in = unwrap(in)
	if text, ok := in.(*texttemplate.Template); ok {
		if templ := text.Lookup(name); templ != nil {
			return templ, true
		}
		return nil, false
	}
	if templ := in.(*htmltemplate.Template).Lookup(name); templ != nil {
		return templ, true
	}
	return nil, false
}

// TODO: transformers next time
func applyTemplateTransformers(
	t *templateState,
	lookupFn func(name string) *templateState) (*templateContext, error) {
	if t == nil {
		return nil, errors.New("expected template, but none provided")
	}

	c := newTemplateContext(t, lookupFn)

	return c, nil
}

type templateContext struct {
	visited          map[string]bool
	identityNotFound map[string]bool
	lookupFn         func(name string) *templateState

	// The last error encountered.
	err error

	// Set when we're done checking for config header.
	configChecked bool

	t *templateState

	// Store away the return node in partials.
	returnNode *parse.CommandNode
}

func newTemplateContext(
	t *templateState,
	lookupFn func(name string) *templateState) *templateContext {
	return &templateContext{
		t:                t,
		lookupFn:         lookupFn,
		visited:          make(map[string]bool),
		identityNotFound: make(map[string]bool),
	}
}

func getParseTree(templ tpl.Template) *parse.Tree {
	templ = unwrap(templ)
	if text, ok := templ.(*texttemplate.Template); ok {
		return text.Tree
	}
	return templ.(*htmltemplate.Template).Tree
}
