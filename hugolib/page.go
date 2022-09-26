package hugolib

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/sunwei/hugo-playground/common/text"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/hugofs/files"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/log"
	"github.com/sunwei/hugo-playground/media"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/parser/metadecoders"
	"github.com/sunwei/hugo-playground/parser/pageparser"
	"github.com/sunwei/hugo-playground/resources/page"
	"github.com/sunwei/hugo-playground/resources/resource"
	"github.com/sunwei/hugo-playground/source"
	"github.com/sunwei/hugo-playground/tpl"
	"go.uber.org/atomic"
	"path/filepath"
	"sort"
	"strings"
)

var (
	_ page.Page = (*pageState)(nil)
)

var (
	pageTypesProvider = resource.NewResourceTypesProvider(media.OctetType, pageResourceType)
	nopPageOutput     = &pageOutput{
		pagePerOutputProviders:  nopPagePerOutput,
		ContentProvider:         page.NopPage,
		TableOfContentsProvider: page.NopPage,
	}
)

type pageState struct {
	// This slice will be of same length as the number of global slice of output
	// formats (for all sites).
	pageOutputs []*pageOutput

	// Used to determine if we can reuse content across output formats.
	pageOutputTemplateVariationsState *atomic.Uint32

	// This will be shifted out when we start to render a new output format.
	*pageOutput

	// Common for all output formats.
	*pageCommon
}

func (p *pageState) Err() resource.ResourceError {
	return nil
}

func (s *Site) sectionsFromFile(fi source.File) []string {
	dirname := fi.Dir()

	dirname = strings.Trim(dirname, helpers.FilePathSeparator)
	if dirname == "" {
		return nil
	}
	parts := strings.Split(dirname, helpers.FilePathSeparator)

	if fii, ok := fi.(*fileInfo); ok {
		if len(parts) > 0 && fii.FileInfo().Meta().Classifier == files.ContentClassLeaf {
			// my-section/mybundle/index.md => my-section
			return parts[:len(parts)-1]
		}
	}

	return parts
}

func (p *pageState) mapContent(bucket *pagesMapBucket, meta *pageMeta) error {
	p.cmap = &pageContentMap{
		items: make([]any, 0, 20),
	}

	return p.mapContentForResult(
		p.source.parsed,
		p.cmap,
		meta.markup,
		func(m map[string]interface{}) error {
			return meta.setMetadata(bucket, p, m)
		},
	)
}

func (p *pageState) mapContentForResult(
	result pageparser.Result,
	rn *pageContentMap,
	markup string,
	withFrontMatter func(map[string]any) error,
) error {

	iter := result.Iterator()

	fail := func(err error, i pageparser.Item) error {
		return errors.New("fail fail fail")
	}

	// the parser is guaranteed to return items in proper order or fail, so …
	// … it's safe to keep some "global" state
	var frontMatterSet bool

Loop:
	for {
		it := iter.Next()

		switch {
		case it.Type == pageparser.TypeIgnore:
		case it.IsFrontMatter():
			f := pageparser.FormatFromFrontMatterType(it.Type)
			m, err := metadecoders.Default.UnmarshalToMap(it.Val(result.Input()), f)
			if err != nil {
				return err
			}

			if withFrontMatter != nil {
				if err := withFrontMatter(m); err != nil {
					return err
				}
			}

			frontMatterSet = true

			next := iter.Peek()
			if !next.IsDone() {
				p.source.posMainContent = next.Pos()
			}

			if !p.s.shouldBuild(p) {
				// Nothing more to do.
				return nil
			}

		case it.Type == pageparser.TypeLeadSummaryDivider:
			posBody := -1
			f := func(item pageparser.Item) bool {
				if posBody == -1 && !item.IsDone() {
					posBody = item.Pos()
				}

				if item.IsNonWhitespace(result.Input()) {
					p.truncated = true

					// Done
					return false
				}
				return true
			}
			iter.PeekWalk(f)

			p.source.posSummaryEnd = it.Pos()
			p.source.posBodyStart = posBody
			p.source.hasSummaryDivider = true

			if markup != "html" {
				// The content will be rendered by Goldmark or similar,
				// and we need to track the summary.
				rn.AddReplacement(internalSummaryDividerPre, it)
			}
		case it.Type == pageparser.TypeEmoji:
			rn.AddBytes(it)
		case it.IsEOF():
			break Loop
		case it.IsError():
			err := fail(errors.New(it.ValStr(result.Input())), it)
			return err

		default:
			rn.AddBytes(it)
		}
	}

	if !frontMatterSet && withFrontMatter != nil {
		// Page content without front matter. Assign default front matter from
		// cascades etc.
		if err := withFrontMatter(nil); err != nil {
			return err
		}
	}

	return nil
}

func (p *pageState) parseError(err error, input []byte, offset int) error {
	return errors.New("pos")
}

func (p *pageState) posFromInput(input []byte, offset int) text.Position {
	if offset < 0 {
		return text.Position{
			Filename: p.pathOrTitle(),
		}
	}
	lf := []byte("\n")
	input = input[:offset]
	lineNumber := bytes.Count(input, lf) + 1
	endOfLastLine := bytes.LastIndex(input, lf)

	return text.Position{
		Filename:     p.pathOrTitle(),
		LineNumber:   lineNumber,
		ColumnNumber: offset - endOfLastLine,
		Offset:       offset,
	}
}

func (p *pageState) pathOrTitle() string {
	if !p.File().IsZero() {
		return p.File().Filename()
	}

	if p.Pathc() != "" {
		return p.Pathc()
	}

	return p.Title()
}

