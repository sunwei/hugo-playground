package hugolib

import (
	"fmt"
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/common/maps"
	"github.com/sunwei/hugo-playground/common/text"
	"github.com/sunwei/hugo-playground/deps"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/langs"
	"github.com/sunwei/hugo-playground/lazy"
	"github.com/sunwei/hugo-playground/log"
	"github.com/sunwei/hugo-playground/markup/converter"
	"github.com/sunwei/hugo-playground/media"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/publisher"
	"github.com/sunwei/hugo-playground/resources/page"
	"io"
	"path"
	"path/filepath"
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
	log.Process("media.DecodeTypes", "set default media types")
	siteMediaTypesConfig, err = media.DecodeTypes(mediaTypesConfig...)
	if err != nil {
		return nil, err
	}

	// [{HTML}, {JSON}, {MARKDOWN}]
	log.Process("output.DecodeFormats", "set default output formats based on media types, and customized output formats configuration")
	siteOutputFormatsConfig, err = output.DecodeFormats(siteMediaTypesConfig, outputFormatsConfig...)

	if err != nil {
		return nil, err
	}

	// Site output formats source
	log.Process("site output formats", "map siteOutputFormats to every hugo page types(KindPage, KindHome...)")
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

func (s *Site) isEnabled(kind string) bool {
	if kind == kindUnknown {
		panic("Unknown kind")
	}
	return true
}

func (s *Site) initialize() (err error) {
	return s.initializeSiteInfo()
}

func (s *Site) publish(path string, r io.Reader, fs afero.Fs) (err error) {
	return helpers.WriteToDisk(filepath.Clean(path), r, fs)
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
