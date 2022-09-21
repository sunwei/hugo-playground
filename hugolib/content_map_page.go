package hugolib

import (
	"context"
	"fmt"
	"github.com/sunwei/hugo-playground/common/hugio"
	"github.com/sunwei/hugo-playground/common/maps"
	"github.com/sunwei/hugo-playground/common/para"
	"github.com/sunwei/hugo-playground/parser/pageparser"
	"github.com/sunwei/hugo-playground/resources/page"
	"github.com/sunwei/hugo-playground/resources/resource"
	"strings"
	"sync"
)

type pageMap struct {
	s *Site
	*contentMap
}

type pageMaps struct {
	workers *para.Workers
	pmaps   []*pageMap
}

type pagesMapBucket struct {
	// Cascading front matter.
	cascade map[page.PageMatcher]maps.Params

	owner *pageState // The branch node

	*pagesMapBucketPages
}

type pagesMapBucketPages struct {
	pagesInit sync.Once
	pages     page.Pages

	pagesAndSectionsInit sync.Once
	pagesAndSections     page.Pages

	sectionsInit sync.Once
	sections     page.Pages
}

type viewName struct {
	singular string // e.g. "category"
	plural   string // e.g. "categories"
}

type pageMapQuery struct {
	Prefix string
	Filter contentTreeNodeCallback
}

func (m *pageMap) createListAllPages() page.Pages {
	pages := make(page.Pages, 0)
	m.contentMap.pageTrees.Walk(func(s string, n *contentNode) bool {
		if n.p == nil {
			panic(fmt.Sprintf("BUG: page not set for %q", s))
		}
		if contentTreeNoListAlwaysFilter(s, n) {
			return false
		}
		pages = append(pages, n.p)
		return false
	})

	page.SortByDefault(pages)
	return pages
}

func (v viewName) IsZero() bool {
	return v.singular == ""
}

func newPageMaps(h *HugoSites) *pageMaps {
	mps := make([]*pageMap, len(h.Sites))
	for i, s := range h.Sites {
		mps[i] = s.pageMap
	}
	return &pageMaps{
		workers: para.New(h.numWorkers),
		pmaps:   mps,
	}
}

func (m *pageMaps) AssemblePages() error {
	return m.withMaps(func(pm *pageMap) error {
		if err := pm.CreateMissingNodes(); err != nil {
			return err
		}

		if err := pm.assemblePages(); err != nil {
			return err
		}

		// Handle any new sections created in the step above.
		if err := pm.assembleSections(); err != nil {
			return err
		}

		return nil
	})
}

func (m *pageMaps) withMaps(fn func(pm *pageMap) error) error {
	g, _ := m.workers.Start(context.Background())
	for _, pm := range m.pmaps {
		pm := pm
		g.Run(func() error {
			return fn(pm)
		})
	}
	return g.Wait()
}

func (m *pageMap) assemblePages() error {
	if err := m.assembleSections(); err != nil {
		return err
	}

	var err error

	if err != nil {
		return err
	}

	m.pages.Walk(func(s string, v any) bool {
		n := v.(*contentNode)

		if n.p != nil {
			// A rebuild
			return false
		}

		var parent *contentNode
		var parentBucket *pagesMapBucket

		_, parent = m.getSection(s)
		fmt.Println("get section parent:")
		fmt.Println(parent)
		fmt.Println("~~~")

		if parent == nil {
			panic(fmt.Sprintf("BUG: parent not set for %q", s))
		}
		parentBucket = parent.p.bucket

		n.p, err = m.newPageFromContentNode(n, parentBucket, nil)
		if err != nil {
			return true
		}

		n.p.treeRef = &contentTreeRef{
			m:   m,
			t:   m.pages,
			n:   n,
			key: s,
		}

		if err = m.assembleResources(s, n.p, parentBucket); err != nil {
			return true
		}

		return false
	})

	m.deleteOrphanSections()

	return err
}

