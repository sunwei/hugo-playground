package hugolib

import (
	"context"
	"fmt"
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugofs/files"
	"github.com/sunwei/hugo-playground/source"
	"os"
	"path/filepath"
)

type fileinfoBundle struct {
	header    hugofs.FileMetaInfo
	resources []hugofs.FileMetaInfo
}

func newPagesCollector(
	sp *source.SourceSpec,
	contentMap *pageMaps,
	proc pagesCollectorProcessorProvider, filenames ...string) *pagesCollector {

	return &pagesCollector{
		fs:         sp.SourceFs,
		contentMap: contentMap,
		proc:       proc,
		sp:         sp,
		filenames:  filenames,
	}
}

type pagesCollector struct {
	sp *source.SourceSpec
	fs afero.Fs

	contentMap *pageMaps

	// Ordered list (bundle headers first) used in partial builds.
	filenames []string

	// Content files tracker used in partial builds.
	tracker *contentChangeMap

	proc pagesCollectorProcessorProvider
}

// Collect pages.
func (c *pagesCollector) Collect() (collectErr error) {
	c.proc.Start(context.Background())
	defer func() {
		err := c.proc.Wait()
		if collectErr == nil {
			collectErr = err
		}
	}()

	if len(c.filenames) == 0 {
		// Collect everything.
		collectErr = c.collectDir("")
	} else {
		panic("not support collect partial")
	}

	return
}

type bundleDirType int

const (
	bundleNot bundleDirType = iota

	// All from here are bundles in one form or another.
	bundleLeaf
	bundleBranch
)

func (c *pagesCollector) collectDir(dirname string) error {
	fi, err := c.fs.Stat(dirname)

	if err != nil {
		if os.IsNotExist(err) {
			// May have been deleted.
			return nil
		}
		return err
	}

	handleDir := func(
		btype bundleDirType,
		dir hugofs.FileMetaInfo,
		path string,
		readdir []hugofs.FileMetaInfo) error {

		if err := c.handleFiles(readdir...); err != nil {
			return err
		}

		return nil
	}

	filter := func(fim hugofs.FileMetaInfo) bool {
		if fim.Meta().SkipDir {
			return false
		}

		if c.sp.IgnoreFile(fim.Meta().Filename) {
			return false
		}

		return true
	}

	preHook := func(dir hugofs.FileMetaInfo, path string, readdir []hugofs.FileMetaInfo) ([]hugofs.FileMetaInfo, error) {
		var btype bundleDirType

		filtered := readdir[:0]
		for _, fi := range readdir {
			if filter(fi) {
				filtered = append(filtered, fi)
			}
		}
		readdir = filtered

		err := handleDir(btype, dir, path, readdir)
		if err != nil {
			return nil, err
		}

		if btype == bundleLeaf {
			return nil, filepath.SkipDir
		}

		// Keep walking.
		return readdir, nil
	}

	wfn := func(path string, info hugofs.FileMetaInfo, err error) error {
		if err != nil {
			return err
		}

		return nil
	}

	fim := fi.(hugofs.FileMetaInfo)
	// Make sure the pages in this directory gets re-rendered,
	// even in fast render mode.
	fim.Meta().IsRootFile = true

	w := hugofs.NewWalkway(hugofs.WalkwayConfig{
		Fs:       c.fs,
		Root:     dirname,
		Info:     fim,
		HookPre:  preHook,
		HookPost: nil,
		WalkFn:   wfn,
	})

	// directory walk only
	return w.Walk()
}

func (c *pagesCollector) handleBundleBranch(readdir []hugofs.FileMetaInfo) error {
	// Maps bundles to its language.
	bundles := pageBundles{}

	var contentFiles []hugofs.FileMetaInfo

	for _, fim := range readdir {

		if fim.IsDir() {
			continue
		}

		meta := fim.Meta()

		switch meta.Classifier {
		case files.ContentClassContent:
			contentFiles = append(contentFiles, fim)
		default:
			if err := c.addToBundle(fim, bundleBranch, bundles); err != nil {
				return err
			}
		}

	}

	// Make sure the section is created before its pages.
	if err := c.proc.Process(bundles); err != nil {
		return err
	}

	return c.handleFiles(contentFiles...)
}

func (c *pagesCollector) addToBundle(info hugofs.FileMetaInfo, btyp bundleDirType, bundles pageBundles) error {
	getBundle := func(lang string) *fileinfoBundle {
		return bundles[lang]
	}

	cloneBundle := func(lang string) *fileinfoBundle {
		// Every bundled content file needs a content file header.
		// Use the default content language if found, else just
		// pick one.
		var (
			source *fileinfoBundle
			found  bool
		)

		source, found = bundles[c.sp.DefaultContentLanguage]
		if !found {
			for _, b := range bundles {
				source = b
				break
			}
		}

		if source == nil {
			panic(fmt.Sprintf("no source found, %d", len(bundles)))
		}

		clone := c.cloneFileInfo(source.header)

		return &fileinfoBundle{
			header: clone,
		}
	}

	lang := c.getLang(info)
	bundle := getBundle(lang)
	isBundleHeader := c.isBundleHeader(info)
	if bundle != nil && isBundleHeader {
		// index.md file inside a bundle, see issue 6208.
		info.Meta().Classifier = files.ContentClassContent
		isBundleHeader = false
	}
	classifier := info.Meta().Classifier
	isContent := classifier == files.ContentClassContent
	if bundle == nil {
		if isBundleHeader {
			bundle = &fileinfoBundle{header: info}
			bundles[lang] = bundle
		} else {
			if btyp == bundleBranch {
				// No special logic for branch bundles.
				// Every language needs its own _index.md file.
				// Also, we only clone bundle headers for lonesome, bundled,
				// content files.
				return c.handleFiles(info)
			}

			if isContent {
				bundle = cloneBundle(lang)
				bundles[lang] = bundle
			}
		}
	}

	if !isBundleHeader && bundle != nil {
		bundle.resources = append(bundle.resources, info)
	}

	if classifier == files.ContentClassFile {

		for _, b := range bundles {
			if !b.containsResource(info.Name()) {

				// Clone and add it to the bundle.
				clone := c.cloneFileInfo(info)
				b.resources = append(b.resources, clone)
			}
		}
	}

	return nil
}

func (c *pagesCollector) cloneFileInfo(fi hugofs.FileMetaInfo) hugofs.FileMetaInfo {
	return hugofs.NewFileMetaInfo(fi, hugofs.NewFileMeta())
}

func (c *pagesCollector) getLang(fi hugofs.FileMetaInfo) string {
	return c.sp.DefaultContentLanguage
}

func (c *pagesCollector) isBundleHeader(fi hugofs.FileMetaInfo) bool {
	class := fi.Meta().Classifier
	return class == files.ContentClassLeaf || class == files.ContentClassBranch
}

func (c *pagesCollector) handleFiles(fis ...hugofs.FileMetaInfo) error {
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}

		if err := c.proc.Process(fi); err != nil {
			return err
		}
	}
	return nil
}

func (b *fileinfoBundle) containsResource(name string) bool {
	for _, r := range b.resources {
		if r.Name() == name {
			return true
		}
	}

	return false
}
