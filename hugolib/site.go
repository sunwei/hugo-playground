package hugolib

import (
	"fmt"
	"github.com/spf13/afero"
	bp "github.com/sunwei/hugo-playground/bufferpool"
	"github.com/sunwei/hugo-playground/common/maps"
	"github.com/sunwei/hugo-playground/common/text"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/deps"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/langs"
	"github.com/sunwei/hugo-playground/lazy"
	"github.com/sunwei/hugo-playground/markup/converter"
	"github.com/sunwei/hugo-playground/media"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/publisher"
	"github.com/sunwei/hugo-playground/resources/page"
	"github.com/sunwei/hugo-playground/source"
	"github.com/sunwei/hugo-playground/tpl"
	"html/template"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Site contains all the information relevant for constructing a static
// site.  The basic flow of information is as follows:
//
// 1. A list of Files is parsed and then converted into Pages.
//
//  2. Pages contain sections (based on the file they were generated from),
//     aliases and slugs (included in a pages frontmatter) which are the
//     various targets that will get generated.  There will be canonical
//     listing.  The canonical path can be overruled based on a pattern.
//
//  3. Taxonomies are created via configuration and will present some aspect of
//     the final page and typically a perm url.
//
//  4. All Pages are passed through a template based on their desired
//     layout based on numerous different elements.
//
// 5. The entire collection of files is written to disk.
type Site struct {
	language   *langs.Language
	siteBucket *pagesMapBucket

	// Output formats defined in site config per Page Kind, or some defaults
	// if not set.
	// Output formats defined in Page front matter will override these.
	outputFormats map[string]output.Formats

	// All the output formats and media types available for this site.
	// These values will be merged from the Hugo defaults, the site config and,
	// finally, the language settings.
	outputFormatsConfig output.Formats
	mediaTypesConfig    media.Types

	// We render each site for all the relevant output formats in serial with
	// this rendering context pointing to the current one.
	rc      *siteRenderingContext
	siteCfg siteConfigHolder

	// The func used to title case titles.
	titleFunc func(s string) string

	// newSite with above infos

	// The owning container. When multiple languages, there will be multiple
	// sites .
	h *HugoSites

	*PageCollections

	Sections Taxonomy
	Info     *SiteInfo

	// The output formats that we need to render this site in. This slice
	// will be fixed once set.
	// This will be the union of Site.Pages' outputFormats.
	// This slice will be sorted.
	renderFormats output.Formats

	// Logger etc.
	*deps.Deps `json:"-"`

	siteRefLinker

	publisher publisher.Publisher

	// Shortcut to the home page. Note that this may be nil if
	// home page, for some odd reason, is disabled.
	home *pageState
}

type siteRenderingContext struct {
	output.Format
}

type siteRefLinker struct {
	s *Site

	notFoundURL string
}

type SiteInfo struct {
	title string

	relativeURLs bool

	owner *HugoSites
	s     *Site
}

// newSite creates a new site with the given configuration.
func newSite(cfg deps.DepsCfg) (*Site, error) {
	var (
		mediaTypesConfig    []map[string]any
		outputFormatsConfig []map[string]any

		siteOutputFormatsConfig output.Formats
		siteMediaTypesConfig    media.Types
		err                     error
	)

	// [{toml}, {html}, {markdown}, {plain}]
	siteMediaTypesConfig, err = media.DecodeTypes(mediaTypesConfig...)
	if err != nil {
		return nil, err
	}

	// [{HTML}, {JSON}, {MARKDOWN}]
	siteOutputFormatsConfig, err = output.DecodeFormats(siteMediaTypesConfig, outputFormatsConfig...)

	if err != nil {
		return nil, err
	}

	// Site output formats source
	outputFormats, err := createSiteOutputFormats(siteOutputFormatsConfig, nil, true)

	if err != nil {
		return nil, err
	}

	// KindTaxonomy, KindTerm like section title
	titleFunc := helpers.GetTitleFunc("")

	siteConfig := siteConfigHolder{
		timeout: 30 * time.Second, // page content output init timeout
	}

	var siteBucket *pagesMapBucket

	s := &Site{
		language:   cfg.Language,
		siteBucket: siteBucket,

		outputFormats:       outputFormats,
		outputFormatsConfig: siteOutputFormatsConfig,
		mediaTypesConfig:    siteMediaTypesConfig,

		siteCfg:   siteConfig,
		titleFunc: titleFunc,

		rc: &siteRenderingContext{output.HTMLFormat},
	}

	return s, nil
}

type siteConfigHolder struct {
	timeout time.Duration
}

func (s *Site) initializeSiteInfo() error {
	// Assemble dependencies to be used in hugo.Deps.
	s.Info = &SiteInfo{
		title:        "title",
		relativeURLs: s.Cfg.GetBool("relativeURLs"),
		owner:        s.h,
		s:            s,
	}

	return nil
}

func (s *SiteInfo) Pages() page.Pages {
	return s.s.Pages()
}

func (s *SiteInfo) RegularPages() page.Pages {
	return s.s.RegularPages()
}

func (s *SiteInfo) AllPages() page.Pages {
	return s.s.AllPages()
}

func (s *SiteInfo) AllRegularPages() page.Pages {
	return s.s.AllRegularPages()
}

func (s *SiteInfo) Title() string {
	return s.title
}

func (s *SiteInfo) Site() page.Site {
	return s
}

func (s *SiteInfo) Data() map[string]any {
	return s.s.h.Data()
}

// Current returns the currently rendered Site.
// If that isn't set yet, which is the situation before we start rendering,
// if will return the Site itself.
func (s *SiteInfo) Current() page.Site {
	if s.s.h.currentSite == nil {
		return s
	}
	return s.s.h.currentSite.Info
}

func (s *SiteInfo) String() string {
	return fmt.Sprintf("Site(%q)", s.title)
}

func (s *SiteInfo) BaseURL() template.URL {
	return template.URL(s.s.PathSpec.BaseURL.String())
}

func (s *Site) isEnabled(kind string) bool {
	if kind == kindUnknown {
		panic("Unknown kind")
	}
	return true
}

func (s *Site) process(config BuildCfg) (err error) {
	if err = s.initialize(); err != nil {
		err = fmt.Errorf("initialize: %w", err)
		return
	}
	if err = s.readAndProcessContent(config); err != nil {
		err = fmt.Errorf("readAndProcessContent: %w", err)
		fmt.Println("read and process content err")
		fmt.Printf("%#v", err)

		return
	}
	return err
}

func (s *Site) initialize() (err error) {
	return s.initializeSiteInfo()
}

func (s *Site) readAndProcessContent(buildConfig BuildCfg, filenames ...string) error {
	sourceSpec := source.NewSourceSpec(s.PathSpec, buildConfig.ContentInclusionFilter, s.BaseFs.Content.Fs)

	proc := newPagesProcessor(s.h, sourceSpec)

	c := newPagesCollector(sourceSpec, s.h.getContentMaps(), proc, filenames...)

	if err := c.Collect(); err != nil {
		return err
	}

	return nil
}

func (s *Site) publish(path string, r io.Reader, fs afero.Fs) (err error) {
	return helpers.WriteToDisk(filepath.Clean(path), r, fs)
}

func (s *Site) newPage(
	n *contentNode,
	parentbBucket *pagesMapBucket,
	kind, title string,
	sections ...string) *pageState {

	m := map[string]any{}
	if title != "" {
		m["title"] = title
	}

	p, err := newPageFromMeta(
		n,
		parentbBucket,
		m,
		&pageMeta{
			s:        s,
			kind:     kind,
			sections: sections,
		})
	if err != nil {
		panic(err)
	}

	return p
}

func (s *SiteInfo) Params() maps.Params {
	return s.s.Language().Params()
}

func (s *Site) Language() *langs.Language {
	return s.language
}

func (s *Site) kindFromFileInfoOrSections(fi *fileInfo, sections []string) string {
	if fi.TranslationBaseName() == "_index" {
		if fi.Dir() == "" {
			return page.KindHome
		}

		return s.kindFromSections(sections)

	}

	return page.KindPage
}

func (s *Site) kindFromSections(sections []string) string {
	if len(sections) == 0 {
		return page.KindHome
	}

	return s.kindFromSectionPath(path.Join(sections...))
}

func (s *Site) kindFromSectionPath(sectionPath string) string {
	return page.KindSection
}

func (s *Site) shouldBuild(p page.Page) bool {
	return true
}

// Sites is a convenience method to get all the Hugo sites/languages configured.
func (s *SiteInfo) Sites() page.Sites {
	return s.s.h.siteInfos()
}

// hookRendererTemplate is the canonical implementation of all hooks.ITEMRenderer,
// where ITEM is the thing being hooked.
type hookRendererTemplate struct {
	templateHandler tpl.TemplateHandler
	identity.SearchProvider
	templ           tpl.Template
	resolvePosition func(ctx any) text.Position
}

func (p *pageState) getContentConverter() converter.Converter {
	var err error
	p.m.contentConverterInit.Do(func() {
		markup := p.m.markup
		if markup == "html" {
			// Only used for shortcode inner content.
			markup = "markdown"
		}
		p.m.contentConverter, err = p.m.newContentConverter(p, markup)
	})

	if err != nil {
		fmt.Printf("Failed to create content converter: %v", err)
	}
	return p.m.contentConverter
}

func (s *Site) initInit(init *lazy.Init, pctx pageContext) bool {
	_, err := init.Do()
	if err != nil {
		fmt.Printf("fatal error %v", pctx.wrapError(err))
	}
	return err == nil
}

// pageContext provides contextual information about this page, for error
// logging and similar.
type pageContext interface {
	posOffset(offset int) text.Position
	wrapError(err error) error
	getContentConverter() converter.Converter
	addDependency(dep identity.Provider)
}

// This is all the kinds we can expect to find in .Site.Pages.
var allKindsInPages = []string{page.KindPage, page.KindHome, page.KindSection, page.KindTerm, page.KindTaxonomy}

func (s *Site) initRenderFormats() {
	formatSet := make(map[string]bool)
	formats := output.Formats{}
	s.pageMap.pageTrees.WalkRenderable(func(s string, n *contentNode) bool {
		// empty
		for _, f := range n.p.m.configuredOutputFormats {
			if !formatSet[f.Name] {
				formats = append(formats, f)
				formatSet[f.Name] = true
			}
		}
		return false
	})

	// media type - format
	// site output format - render format
	// Add the per kind configured output formats
	for _, kind := range allKindsInPages {
		if siteFormats, found := s.outputFormats[kind]; found {
			for _, f := range siteFormats {
				if !formatSet[f.Name] {
					formats = append(formats, f)
					formatSet[f.Name] = true
				}
			}
		}
	}

	sort.Sort(formats)

	// HTML
	s.renderFormats = formats
}

func (s *Site) render(ctx *siteRenderContext) (err error) {
	if err = s.renderPages(ctx); err != nil {
		return
	}

	if ctx.outIdx == 0 {
		if err = s.render404(); err != nil {
			return
		}
	}

	return
}

func (s *Site) renderAndWritePage(name string, targetPath string, p *pageState, templ tpl.Template) error {
	renderBuffer := bp.GetBuffer()
	defer bp.PutBuffer(renderBuffer)

	of := p.outputFormat()

	if err := s.renderForTemplate(p.Kind(), of.Name, p, renderBuffer, templ); err != nil {
		return err
	}

	if renderBuffer.Len() == 0 {
		return nil
	}

	isHTML := of.IsHTML

	pd := publisher.Descriptor{
		Src:          renderBuffer,
		TargetPath:   targetPath,
		OutputFormat: p.outputFormat(),
	}

	if isHTML {
		if s.Info.relativeURLs {
			fmt.Println("based on default configuration, should never been here")
			pd.AbsURLPath = s.absURLPath(targetPath)
		}

		// For performance reasons we only inject the Hugo generator tag on the home page.
		if p.IsHome() {
			pd.AddHugoGeneratorTag = !s.Cfg.GetBool("disableHugoGeneratorInject")
		}
	}

	return s.publisher.Publish(pd)
}

func (s *Site) renderForTemplate(name, outputFormat string, d any, w io.Writer, templ tpl.Template) (err error) {
	if templ == nil {
		fmt.Printf("missing layout name: %s, output format: %s", name, outputFormat)
		return nil
	}

	if err = s.Tmpl().Execute(templ, w, d); err != nil {
		return fmt.Errorf("render of %q failed: %w", name, err)
	}
	return
}

func (s *Site) absURLPath(targetPath string) string {
	var path string
	if s.Info.relativeURLs {
		path = helpers.GetDottedRelativePath(targetPath)
	} else {
		url := s.PathSpec.BaseURL.String()
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		path = url
	}

	return path
}

func (s *Site) lookupLayouts(layouts ...string) tpl.Template {
	for _, l := range layouts {
		if templ, found := s.Tmpl().Lookup(l); found {
			return templ
		}
	}

	return nil
}

func newSiteRefLinker(cfg config.Provider, s *Site) (siteRefLinker, error) {

	notFoundURL := cfg.GetString("refLinksNotFoundURL")

	return siteRefLinker{s: s, notFoundURL: notFoundURL}, nil
}

func (s siteRefLinker) logNotFound(ref, what string, p page.Page, position text.Position) {
	if position.IsValid() {
		fmt.Printf("[%s] REF_NOT_FOUND: Ref %q: %s: %s", s.s.Lang(), ref, position.String(), what)
	} else if p == nil {
		fmt.Printf("[%s] REF_NOT_FOUND: Ref %q: %s", s.s.Lang(), ref, what)
	} else {
		fmt.Printf("[%s] REF_NOT_FOUND: Ref %q from page %q: %s", s.s.Lang(), ref, p.Pathc(), what)
	}
}

func (s *siteRefLinker) refLink(ref string, source any, relative bool, outputFormat string) (string, error) {
	return "", nil
}

func (s *Site) errorCollator(results <-chan error, errs chan<- error) {
	var errors []error
	for e := range results {
		errors = append(errors, e)
	}

	errs <- s.h.pickOneAndLogTheRest(errors)

	close(errs)
}
