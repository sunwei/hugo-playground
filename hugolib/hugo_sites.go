package hugolib

import (
	"context"
	"fmt"
	"github.com/armon/go-radix"
	"github.com/sunwei/hugo-playground/common/para"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/deps"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugofs/glob"
	"github.com/sunwei/hugo-playground/lazy"
	"github.com/sunwei/hugo-playground/log"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/parser/metadecoders"
	"github.com/sunwei/hugo-playground/publisher"
	"github.com/sunwei/hugo-playground/source"
	"github.com/sunwei/hugo-playground/tpl"
	"github.com/sunwei/hugo-playground/tpl/tplimpl"
	"strings"
	"sync"
)

// BuildCfg holds build options used to, as an example, skip the render step.
type BuildCfg struct {
	// Can be set to build only with a sub set of the content source.
	ContentInclusionFilter *glob.FilenameFilter
}

type hugoSitesInit struct {
	// Loads the data from all of the /data folders.
	data *lazy.Init

	// Performs late initialization (before render) of the templates.
	layouts *lazy.Init
}

// HugoSites represents the sites to build. Each site represents a language.
type HugoSites struct {
	Sites []*Site
	// Render output formats for all sites.
	renderFormats output.Formats

	// The currently rendered Site.
	currentSite *Site

	*deps.Deps

	contentInit sync.Once
	content     *pageMaps

	init *hugoSitesInit

	workers    *para.Workers
	numWorkers int

	// As loaded from the /data dirs
	data map[string]any
}

// NewHugoSites creates HugoSites from the given config.
func NewHugoSites(cfg deps.DepsCfg) (*HugoSites, error) {
	sites, err := createSitesFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("from config: %w", err)
	}

	return newHugoSites(cfg, sites...)
}

func createSitesFromConfig(cfg deps.DepsCfg) ([]*Site, error) {
	log.Process("createSitesFromConfig", "start")
	var sites []*Site

	// [en]
	languages := getLanguages(cfg.Cfg)
	for _, lang := range languages {
		var s *Site
		var err error
		cfg.Language = lang
		log.Process("newSite", "create site with DepsCfg with language setup")
		s, err = newSite(cfg)

		if err != nil {
			return nil, err
		}

		sites = append(sites, s)
	}

	log.Process("createSitesFromConfig", "end")
	return sites, nil
}

