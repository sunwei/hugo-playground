package page

import (
	"fmt"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/related"
	"github.com/sunwei/hugo-playground/resources/resource"
	"github.com/sunwei/hugo-playground/source"
	"github.com/sunwei/hugo-playground/tpl"
	"html/template"
)

// Page is the core interface in Hugo.
type Page interface {
	ContentProvider
	TableOfContentsProvider
	PageWithoutContent
}

// ContentProvider provides the content related values for a Page.
type ContentProvider interface {
	Content() (any, error)

	// Plain returns the Page Content stripped of HTML markup.
	Plain() string

	// PlainWords returns a string slice from splitting Plain using https://pkg.go.dev/strings#Fields.
	PlainWords() []string

	// Summary returns a generated summary of the content.
	// The breakpoint can be set manually by inserting a summary separator in the source file.
	Summary() template.HTML

	// Truncated returns whether the Summary  is truncated or not.
	Truncated() bool

	// FuzzyWordCount returns the approximate number of words in the content.
	FuzzyWordCount() int

	// WordCount returns the number of words in the content.
	WordCount() int

	// ReadingTime returns the reading time based on the length of plain text.
	ReadingTime() int

	// Len returns the length of the content.
	Len() int
}

// TableOfContentsProvider provides the table of contents for a Page.
type TableOfContentsProvider interface {
	TableOfContents() template.HTML
}

// RawContentProvider provides the raw, unprocessed content of the page.
type RawContentProvider interface {
	RawContent() string
}

// PageWithoutContent is the Page without any of the content methods.
type PageWithoutContent interface {
	resource.Resource
	PageMetaProvider

	// FileProvider For pages backed by a file.
	FileProvider

	// OutputFormatsProvider Output formats
	OutputFormatsProvider

	TreeProvider

	SitesProvider
	identity.Provider
	PaginatorProvider
	PageRenderProvider
	AlternativeOutputFormatsProvider
}

// PageMetaProvider provides page metadata, typically provided via front matter.
type PageMetaProvider interface {
	// Dated The 4 page dates
	resource.Dated

	// Description A configured description.
	Description() string

	// IsHome returns whether this is the home page.
	IsHome() bool

	// Kind The Page Kind. One of page, home, section, taxonomy, term.
	Kind() string

	// Layout The configured layout to use to render this page. Typically set in front matter.
	Layout() string

	// LinkTitle The title used for links.
	LinkTitle() string

	// IsNode returns whether this is an item of one of the list types in Hugo,
	// i.e. not a regular content
	IsNode() bool

	// IsPage returns whether this is a regular content
	IsPage() bool

	// Param looks for a param in Page and then in Site config.
	Param(key any) (any, error)

	// Path gets the relative path, including file name and extension if relevant,
	// to the source of this Page. It will be relative to any content root.
	Path() string

	// Pathc This is just a temporary bridge method. Use Path in templates.
	// Pathc is for internal usage only.
	Pathc() string

	// Slug The slug, typically defined in front matter.
	Slug() string

	// IsSection returns whether this is a section
	IsSection() bool

	// Section returns the first path element below the content root.
	Section() string

	// SectionsEntries Returns a slice of sections (directories if it's a file) to this
	// Page.
	SectionsEntries() []string

	// SectionsPath is SectionsEntries joined with a /.
	SectionsPath() string

	// Type is a discriminator used to select layouts etc. It is typically set
	// in front matter, but will fall back to the root section.
	Type() string

	// Weight The configured weight, used as the first sort value in the default
	// page sort if non-zero.
	Weight() int
}

// FileProvider provides the source file.
type FileProvider interface {
	File() source.File
}

// OutputFormatsProvider provides the OutputFormats of a Page.
type OutputFormatsProvider interface {
	OutputFormats() OutputFormats
}

// PageRenderProvider provides a way for a Page to render content.
type PageRenderProvider interface {
	RenderString(args ...any) (template.HTML, error)
}

// ChildCareProvider provides accessors to child resources.
type ChildCareProvider interface {
	Pages() Pages

	// RegularPages returns a list of pages of kind 'Page'.
	// In Hugo 0.57 we changed the Pages method so it returns all page
	// kinds, even sections. If you want the old behaviour, you can
	// use RegularPages.
	RegularPages() Pages

	// RegularPagesRecursive returns all regular pages below the current
	// section.
	RegularPagesRecursive() Pages

	Resources() resource.Resources
}

