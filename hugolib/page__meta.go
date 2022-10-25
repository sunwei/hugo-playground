// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hugolib

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/sunwei/hugo-playground/common/maps"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/markup/converter"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/related"
	"github.com/sunwei/hugo-playground/resources/page"
	"github.com/sunwei/hugo-playground/resources/page/pagemeta"
	"github.com/sunwei/hugo-playground/resources/resource"
	"github.com/sunwei/hugo-playground/source"
	"path"
	"regexp"
	"strings"
	"sync"
)

var cjkRe = regexp.MustCompile(`\p{Han}|\p{Hangul}|\p{Hiragana}|\p{Katakana}`)

type pageMeta struct {
	// kind is the discriminator that identifies the different page types
	// in the different page collections. This can, as an example, be used
	// to to filter regular pages, find sections etc.
	// Kind will, for the pages available to the templates, be one of:
	// page, home, section, taxonomy and term.
	// It is of string type to make it easy to reason about in
	// the templates.
	kind string

	// This is a standalone page not part of any page collection. These
	// include sitemap, robotsTXT and similar. It will have no pageOutputs, but
	// a fixed pageOutput.
	standalone bool

	buildConfig pagemeta.BuildConfig

	// Params contains configuration defined in the params section of page frontmatter.
	params map[string]any

	title     string
	linkTitle string

	summary string

	resourcePath string

	weight int

	markup      string
	contentType string

	// whether the content is in a CJK language.
	isCJKLanguage bool

	layout string

	aliases []string

	description string
	keywords    []string

	urlPaths pagemeta.URLPath

	resource.Dates

	// Set if this page is bundled inside another.
	bundled bool

	// A key that maps to translation(s) of this page. This value is fetched
	// from the page front matter.
	translationKey string

	// From front matter.
	configuredOutputFormats output.Formats

	// This is the raw front matter metadata that is going to be assigned to
	// the Resources above.
	resourcesMetadata []map[string]any

	f source.File

	sections []string

	s *Site

	contentConverterInit sync.Once
	contentConverter     converter.Converter
}

func (p *pageMeta) noRender() bool {
	return p.buildConfig.Render != pagemeta.Always
}

func (p *pageMeta) noListAlways() bool {
	return p.buildConfig.List != pagemeta.Always
}

func (p *pageMeta) File() source.File {
	return p.f
}

func (p *pageMeta) Name() string {
	if p.resourcePath != "" {
		return p.resourcePath
	}
	return p.Title()
}

func (p *pageMeta) Title() string {
	return p.title
}

func (p *pageMeta) Params() maps.Params {
	return p.params
}

func (p *pageMeta) Description() string {
	return p.description
}

func (p *pageMeta) IsHome() bool {
	return p.Kind() == page.KindHome
}

func (p *pageMeta) Kind() string {
	return p.kind
}

func (p *pageMeta) Layout() string {
	return p.layout
}

func (p *pageMeta) LinkTitle() string {
	if p.linkTitle != "" {
		return p.linkTitle
	}

	return p.Title()
}

func (p *pageMeta) IsNode() bool {
	return !p.IsPage()
}

func (p *pageMeta) IsPage() bool {
	return p.Kind() == page.KindPage
}

// Param is a convenience method to do lookups in Page's and Site's Params map,
// in that order.
//
// This method is also implemented on SiteInfo.
// TODO(bep) interface
func (p *pageMeta) Param(key any) (any, error) {
	return resource.Param(p, p.s.Info.Params(), key)
}

func (p *pageMeta) Path() string {
	if !p.File().IsZero() {
		const example = `
  {{ $path := "" }}
  {{ with .File }}
	{{ $path = .Path }}
  {{ else }}
	{{ $path = .Path }}
  {{ end }}
`
		helpers.Deprecated(".Path when the page is backed by a file", "We plan to use Path for a canonical source path and you probably want to check the source is a file. To get the current behaviour, you can use a construct similar to the one below:\n"+example, false)

	}

	return p.Pathc()
}

// This is just a bridge method, use Path in templates.
func (p *pageMeta) Pathc() string {
	if !p.File().IsZero() {
		return p.File().Path()
	}
	return p.SectionsPath()
}

func (p *pageMeta) SectionsPath() string {
	return path.Join(p.SectionsEntries()...)
}

func (p *pageMeta) SectionsEntries() []string {
	return p.sections
}

func (p *pageMeta) Slug() string {
	return p.urlPaths.Slug
}

func (p *pageMeta) IsSection() bool {
	return p.Kind() == page.KindSection
}

func (p *pageMeta) Section() string {
	if p.IsHome() {
		return ""
	}

	if p.IsNode() {
		if len(p.sections) == 0 {
			// May be a sitemap or similar.
			return ""
		}
		return p.sections[0]
	}

	if !p.File().IsZero() {
		return p.File().Section()
	}

	panic("invalid page state")
}