func (m *pageMap) assembleSections() error {
	var err error

	m.sections.Walk(func(s string, v any) bool {
		fmt.Println("assemble sections walk")
		fmt.Println(s)
		fmt.Println("+++")

		n := v.(*contentNode)
		var shouldBuild bool

		defer func() {
			// Make sure we always rebuild the view cache.
			if shouldBuild && err == nil && n.p != nil {
				if n.p.IsHome() {
					m.s.home = n.p
				}
			}
		}()

		sections := m.splitKey(s)
		fmt.Println("assemble sections sections:")
		fmt.Println(sections)
		fmt.Printf("%#v\n", n.p)
		fmt.Println("___")

		if n.p != nil {
			if n.p.IsHome() {
				m.s.home = n.p
			}
			shouldBuild = true
			return false
		}

		var parent *contentNode
		var parentBucket *pagesMapBucket

		if s != "/" {
			_, parent = m.getSection(s)
			if parent == nil || parent.p == nil {
				panic(fmt.Sprintf("BUG: parent not set for %q", s))
			}
		}

		if parent != nil {
			parentBucket = parent.p.bucket
		} else if s == "/" {
			parentBucket = m.s.siteBucket
		}

		kind := page.KindSection
		if s == "/" {
			kind = page.KindHome
		}

		if n.fi != nil {
			panic("assembleSections newPageFromContentNode not ready")
		} else { // new page
			n.p = m.s.newPage(n, parentBucket, kind, "", sections...)
		}

		n.p.treeRef = &contentTreeRef{
			m:   m,
			t:   m.sections,
			n:   n,
			key: s,
		}

		if err = m.assembleResources(s+cmLeafSeparator, n.p, parentBucket); err != nil {
			return true
		}

		return false
	})

	return err
}

func (b *pagesMapBucket) getSections() page.Pages {
	b.sectionsInit.Do(func() {
		if b.owner.treeRef == nil {
			return
		}
		b.sections = b.owner.treeRef.getSections()
	})

	return b.sections
}

func (m *pageMap) collectSections(query pageMapQuery, fn func(c *contentNode)) error {
	level := strings.Count(query.Prefix, "/")

	return m.collectSectionsFn(query, func(s string, c *contentNode) bool {
		if strings.Count(s, "/") != level+1 {
			return false
		}

		fn(c)

		return false
	})
}

func (m *pageMap) collectSectionsFn(query pageMapQuery, fn func(s string, c *contentNode) bool) error {
	if !strings.HasSuffix(query.Prefix, "/") {
		query.Prefix += "/"
	}

	m.sections.WalkQuery(query, func(s string, n *contentNode) bool {
		return fn(s, n)
	})

	return nil
}

type sectionAggregateHandler struct {
	sectionAggregate
	sectionPageCount int

	// Section
	b *contentNode
	s string
}

type sectionAggregate struct {
	datesAll             resource.Dates
	datesSection         resource.Dates
	pageCount            int
	mainSection          string
	mainSectionPageCount int
}

type sectionWalkHandler interface {
	handleNested(v sectionWalkHandler) error
	handlePage(s string, b *contentNode) error
	handleSectionPost() error
	handleSectionPre(s string, b *contentNode) error
}

func (h *sectionAggregateHandler) String() string {
	return fmt.Sprintf("%s/%s - %d - %s", h.sectionAggregate.datesAll, h.sectionAggregate.datesSection, h.sectionPageCount, h.s)
}

func (h *sectionAggregateHandler) isRootSection() bool {
	return h.s != "/" && strings.Count(h.s, "/") == 2
}

func (h *sectionAggregateHandler) handleNested(v sectionWalkHandler) error {
	nested := v.(*sectionAggregateHandler)
	h.sectionPageCount += nested.pageCount
	h.pageCount += h.sectionPageCount
	h.datesAll.UpdateDateAndLastmodIfAfter(nested.datesAll)
	h.datesSection.UpdateDateAndLastmodIfAfter(nested.datesAll)
	return nil
}

func (h *sectionAggregateHandler) handlePage(s string, n *contentNode) error {
	h.sectionPageCount++

	var d resource.Dated
	if n.p != nil {
		d = n.p
	} else if n.viewInfo != nil && n.viewInfo.ref != nil {
		d = n.viewInfo.ref.p
	} else {
		return nil
	}

	h.datesAll.UpdateDateAndLastmodIfAfter(d)
	h.datesSection.UpdateDateAndLastmodIfAfter(d)
	return nil
}

func (h *sectionAggregateHandler) handleSectionPost() error {
	if h.sectionPageCount > h.mainSectionPageCount && h.isRootSection() {
		h.mainSectionPageCount = h.sectionPageCount
		h.mainSection = strings.TrimPrefix(h.s, "/")
	}

	if resource.IsZeroDates(h.b.p) {
		h.b.p.m.Dates = h.datesSection
	}

	h.datesSection = resource.Dates{}

	return nil
}