// GetPageProvider provides the GetPage method.
type GetPageProvider interface {
	// GetPage looks up a page for the given ref.
	//    {{ with .GetPage "blog" }}{{ .Title }}{{ end }}
	//
	// This will return nil when no page could be found, and will return
	// an error if the ref is ambiguous.
	GetPage(ref string) (Page, error)

	// GetPageWithTemplateInfo is for internal use only.
	GetPageWithTemplateInfo(info tpl.Info, ref string) (Page, error)
}

// InSectionPositioner provides section navigation.
type InSectionPositioner interface {
	NextInSection() Page
	PrevInSection() Page
}

// Positioner provides next/prev navigation.
type Positioner interface {
	Next() Page
	Prev() Page

	// Deprecated: Use Prev. Will be removed in Hugo 0.57
	PrevPage() Page

	// Deprecated: Use Next. Will be removed in Hugo 0.57
	NextPage() Page
}

// RelatedKeywordsProvider allows a Page to be indexed.
type RelatedKeywordsProvider interface {
	// Make it indexable as a related.Document
	// RelatedKeywords is meant for internal usage only.
	RelatedKeywords(cfg related.IndexConfig) ([]related.Keyword, error)
}

// RefProvider provides the methods needed to create reflinks to pages.
type RefProvider interface {
	Ref(argsm map[string]any) (string, error)

	// RefFrom is for internal use only.
	RefFrom(argsm map[string]any, source any) (string, error)

	RelRef(argsm map[string]any) (string, error)

	// RefFrom is for internal use only.
	RelRefFrom(argsm map[string]any, source any) (string, error)
}

// ShortcodeInfoProvider provides info about the shortcodes in a Page.
type ShortcodeInfoProvider interface {
	// HasShortcode return whether the page has a shortcode with the given name.
	// This method is mainly motivated with the Hugo Docs site's need for a list
	// of pages with the `todo` shortcode in it.
	HasShortcode(name string) bool
}

// SitesProvider provide accessors to get sites.
type SitesProvider interface {
	Site() Site
	Sites() Sites
}

// TreeProvider provides section tree navigation.
type TreeProvider interface {

	// IsAncestor returns whether the current page is an ancestor of the given
	// Note that this method is not relevant for taxonomy lists and taxonomy terms pages.
	IsAncestor(other any) (bool, error)

	// CurrentSection returns the page's current section or the page itself if home or a section.
	// Note that this will return nil for pages that is not regular, home or section pages.
	CurrentSection() Page

	// IsDescendant returns whether the current page is a descendant of the given
	// Note that this method is not relevant for taxonomy lists and taxonomy terms pages.
	IsDescendant(other any) (bool, error)

	// FirstSection returns the section on level 1 below home, e.g. "/docs".
	// For the home page, this will return itself.
	FirstSection() Page

	// InSection returns whether the given page is in the current section.
	// Note that this will always return false for pages that are
	// not either regular, home or section pages.
	InSection(other any) (bool, error)

	// Parent returns a section's parent section or a page's section.
	// To get a section's subsections, see Page's Sections method.
	Parent() Page

	// Sections returns this section's subsections, if any.
	// Note that for non-sections, this method will always return an empty list.
	Sections() Pages

	// Page returns a reference to the Page itself, kept here mostly
	// for legacy reasons.
	Page() Page
}

// PagesFactory somehow creates some Pages.
// We do a lot of lazy Pages initialization in Hugo, so we need a type.
type PagesFactory func() Pages

// AlternativeOutputFormatsProvider provides alternative output formats for a
// Page.
type AlternativeOutputFormatsProvider interface {
	// AlternativeOutputFormats gives the alternative output formats for the
	// current output.
	// Note that we use the term "alternative" and not "alternate" here, as it
	// does not necessarily replace the other format, it is an alternative representation.
	AlternativeOutputFormats() OutputFormats
}

// ToPages tries to convert seq into Pages.
func ToPages(seq any) (Pages, error) {
	if seq == nil {
		return Pages{}, nil
	}

	switch v := seq.(type) {
	case Pages:
		return v, nil
	case *Pages:
		return *(v), nil
	case WeightedPages:
		return v.Pages(), nil
	case PageGroup:
		return v.Pages, nil
	case []Page:
		pages := make(Pages, len(v))
		for i, vv := range v {
			pages[i] = vv
		}
		return pages, nil
	case []any:
		pages := make(Pages, len(v))
		success := true
		for i, vv := range v {
			p, ok := vv.(Page)
			if !ok {
				success = false
				break
			}
			pages[i] = p
		}
		if success {
			return pages, nil
		}
	}

	return nil, fmt.Errorf("cannot convert type %T to Pages", seq)
}