const defaultContentType = "page"

func (p *pageMeta) Type() string {
	if p.contentType != "" {
		return p.contentType
	}

	if sect := p.Section(); sect != "" {
		return sect
	}

	return defaultContentType
}

func (p *pageMeta) Weight() int {
	return p.weight
}

// RelatedKeywords implements the related.Document interface needed for fast page searches.
func (p *pageMeta) RelatedKeywords(cfg related.IndexConfig) ([]related.Keyword, error) {
	v, err := p.Param(cfg.Name)
	if err != nil {
		return nil, err
	}

	return cfg.ToKeywords(v)
}

func (pm *pageMeta) setMetadata(parentBucket *pagesMapBucket, p *pageState, frontmatter map[string]any) error {
	pm.params = make(maps.Params)

	// []
	if frontmatter == nil && (parentBucket == nil || parentBucket.cascade == nil) {
		return nil
	}

	if frontmatter != nil {
		// Needed for case insensitive fetching of params values
		maps.PrepareParams(frontmatter)
		if p.bucket != nil {
			// Check for any cascade define on itself.
			if _, found := frontmatter["cascade"]; found {
				panic("not ready for front matter cascade")
			}
		}
	} else {
		frontmatter = make(map[string]any)
	}

	var err error
	pm.buildConfig, err = pagemeta.DecodeBuildConfig(frontmatter["_build"]) // defaultBuildConfig
	if err != nil {
		return err
	}

	for k, v := range frontmatter { // map[title:P1]
		loki := strings.ToLower(k)

		switch loki {
		case "title":
			pm.title = cast.ToString(v)
			pm.params[loki] = pm.title
		}
	}

	pm.markup = p.s.ContentSpec.ResolveMarkup(pm.markup) // ""

	return nil
}

func (pm *pageMeta) mergeBucketCascades(b1, b2 *pagesMapBucket) {
	if b1.cascade == nil {
		b1.cascade = make(map[page.PageMatcher]maps.Params)
	}

	if b2 != nil && b2.cascade != nil {
		for k, v := range b2.cascade {

			vv, found := b1.cascade[k]
			if !found {
				b1.cascade[k] = v
			} else {
				// Merge
				for ck, cv := range v {
					if _, found := vv[ck]; !found {
						vv[ck] = cv
					}
				}
			}
		}
	}
}

func (p *pageMeta) applyDefaultValues(n *contentNode) error { // buildConfig, markup, title
	if p.buildConfig.IsZero() {
		p.buildConfig, _ = pagemeta.DecodeBuildConfig(nil)
	}

	if !p.s.isEnabled(p.Kind()) {
		(&p.buildConfig).Disable()
	}

	if p.markup == "" {
		if !p.File().IsZero() {
			// Fall back to file extension
			p.markup = p.s.ContentSpec.ResolveMarkup(p.File().Ext())
		}
		if p.markup == "" {
			p.markup = "markdown"
		}
	}

	if p.title == "" && p.f.IsZero() {
		switch p.Kind() {
		case page.KindHome:
			p.title = p.s.Info.title
		case page.KindSection:
			var sectionName string
			if n != nil {
				sectionName = n.rootSection()
			} else {
				sectionName = p.sections[0]
			}
			sectionName = helpers.FirstUpper(sectionName)
			p.title = sectionName
		case kind404:
			p.title = "404 Page not found"

		}
	}

	return nil
}

// The output formats this page will be rendered to.
func (m *pageMeta) outputFormats() output.Formats {
	if len(m.configuredOutputFormats) > 0 {
		return m.configuredOutputFormats
	}

	return m.s.outputFormats[m.Kind()]
}

func (p *pageMeta) noLink() bool {
	return p.buildConfig.Render == pagemeta.Never
}

func (p *pageMeta) newContentConverter(ps *pageState, markup string) (converter.Converter, error) {
	if ps == nil {
		panic("no Page provided")
	}
	cp := p.s.ContentSpec.Converters.Get(markup)
	if cp == nil {
		return converter.NopConverter, fmt.Errorf("no content renderer found for markup %q", p.markup)
	}

	var id string
	var filename string
	var path string
	if !p.f.IsZero() {
		id = p.f.UniqueID()
		filename = p.f.Filename()
		path = p.f.Path()
	} else {
		path = p.Pathc()
	}

	cpp, err := cp.New(
		converter.DocumentContext{
			Document:     newPageForRenderHook(ps),
			DocumentID:   id,
			DocumentName: path,
			Filename:     filename,
		},
	)
	if err != nil {
		return converter.NopConverter, err
	}

	return cpp, nil
}
