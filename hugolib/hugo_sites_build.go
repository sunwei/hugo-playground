package hugolib

import (
	"fmt"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/resources/page/pagemeta"
)

// Build builds all sites. If filesystem events are provided,
// this is considered to be a potential partial rebuild.
// ---
// Let's focus on full build, remove events
func (h *HugoSites) Build(config BuildCfg) error {
	conf := &config

	// process file system to create content map
	err := h.process(conf)
	if err != nil {
		return err
	}

	// based on content map, setup section, page, resource tree
	err = h.assemble(conf)
	if err != nil {
		return err
	}

	err = h.render(conf)
	if err != nil {
		return err
	}

	return nil
}

func (h *HugoSites) process(config *BuildCfg) error {
	firstSite := h.Sites[0]
	return firstSite.process(*config)
}

func (h *HugoSites) assemble(bcfg *BuildCfg) error {

	// node - page info - page meta - page state
	// get ready for render
	if err := h.getContentMaps().AssemblePages(); err != nil {
		return err
	}

	return nil
}

func (h *HugoSites) render(config *BuildCfg) error {
	// template.go MarkReady
	if _, err := h.init.layouts.Do(); err != nil {
		return err
	}

	siteRenderContext := &siteRenderContext{cfg: config, multihost: false}

	h.renderFormats = output.Formats{}
	h.withSite(func(s *Site) error {
		s.initRenderFormats()
		return nil
	})

	for _, s := range h.Sites {
		h.renderFormats = append(h.renderFormats, s.renderFormats...)
	}

	fmt.Println("render formats all:")
	fmt.Println(h.renderFormats) // HTML

	i := 0
	for _, s := range h.Sites {
		h.currentSite = s
		for siteOutIdx, renderFormat := range s.renderFormats {
			siteRenderContext.outIdx = siteOutIdx
			siteRenderContext.sitesOutIdx = i
			i++

			for _, s2 := range h.Sites {
				// We render site by site, but since the content is lazily rendered
				// and a site can "borrow" content from other sites, every site
				// needs this set.
				s2.rc = &siteRenderingContext{Format: renderFormat}

				// Get page output ready
				if err := s2.preparePagesForRender(s == s2, siteRenderContext.sitesOutIdx); err != nil {
					return err
				}
			}

			if err := s.render(siteRenderContext); err != nil {
				return err
			}
		}
	}
	if err := h.renderCrossSitesRobotsTXT(); err != nil {
		return err
	}

	return nil
}

func (h *HugoSites) renderCrossSitesRobotsTXT() error {
	s := h.Sites[0]

	p, err := newPageStandalone(&pageMeta{
		s:    s,
		kind: kindRobotsTXT,
		urlPaths: pagemeta.URLPath{
			URL: "robots.txt",
		},
	},
		output.RobotsTxtFormat)
	if err != nil {
		return err
	}

	if !p.render {
		return nil
	}

	templ := s.lookupLayouts("robots.txt", "_default/robots.txt", "_internal/_default/robots.txt")

	return s.renderAndWritePage("Robots Txt", "robots.txt", p, templ)
}
