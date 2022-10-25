package hugolib

import (
	"github.com/sunwei/hugo-playground/log"
)

// Build builds all sites. If filesystem events are provided,
// this is considered to be a potential partial rebuild.
// ---
// Let's focus on full build, remove events
func (h *HugoSites) Build(config BuildCfg) error {
	log.Process("HugoSites Build", "start")
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

	log.Process("HugoSites Build", "done")
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
