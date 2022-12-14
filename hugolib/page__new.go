package hugolib

import (
	"github.com/sunwei/hugo-playground/lazy"
	"github.com/sunwei/hugo-playground/log"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/resources/page"
	"go.uber.org/atomic"
)

func newPageBase(metaProvider *pageMeta) (*pageState, error) {
	if metaProvider.s == nil {
		panic("must provide a Site")
	}

	s := metaProvider.s

	ps := &pageState{
		pageOutput:                        nopPageOutput,
		pageOutputTemplateVariationsState: atomic.NewUint32(0),
		pageCommon: &pageCommon{
			FileProvider:           metaProvider,
			ResourceMetaProvider:   metaProvider,
			ResourceParamsProvider: metaProvider,
			PageMetaProvider:       metaProvider,
			OutputFormatsProvider:  page.NopPage,
			ResourceTypeProvider:   pageTypesProvider,
			LanguageProvider:       s,

			init: lazy.New(),
			m:    metaProvider,
			s:    s,
		},
	}

	ps.ChildCareProvider = ps
	ps.TreeProvider = pageTree{p: ps}
	ps.Eqer = ps

	return ps, nil
}

func newPageBucket(p *pageState) *pagesMapBucket {
	return &pagesMapBucket{owner: p, pagesMapBucketPages: &pagesMapBucketPages{}}
}

func newPageFromMeta(
	n *contentNode,
	parentBucket *pagesMapBucket,
	meta map[string]any,
	metaProvider *pageMeta) (*pageState, error) {

	if metaProvider.f == nil {
		metaProvider.f = page.NewZeroFile()
	}

	ps, err := newPageBase(metaProvider)
	if err != nil {
		return nil, err
	}

	bucket := parentBucket

	if ps.IsNode() { // "/"
		ps.bucket = newPageBucket(ps)
	}

	if meta != nil || parentBucket != nil {
		if err := metaProvider.setMetadata(bucket, ps, meta); err != nil {
			return nil, err
		}
	}

	if err := metaProvider.applyDefaultValues(n); err != nil {
		return nil, err
	}

	ps.init.Add(func() (any, error) {
		log.Process("pageState init", "new page paths")
		pp, err := newPagePaths(metaProvider.s, ps, metaProvider)
		if err != nil {
			return nil, err
		}

		makeOut := func(f output.Format, render bool) *pageOutput {
			log.Process("pageState init", "new page output")
			return newPageOutput(ps, pp, f, render)
		}

		shouldRenderPage := !ps.m.noRender()

		if ps.m.standalone {
			ps.pageOutput = makeOut(ps.m.outputFormats()[0], shouldRenderPage)
		} else {
			outputFormatsForPage := ps.m.outputFormats()

			// Prepare output formats for all sites.
			// We do this even if this page does not get rendered on
			// its own. It may be referenced via .Site.GetPage and
			// it will then need an output format.
			ps.pageOutputs = make([]*pageOutput, len(ps.s.h.renderFormats))
			created := make(map[string]*pageOutput)
			for i, f := range ps.s.h.renderFormats {
				po, found := created[f.Name]
				if !found {
					render := shouldRenderPage
					if render {
						_, render = outputFormatsForPage.GetByName(f.Name)
					}
					po = makeOut(f, render)
					created[f.Name] = po
				}
				ps.pageOutputs[i] = po
			}
		}

		log.Process("pageState init", "init OutputFormatsProvider, targetPathDescriptor, SitesProvider")
		if err := ps.initCommonProviders(pp); err != nil {
			return nil, err
		}

		return nil, nil
	})

	return ps, err
}

// Used by the legacy 404, sitemap and robots.txt rendering
func newPageStandalone(m *pageMeta, f output.Format) (*pageState, error) {
	m.configuredOutputFormats = output.Formats{f}
	m.standalone = true
	p, err := newPageFromMeta(nil, nil, nil, m)
	if err != nil {
		return nil, err
	}

	if err := p.initPage(); err != nil {
		return nil, err
	}

	return p, nil
}
