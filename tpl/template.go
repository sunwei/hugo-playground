package tpl

import (
	"context"
	bp "github.com/sunwei/hugo-playground/bufferpool"
	"github.com/sunwei/hugo-playground/output"
	htmltemplate "github.com/sunwei/hugo-playground/tpl/internal/go_templates/htmltemplate"
	"github.com/sunwei/hugo-playground/tpl/internal/go_templates/texttemplate"
	"io"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

// TemplateManager manages the collection of templates.
type TemplateManager interface {
	TemplateHandler
	TemplateFuncGetter
	AddTemplate(name, tpl string) error
	MarkReady() error
}

// TemplateHandler finds and executes templates.
type TemplateHandler interface {
	TemplateFinder
	Execute(t Template, wr io.Writer, data any) error
	ExecuteWithContext(ctx context.Context, t Template, wr io.Writer, data any) error
	LookupLayout(d output.LayoutDescriptor, f output.Format) (Template, bool, error)
	HasTemplate(name string) bool
}

// Template is the common interface between text/template and html/template.
type Template interface {
	Name() string
	Prepare() (*texttemplate.Template, error)
}

// TemplateFinder finds templates.
type TemplateFinder interface {
	TemplateLookup
}

// TemplateFuncGetter allows to find a template func by name.
type TemplateFuncGetter interface {
	GetFunc(name string) (reflect.Value, bool)
}

// TemplateParseFinder provides both parsing and finding.
type TemplateParseFinder interface {
	TemplateParser
	TemplateFinder
}

// TemplateParser is used to parse ad-hoc templates, e.g. in the Resource chain.
type TemplateParser interface {
	Parse(name, tpl string) (Template, error)
}

// TemplateVariants describes the possible variants of a template.
// All of these may be empty.
type TemplateVariants struct {
	Language     string
	OutputFormat output.Format
}

type TemplateLookup interface {
	Lookup(name string) (Template, bool)
}

const hugoNewLinePlaceholder = "___hugonl_"

var (
	stripHTMLReplacerPre = strings.NewReplacer("\n", " ", "</p>", hugoNewLinePlaceholder, "<br>", hugoNewLinePlaceholder, "<br />", hugoNewLinePlaceholder)
	whitespaceRe         = regexp.MustCompile(`\s+`)
)

// StripHTML strips out all HTML tags in s.
func StripHTML(s string) string {
	// Shortcut strings with no tags in them
	if !strings.ContainsAny(s, "<>") {
		return s
	}

	pre := stripHTMLReplacerPre.Replace(s)
	preReplaced := pre != s

	s = htmltemplate.StripTags(pre)

	if preReplaced {
		s = strings.ReplaceAll(s, hugoNewLinePlaceholder, "\n")
	}

	var wasSpace bool
	b := bp.GetBuffer()
	defer bp.PutBuffer(b)
	for _, r := range s {
		isSpace := unicode.IsSpace(r)
		if !(isSpace && wasSpace) {
			b.WriteRune(r)
		}
		wasSpace = isSpace
	}

	if b.Len() > 0 {
		s = b.String()
	}

	return s
}
