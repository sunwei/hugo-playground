package hugolib

import (
	"github.com/sunwei/hugo-playground/compare"
	"github.com/sunwei/hugo-playground/lazy"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/resources/page"
	"github.com/sunwei/hugo-playground/resources/resource"
	"sync"
)

type pageCommon struct {
	s *Site
	m *pageMeta

	bucket  *pagesMapBucket
	treeRef *contentTreeRef

	// Lazily initialized dependencies.
	init *lazy.Init

	// All of these represents the common parts of a page.Page
	page.ChildCareProvider
	page.FileProvider
	page.OutputFormatsProvider
	page.PageMetaProvider
	page.SitesProvider
	page.TreeProvider
	resource.LanguageProvider
	resource.ResourceMetaProvider
	resource.ResourceParamsProvider
	resource.ResourceTypeProvider
	compare.Eqer

	// Describes how paths and URLs for this page and its descendants
	// should look like.
	targetPathDescriptor page.TargetPathDescriptor

	layoutDescriptor     output.LayoutDescriptor
	layoutDescriptorInit sync.Once

	// The parsed page content.
	pageContent

	// Any bundled resources
	resources            resource.Resources
	resourcesInit        sync.Once
	resourcesPublishInit sync.Once

	translations    page.Pages
	allTranslations page.Pages

	// Calculated an cached translation mapping key
	translationKey     string
	translationKeyInit sync.Once

	// Will only be set for bundled pages.
	parent *pageState

	// Set in fast render mode to force render a given page.
	forceRender bool
}

type treeRefProvider interface {
	getTreeRef() *contentTreeRef
}

func (p *pageCommon) getTreeRef() *contentTreeRef {
	return p.treeRef
}