// NewHugoSites creates a new collection of sites given the input sites, building
// a language configuration based on those.
func newHugoSites(cfg deps.DepsCfg, sites ...*Site) (*HugoSites, error) {
	// Return error at the end. Make the caller decide if it's fatal or not.
	var initErr error

	// 3
	log.Process("newHugoSites", "get number of worker")
	numWorkers := config.GetNumWorkerMultiplier()
	if numWorkers > len(sites) { // sites [en]: 1
		numWorkers = len(sites)
	}

	var workers *para.Workers

	log.Process("newHugoSites", "init HugoSites")
	h := &HugoSites{
		Sites:      sites,
		workers:    workers,    // nil
		numWorkers: numWorkers, // 1
		init: &hugoSitesInit{
			data:    lazy.New(),
			layouts: lazy.New(),
		},
	}

	log.Process("newHugoSites", "add data to h.init")
	h.init.data.Add(func() (any, error) {
		log.Process("newHugoSites", "h.init run h.loadData")
		err := h.loadData(h.PathSpec.BaseFs.Data.Dirs)
		if err != nil {
			return nil, fmt.Errorf("failed to load data: %w", err)
		}
		return nil, nil
	})

	log.Process("newHugoSites", "add layouts to h.init")
	h.init.layouts.Add(func() (any, error) {
		log.Process("newHugoSites", "h.init run s.Tmpl().MarkReady")
		for _, s := range h.Sites {
			if err := s.Tmpl().(tpl.TemplateManager).MarkReady(); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	for _, s := range sites {
		s.h = h
	}

	log.Process("newHugoSites", "configLoader applyDeps")
	var l configLoader
	if err := l.applyDeps(cfg, sites...); err != nil {
		initErr = fmt.Errorf("add site dependencies: %w", err)
	}

	h.Deps = sites[0].Deps
	if h.Deps == nil {
		return nil, initErr
	}

	return h, initErr
}

func (h *HugoSites) loadData(fis []hugofs.FileMetaInfo) (err error) {
	spec := source.NewSourceSpec(h.PathSpec, nil, nil)

	h.data = make(map[string]any)
	for _, fi := range fis {
		fileSystem := spec.NewFilesystemFromFileMetaInfo(fi)
		files, err := fileSystem.Files()
		if err != nil {
			return err
		}
		for _, r := range files {
			if err := h.handleDataFile(r); err != nil {
				return err
			}
		}
	}

	return
}

func (h *HugoSites) handleDataFile(r source.File) error {
	var current map[string]any

	f, err := r.FileInfo().Meta().Open()
	if err != nil {
		return fmt.Errorf("data: failed to open %q: %w", r.LogicalName(), err)
	}
	defer f.Close()

	// Crawl in data tree to insert data
	current = h.data
	keyParts := strings.Split(r.Dir(), helpers.FilePathSeparator)

	for _, key := range keyParts {
		if key != "" {
			if _, ok := current[key]; !ok {
				current[key] = make(map[string]any)
			}
			current = current[key].(map[string]any)
		}
	}

	data, err := h.readData(r)
	if err != nil {
		return err
	}

	if data == nil {
		return nil
	}

	// filepath.Walk walks the files in lexical order, '/' comes before '.'
	higherPrecedentData := current[r.BaseFileName()]

	switch data.(type) {
	case nil:
	case map[string]any:

		switch higherPrecedentData.(type) {
		case nil:
			current[r.BaseFileName()] = data
		case map[string]any:
			// merge maps: insert entries from data for keys that
			// don't already exist in higherPrecedentData
			higherPrecedentMap := higherPrecedentData.(map[string]any)
			for key, value := range data.(map[string]any) {
				if _, exists := higherPrecedentMap[key]; exists {
					// this warning could happen if
					// 1. A theme uses the same key; the main data folder wins
					// 2. A sub folder uses the same key: the sub folder wins
					// TODO(bep) figure out a way to detect 2) above and make that a WARN
					fmt.Printf("Data for key '%s' in path '%s' is overridden by higher precedence data already in the data tree", key, r.Path())
				} else {
					higherPrecedentMap[key] = value
				}
			}
		default:
			// can't merge: higherPrecedentData is not a map
			fmt.Printf("The %T data from '%s' overridden by "+
				"higher precedence %T data already in the data tree", data, r.Path(), higherPrecedentData)
		}

	case []any:
		if higherPrecedentData == nil {
			current[r.BaseFileName()] = data
		} else {
			// we don't merge array data
			fmt.Printf("The %T data from '%s' overridden by "+
				"higher precedence %T data already in the data tree", data, r.Path(), higherPrecedentData)
		}

	default:
		fmt.Printf("unexpected data type %T in file %s", data, r.LogicalName())
	}

	return nil
}

func (l configLoader) applyDeps(cfg deps.DepsCfg, sites ...*Site) error {
	log.Process("applyDeps", "set cfg.TemplateProvider with DefaultTemplateProvider")
	if cfg.TemplateProvider == nil {
		cfg.TemplateProvider = tplimpl.DefaultTemplateProvider
	}

	var (
		d *deps.Deps
	)

	for _, s := range sites {
		if s.Deps != nil {
			continue
		}

		onCreated := func(d *deps.Deps) error {
			s.Deps = d

			log.Process("applyDeps-onCreate", "set site publisher as DestinationPublisher")
			// Set up the main publishing chain.
			pub, err := publisher.NewDestinationPublisher(
				d.ResourceSpec,
				s.outputFormatsConfig,
				s.mediaTypesConfig,
			)
			if err != nil {
				return err
			}
			s.publisher = pub

			log.Process("applyDeps-onCreate site initializeSiteInfo", "set site title and owner")
			if err := s.initializeSiteInfo(); err != nil {
				return err
			}

			log.Process("applyDeps-onCreate pageMap", "with pageTree, bundleTree and pages, sections, resources")
			pm := &pageMap{
				contentMap: newContentMap(),
				s:          s,
			}

			log.Process("applyDeps-onCreate site PageCollections", "with pageMap")
			s.PageCollections = newPageCollections(pm)
			return err
		}

		cfg.Language = s.language
		cfg.MediaTypes = s.mediaTypesConfig
		cfg.OutputFormats = s.outputFormatsConfig

		var err error
		log.Process("applyDeps", "new deps")
		d, err = deps.New(cfg)
		if err != nil {
			return fmt.Errorf("create deps: %w", err)
		}

		d.OutputFormatsConfig = s.outputFormatsConfig

		if err := onCreated(d); err != nil {
			return fmt.Errorf("on created: %w", err)
		}

		log.Process("applyDeps", "deps LoadResources to update template provider, need to make template ready")
		if err = d.LoadResources(); err != nil {
			return fmt.Errorf("load resources: %w", err)
		}
	}

	return nil
}

func (h *HugoSites) readData(f source.File) (any, error) {
	file, err := f.FileInfo().Meta().Open()
	if err != nil {
		return nil, fmt.Errorf("readData: failed to open data file: %w", err)
	}
	defer file.Close()
	content := helpers.ReaderToBytes(file)

	format := metadecoders.FormatFromString(f.Ext())
	return metadecoders.Default.Unmarshal(content, format)
}

func (h *HugoSites) Data() map[string]any {
	if _, err := h.init.data.Do(); err != nil {
		fmt.Errorf("failed to load data: %w", err)
		return nil
	}
	return h.data
}

func (s *Site) withSiteTemplates(withTemplates ...func(templ tpl.TemplateManager) error) func(templ tpl.TemplateManager) error {
	return func(templ tpl.TemplateManager) error {
		for _, wt := range withTemplates {
			if wt == nil {
				continue
			}
			if err := wt(templ); err != nil {
				return err
			}
		}

		return nil
	}
}

// Used in partial reloading to determine if the change is in a bundle.
type contentChangeMap struct {
	mu sync.RWMutex

	// Holds directories with leaf bundles.
	leafBundles *radix.Tree

	// Holds directories with branch bundles.
	branchBundles map[string]bool

	pathSpec *helpers.PathSpec

	// Hugo supports symlinked content (both directories and files). This
	// can lead to situations where the same file can be referenced from several
	// locations in /content -- which is really cool, but also means we have to
	// go an extra mile to handle changes.
	// This map is only used in watch mode.
	// It maps either file to files or the real dir to a set of content directories
	// where it is in use.
	symContentMu sync.Mutex
	symContent   map[string]map[string]bool
}

func (h *HugoSites) getContentMaps() *pageMaps {
	h.contentInit.Do(func() {
		h.content = newPageMaps(h)
	})
	return h.content
}

func (h *HugoSites) createPageCollections() error {
	return nil
}

func (h *HugoSites) withSite(fn func(s *Site) error) error {
	if h.workers == nil {
		for _, s := range h.Sites {
			if err := fn(s); err != nil {
				return err
			}
		}
		return nil
	}

	g, _ := h.workers.Start(context.Background())
	for _, s := range h.Sites {
		s := s
		g.Run(func() error {
			return fn(s)
		})
	}
	return g.Wait()
}

// shouldRender is used in the Fast Render Mode to determine if we need to re-render
// a Page: If it is recently visited (the home pages will always be in this set) or changed.
// Note that a page does not have to have a content page / file.
// For regular builds, this will allways return true.
// TODO(bep) rename/work this.
func (cfg *BuildCfg) shouldRender(p *pageState) bool {
	return true
}

func (h *HugoSites) pickOneAndLogTheRest(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	var i int

	// Log the rest, but add a threshold to avoid flooding the log.
	const errLogThreshold = 5

	for j, err := range errors {
		if j == i || err == nil {
			continue
		}

		if j >= errLogThreshold {
			break
		}

		h.Log.Errorln(err)
	}

	return errors[i]
}