func (p *pageState) GetIdentity() identity.Identity {
	return identity.NewPathIdentity(files.ComponentFolderContent, filepath.FromSlash(p.Pathc()))
}

func (p *pageState) outputFormat() (f output.Format) {
	if p.pageOutput == nil {
		panic("no pageOutput")
	}
	return p.pageOutput.f
}

func (p *pageState) AlternativeOutputFormats() page.OutputFormats {
	f := p.outputFormat()
	var o page.OutputFormats
	for _, of := range p.OutputFormats() {
		if of.Format.NotAlternative || of.Format.Name == f.Name {
			continue
		}

		o = append(o, of)
	}
	return o
}

func (ps *pageState) initCommonProviders(pp pagePaths) error {
	ps.OutputFormatsProvider = pp
	ps.targetPathDescriptor = pp.targetPathDescriptor
	ps.SitesProvider = ps.s.Info

	return nil
}

func (p *pageState) getLayoutDescriptor() output.LayoutDescriptor {
	p.layoutDescriptorInit.Do(func() {
		var section string
		sections := p.SectionsEntries()

		switch p.Kind() {
		case page.KindSection:
			if len(sections) > 0 {
				section = sections[0]
			}
		case page.KindTaxonomy, page.KindTerm:
			b := p.getTreeRef().n
			section = b.viewInfo.name.singular
		default:
		}

		p.layoutDescriptor = output.LayoutDescriptor{
			Kind:    p.Kind(),
			Type:    p.Type(),
			Lang:    p.Language().Lang,
			Layout:  p.Layout(),
			Section: section,
		}
	})

	return p.layoutDescriptor
}

func (p *pageState) reusePageOutputContent() bool {
	return p.pageOutputTemplateVariationsState.Load() == 1
}

func (p *pageState) addDependency(dep identity.Provider) {
	return
}

func (p *pageState) resolveTemplate() (tpl.Template, bool, error) {
	f := p.outputFormat() // set in shiftToOutputFormat
	d := p.getLayoutDescriptor()

	return p.s.Tmpl().LookupLayout(d, f)
}

func (p *pageState) posOffset(offset int) text.Position {
	return p.posFromInput(p.source.parsed.Input(), offset)
}

// wrapError adds some more context to the given error if possible/needed
func (p *pageState) wrapError(err error) error {
	if err == nil {
		panic("wrapError with nil")
	}

	if p.File().IsZero() {
		// No more details to add.
		return fmt.Errorf("%q: %w", p.Pathc(), err)
	}

	filename := p.File().Filename()

	return errors.New(filename)

}

// This is serialized
func (p *pageState) initOutputFormat(isRenderingSite bool, idx int) error {
	if err := p.shiftToOutputFormat(isRenderingSite, idx); err != nil {
		return err
	}

	return nil
}

// shiftToOutputFormat is serialized. The output format idx refers to the
// full set of output formats for all sites.
func (p *pageState) shiftToOutputFormat(isRenderingSite bool, idx int) error {
	log.Process("pageState", "init page do start")
	if err := p.initPage(); err != nil {
		return err
	}

	if len(p.pageOutputs) == 1 {
		idx = 0
	}

	p.pageOutput = p.pageOutputs[idx]
	if p.pageOutput == nil {
		panic(fmt.Sprintf("pageOutput is nil for output idx %d", idx))
	}

	// Reset any built paginator. This will trigger when re-rendering pages in
	// server mode.
	if isRenderingSite && p.pageOutput.paginator != nil && p.pageOutput.paginator.current != nil {
		p.pageOutput.paginator.reset()
	}

	if isRenderingSite {
		cp := p.pageOutput.cp
		if cp == nil && p.reusePageOutputContent() {
			// Look for content to reuse.
			for i := 0; i < len(p.pageOutputs); i++ {
				if i == idx {
					continue
				}
				po := p.pageOutputs[i]

				if po.cp != nil {
					cp = po.cp
					break
				}
			}
		}

		if cp == nil {
			var err error
			log.Process("pageState", "new page content output")
			cp, err = newPageContentOutput(p, p.pageOutput)
			if err != nil {
				return err
			}
		}
		log.Process("pageState", "init contentProvider with page content output")
		p.pageOutput.initContentProvider(cp)
	} else {
		panic("not ready for unrendering site yet")
	}

	return nil
}

// Must be run after the site section tree etc. is built and ready.
func (p *pageState) initPage() error {
	if _, err := p.init.Do(); err != nil {
		return err
	}
	return nil
}

func (p *pageState) Resources() resource.Resources {
	p.resourcesInit.Do(func() {
		p.sortResources()
		if len(p.m.resourcesMetadata) > 0 {
			panic("resources metadata not supported yet")
		}
	})
	return p.resources
}

func (p *pageState) sortResources() {
	sort.SliceStable(p.resources, func(i, j int) bool {
		ri, rj := p.resources[i], p.resources[j]
		if ri.ResourceType() < rj.ResourceType() {
			return true
		}

		p1, ok1 := ri.(page.Page)
		p2, ok2 := rj.(page.Page)

		if ok1 != ok2 {
			return ok2
		}

		if ok1 {
			return page.DefaultPageSort(p1, p2)
		}

		// Make sure not to use RelPermalink or any of the other methods that
		// trigger lazy publishing.
		return ri.Name() < rj.Name()
	})
}

func (p *pageState) getTargetPaths() page.TargetPaths {
	return p.targetPaths()
}