func (h *sectionAggregateHandler) handleSectionPre(s string, b *contentNode) error {
	h.s = s
	h.b = b
	h.sectionPageCount = 0
	h.datesAll.UpdateDateAndLastmodIfAfter(b.p)
	return nil
}

func (m *pageMap) newPageFromContentNode(n *contentNode, parentBucket *pagesMapBucket, owner *pageState) (*pageState, error) {
	if n.fi == nil {
		panic("FileInfo must (currently) be set")
	}

	f, err := newFileInfo(m.s.SourceSpec, n.fi)
	if err != nil {
		return nil, err
	}

	meta := n.fi.Meta()
	content := func() (hugio.ReadSeekCloser, error) {
		return meta.Open()
	}

	bundled := owner != nil // false
	s := m.s

	sections := s.sectionsFromFile(f)

	kind := s.kindFromFileInfoOrSections(f, sections)
	metaProvider := &pageMeta{kind: kind, sections: sections, bundled: bundled, s: s, f: f}

	ps, err := newPageBase(metaProvider)
	if err != nil {
		return nil, err
	}

	if n.fi.Meta().IsRootFile {
		// Make sure that the bundle/section we start walking from is always
		// rendered.
		// This is only relevant in server fast render mode.
		ps.forceRender = true
	}

	n.p = ps
	if ps.IsNode() {
		ps.bucket = newPageBucket(ps)
	}

	r, err := content()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// .md parseResult
	// TODO: parser works way
	parseResult, err := pageparser.Parse(
		r,
		pageparser.Config{EnableEmoji: false},
	)
	if err != nil {
		return nil, err
	}

	ps.pageContent = pageContent{
		source: rawPageContent{
			parsed:         parseResult,
			posMainContent: -1,
			posSummaryEnd:  -1,
			posBodyStart:   -1,
		},
	}

	if err := ps.mapContent(parentBucket, metaProvider); err != nil {
		return nil, err
	}

	if err := metaProvider.applyDefaultValues(n); err != nil {
		return nil, err
	}

	ps.init.Add(func() (any, error) {
		pp, err := newPagePaths(s, ps, metaProvider)
		if err != nil {
			return nil, err
		}

		outputFormatsForPage := ps.m.outputFormats()

		// Prepare output formats for all sites.
		// We do this even if this page does not get rendered on
		// its own. It may be referenced via .Site.GetPage and
		// it will then need an output format.
		ps.pageOutputs = make([]*pageOutput, len(ps.s.h.renderFormats))
		created := make(map[string]*pageOutput)
		shouldRenderPage := !ps.m.noRender()

		// all pages should get ready for h.renderFormats
		// and page has its own output formats from pageState meta
		// We need to prepare all the page output at this time with render sign setup
		for i, f := range ps.s.h.renderFormats {
			if po, found := created[f.Name]; found {
				ps.pageOutputs[i] = po
				continue
			}

			render := shouldRenderPage
			if render {
				_, render = outputFormatsForPage.GetByName(f.Name)
			}

			po := newPageOutput(ps, pp, f, render)

			// Create a content provider for the first,
			// we may be able to reuse it.
			if i == 0 {
				contentProvider, err := newPageContentOutput(ps, po)
				if err != nil {
					return nil, err
				}
				po.initContentProvider(contentProvider)
			}

			ps.pageOutputs[i] = po
			created[f.Name] = po

		}

		if err := ps.initCommonProviders(pp); err != nil {
			return nil, err
		}

		return nil, nil
	})

	ps.parent = owner

	return ps, nil
}

// withEveryBundlePage applies fn to every Page, including those bundled inside
// leaf bundles.
func (m *pageMap) withEveryBundlePage(fn func(p *pageState) bool) {
	m.bundleTrees.Walk(func(s string, n *contentNode) bool {
		if n.p != nil {
			return fn(n.p)
		}
		return false
	})
}

func (m *pageMap) assembleResources(s string, p *pageState, parentBucket *pagesMapBucket) error {
	var err error

	m.resources.WalkPrefix(s, func(s string, v any) bool {
		panic("assemble resources not ready yet")
	})

	return err
}
