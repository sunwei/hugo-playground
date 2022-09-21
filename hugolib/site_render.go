package hugolib

import (
	"errors"
	"fmt"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/resources/page/pagemeta"
	"sync"
)

type siteRenderContext struct {
	cfg *BuildCfg

	// Zero based index for all output formats combined.
	sitesOutIdx int

	// Zero based index of the output formats configured within a Site.
	// Note that these outputs are sorted.
	outIdx int

	multihost bool
}

// renderPages renders pages each corresponding to a markdown file.
func (s *Site) renderPages(ctx *siteRenderContext) error {
	numWorkers := 3

	results := make(chan error)
	pages := make(chan *pageState, numWorkers) // buffered for performance
	errs := make(chan error)

	go s.errorCollator(results, errs)

	wg := &sync.WaitGroup{}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go pageRenderer(ctx, s, pages, results, wg)
	}

	cfg := ctx.cfg

	var count int
	s.pageMap.pageTrees.Walk(func(ss string, n *contentNode) bool {
		if cfg.shouldRender(n.p) {
			select {
			default:
				count++
				pages <- n.p
			}
		}
		return false
	})

	fmt.Println("close pages...")
	close(pages)

	wg.Wait()

	fmt.Println("wait nothing...")
	close(results)
	fmt.Println("result closed...")

	err := <-errs
	if err != nil {
		return fmt.Errorf("failed to render pages: %w", err)
	}
	return nil
}

func pageRenderer(
	ctx *siteRenderContext,
	s *Site,
	pages <-chan *pageState,
	results chan<- error,
	wg *sync.WaitGroup) {
	defer wg.Done()

	for p := range pages {
		templ, found, err := p.resolveTemplate()
		if err != nil {
			fmt.Println("failed to resolve template")
			continue
		}

		if !found { // layout: "", kind: section, name: HTML
			fmt.Printf("layout: %s, kind: %s, name: %s", p.Layout(), p.Kind(), p.f.Name)
			continue
		}

		targetPath := p.targetPaths().TargetFilename

		if err := s.renderAndWritePage("page "+p.Title(), targetPath, p, templ); err != nil {
			fmt.Println(" render err")
			fmt.Printf("%#v", err)
			results <- err
		}

		if p.paginator != nil && p.paginator.current != nil {
			panic("render paginator is not ready")
		}
	}
	fmt.Println("render page done...")
}

func (s *Site) render404() error {
	p, err := newPageStandalone(&pageMeta{
		s:    s,
		kind: kind404,
		urlPaths: pagemeta.URLPath{
			URL: "404.html",
		},
	},
		output.HTMLFormat,
	)
	if err != nil {
		return err
	}

	if !p.render {
		return nil
	}

	var d output.LayoutDescriptor
	d.Kind = kind404

	templ, found, err := s.Tmpl().LookupLayout(d, output.HTMLFormat)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	targetPath := p.targetPaths().TargetFilename

	if targetPath == "" {
		return errors.New("failed to create targetPath for 404 page")
	}

	return s.renderAndWritePage("404 page", targetPath, p, templ)
}

// Whether to render 404.html, robotsTXT.txt which usually is rendered
// once only in the site root.
func (s siteRenderContext) renderSingletonPages() bool {
	if s.multihost {
		// 1 per site
		return s.outIdx == 0
	}

	// 1 for all sites
	return s.sitesOutIdx == 0
}
