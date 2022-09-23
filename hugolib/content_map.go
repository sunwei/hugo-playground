package hugolib

import (
	"fmt"
	"github.com/armon/go-radix"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugofs/files"
	"github.com/sunwei/hugo-playground/resources/page"
	"path"
	"path/filepath"
	"strings"
)

type contentTree struct {
	Name string
	*radix.Tree
}

type contentTrees []*contentTree

type contentMap struct {
	// View of regular pages, sections, and taxonomies.
	pageTrees contentTrees

	// View of pages, sections, taxonomies, and resources.
	bundleTrees contentTrees

	// Stores page bundles keyed by its path's directory or the base filename,
	// e.g. "blog/post.md" => "/blog/post", "blog/post/index.md" => "/blog/post"
	// These are the "regular pages" and all of them are bundles.
	pages *contentTree

	// Section nodes.
	sections *contentTree

	// Resources stored per bundle below a common prefix, e.g. "/blog/post__hb_".
	resources *contentTree
}

// contentTreeRef points to a node in the given tree.
type contentTreeRef struct {
	m   *pageMap
	t   *contentTree
	n   *contentNode
	key string
}

type contentNode struct {
	p *pageState

	// Set for taxonomy nodes.
	viewInfo *contentBundleViewInfo

	// Set if source is a file.
	// We will soon get other sources.
	fi hugofs.FileMetaInfo

	// The source path. Unix slashes. No leading slash.
	path string
}

type contentBundleViewInfo struct {
	ordinal    int
	name       viewName
	termKey    string
	termOrigin string
	weight     int
	ref        *contentNode
}

type contentTreeNodeCallback func(s string, n *contentNode) bool

var (
	contentTreeNoListAlwaysFilter = func(s string, n *contentNode) bool {
		if n.p == nil {
			return true
		}
		return n.p.m.noListAlways()
	}

	contentTreeNoRenderFilter = func(s string, n *contentNode) bool {
		if n.p == nil {
			return true
		}
		return n.p.m.noRender()
	}
)

func (c contentTrees) WalkRenderable(fn contentTreeNodeCallback) {
	query := pageMapQuery{Filter: contentTreeNoRenderFilter}
	for _, tree := range c {
		tree.WalkQuery(query, fn)
	}
}

func (c *contentTree) WalkQuery(query pageMapQuery, walkFn contentTreeNodeCallback) {
	filter := query.Filter
	if filter == nil {
		filter = contentTreeNoListAlwaysFilter
	}
	if query.Prefix != "" {
		c.WalkBelow(query.Prefix, func(s string, v any) bool {
			n := v.(*contentNode)
			if filter != nil && filter(s, n) {
				return false
			}
			return walkFn(s, n)
		})

		return
	}

	c.Walk(func(s string, v any) bool {
		n := v.(*contentNode)
		if filter != nil && filter(s, n) {
			return false
		}
		return walkFn(s, n)
	})
}

// WalkBelow walks the tree below the given prefix, i.e. it skips the
// node with the given prefix as key.
func (c *contentTree) WalkBelow(prefix string, fn radix.WalkFn) {
	c.Tree.WalkPrefix(prefix, func(s string, v any) bool {
		if s == prefix {
			return false
		}
		return fn(s, v)
	})
}

func newContentMap() *contentMap {
	m := &contentMap{
		pages:     &contentTree{Name: "pages", Tree: radix.New()},
		sections:  &contentTree{Name: "sections", Tree: radix.New()},
		resources: &contentTree{Name: "resources", Tree: radix.New()},
	}

	m.pageTrees = []*contentTree{
		m.pages, m.sections,
	}

	m.bundleTrees = []*contentTree{
		m.pages, m.sections,
	}

	return m
}

const (
	cmBranchSeparator = "__hb_"
	cmLeafSeparator   = "__hl_"
)

func (c contentTrees) Walk(fn contentTreeNodeCallback) {
	for _, tree := range c {
		tree.Walk(func(s string, v any) bool {
			n := v.(*contentNode)
			return fn(s, n)
		})
	}
}

func (m *contentMap) AddFilesBundle(header hugofs.FileMetaInfo, resources ...hugofs.FileMetaInfo) error {
	var (
		meta       = header.Meta()
		bundlePath = m.getBundleDir(meta)

		n = m.newContentNodeFromFi(header)
		b = m.newKeyBuilder()

		section string
	)

	// A regular page. Attach it to its section.
	section, _ = m.getOrCreateSection(n, bundlePath) // /abc/
	b = b.WithSection(section).ForPage(bundlePath).Insert(n)

	return nil
}

func (m *contentMap) getBundleDir(meta *hugofs.FileMeta) string {
	dir := cleanTreeKey(filepath.Dir(meta.Path))

	switch meta.Classifier {
	case files.ContentClassContent:
		return path.Join(dir, meta.TranslationBaseName)
	default:
		return dir
	}
}

func cleanTreeKey(k string) string {
	k = "/" + strings.ToLower(strings.Trim(path.Clean(filepath.ToSlash(k)), "./"))
	return k
}

func (m *contentMap) newContentNodeFromFi(fi hugofs.FileMetaInfo) *contentNode {
	return &contentNode{
		fi:   fi,
		path: strings.TrimPrefix(filepath.ToSlash(fi.Meta().Path), "/"),
	}
}

func (m *contentMap) newKeyBuilder() *cmInsertKeyBuilder {
	return &cmInsertKeyBuilder{m: m}
}

type cmInsertKeyBuilder struct {
	m *contentMap

	err error

	// Builder state
	tree    *contentTree
	baseKey string // Section or page key
	key     string
}

func (m *contentMap) getOrCreateSection(n *contentNode, s string) (string, *contentNode) {
	level := strings.Count(s, "/")
	k, b := m.getSection(s)

	mustCreate := false

	if k == "" {
		mustCreate = true
	} else if level > 1 && k == "/" {
		// We found the home section, but this page needs to be placed in
		// the root, e.g. "/blog", section.
		mustCreate = true
	}

	if mustCreate {
		k = cleanSectionTreeKey(s[:strings.Index(s[1:], "/")+1])

		b = &contentNode{
			path: n.rootSection(),
		}

		m.sections.Insert(k, b)
	}

	return k, b
}

func (m *contentMap) getSection(s string) (string, *contentNode) {
	s = helpers.AddTrailingSlash(path.Dir(strings.TrimSuffix(s, "/")))

	k, v, found := m.sections.LongestPrefix(s)

	if found {
		return k, v.(*contentNode)
	}
	return "", nil
}

func cleanSectionTreeKey(k string) string {
	k = cleanTreeKey(k)
	if k != "/" {
		k += "/"
	}

	return k
}

func (b *contentNode) rootSection() string {
	if b.path == "" {
		return ""
	}
	firstSlash := strings.Index(b.path, "/")
	if firstSlash == -1 {
		return b.path
	}
	return b.path[:firstSlash]
}

func (b *cmInsertKeyBuilder) WithSection(s string) *cmInsertKeyBuilder {
	s = cleanSectionTreeKey(s)
	b.newTopLevel()
	b.tree = b.m.sections
	b.baseKey = s
	b.key = s
	return b
}

func (b *cmInsertKeyBuilder) newTopLevel() {
	b.key = ""
}

func (b cmInsertKeyBuilder) ForPage(s string) *cmInsertKeyBuilder {
	baseKey := b.baseKey
	b.baseKey = s

	if baseKey != "/" {
		// Don't repeat the section path in the key.
		s = strings.TrimPrefix(s, baseKey)
	}
	s = strings.TrimPrefix(s, "/")

	switch b.tree {
	case b.m.sections:
		b.tree = b.m.pages
		b.key = baseKey + cmBranchSeparator + s + cmLeafSeparator
	default:
		panic("invalid state")
	}

	return &b
}

func (b *cmInsertKeyBuilder) Insert(n *contentNode) *cmInsertKeyBuilder {
	if b.err == nil {
		b.tree.Insert(b.Key(), n)
	}
	return b
}

func (b *cmInsertKeyBuilder) Key() string {
	switch b.tree {
	case b.m.sections:
		return cleanSectionTreeKey(b.key)
	default:
		return cleanTreeKey(b.key)
	}
}

func (b cmInsertKeyBuilder) ForResource(s string) *cmInsertKeyBuilder {
	baseKey := helpers.AddTrailingSlash(b.baseKey)
	s = strings.TrimPrefix(s, baseKey)

	switch b.tree {
	case b.m.pages:
		b.key = b.key + s
	case b.m.sections:
		b.key = b.key + cmLeafSeparator + s
	default:
		panic(fmt.Sprintf("invalid state: %#v", b.tree))
	}
	b.tree = b.m.resources
	return &b
}

func (m *contentMap) CreateMissingNodes() error {
	// Create missing home and root sections
	rootSections := make(map[string]any)
	rootSections["/"] = true // not found in both sections and pages

	for sect, _ := range rootSections {
		var sectionPath string
		sect = cleanSectionTreeKey(sect)

		_, found := m.sections.Get(sect)
		if !found {
			mm := &contentNode{path: sectionPath} // ""
			_, _ = m.sections.Insert(sect, mm)    // "/"
		}
	}

	return nil
}

func (m *contentMap) splitKey(k string) []string {
	if k == "" || k == "/" {
		return nil
	}

	return strings.Split(k, "/")[1:]
}

func (m *contentMap) getPage(section, name string) *contentNode {
	section = helpers.AddTrailingSlash(section)
	key := section + cmBranchSeparator + name + cmLeafSeparator

	v, found := m.pages.Get(key)
	if found {
		return v.(*contentNode)
	}
	return nil
}

func (m *contentMap) onSameLevel(s1, s2 string) bool {
	return strings.Count(s1, "/") == strings.Count(s2, "/")
}

func (c *contentTreeRef) isSection() bool {
	return c.t == c.m.sections
}

func (m *contentMap) getFirstSection(s string) (string, *contentNode) {
	s = helpers.AddTrailingSlash(s)
	for {
		k, v, found := m.sections.LongestPrefix(s)

		if !found {
			return "", nil
		}

		if strings.Count(k, "/") <= 2 {
			return k, v.(*contentNode)
		}

		s = helpers.AddTrailingSlash(path.Dir(strings.TrimSuffix(s, "/")))

	}
}

func (c *contentTreeRef) getCurrentSection() (string, *contentNode) {
	if c.isSection() {
		return c.key, c.n
	}
	return c.getSection()
}

func (c *contentTreeRef) getSection() (string, *contentNode) {
	return c.m.getSection(c.key)
}

func (m *contentMap) getTaxonomyParent(s string) (string, *contentNode) {
	s = helpers.AddTrailingSlash(path.Dir(strings.TrimSuffix(s, "/")))

	v, found := m.sections.Get("/")
	if found {
		return s, v.(*contentNode)
	}

	return "", nil
}

func (c *contentTreeRef) getSections() page.Pages {
	var pas page.Pages

	query := pageMapQuery{
		Filter: c.n.p.m.getListFilter(true),
		Prefix: c.key,
	}

	c.m.collectSections(query, func(c *contentNode) {
		pas = append(pas, c.p)
	})

	page.SortByDefault(pas)

	return pas
}

func newContentTreeFilter(fn func(n *contentNode) bool) contentTreeNodeCallback {
	return func(s string, n *contentNode) bool {
		return fn(n)
	}
}

func (m *contentMap) deleteSectionByPath(s string) {
	if !strings.HasSuffix(s, "/") {
		panic("section must end with a slash")
	}
	if !strings.HasPrefix(s, "/") {
		panic("section must start with a slash")
	}
	m.sections.DeletePrefix(s)
	m.pages.DeletePrefix(s)
	m.resources.DeletePrefix(s)
}

// Deletes any empty root section that's not backed by a content file.
func (m *contentMap) deleteOrphanSections() {
	var sectionsToDelete []string

	m.sections.Walk(func(s string, v any) bool {
		n := v.(*contentNode)

		if n.fi != nil {
			// Section may be empty, but is backed by a content file.
			return false
		}

		if s == "/" || strings.Count(s, "/") > 2 {
			return false
		}

		prefixBundle := s + cmBranchSeparator

		if !(m.sections.hasBelow(s) || m.pages.hasBelow(prefixBundle) || m.resources.hasBelow(prefixBundle)) {
			sectionsToDelete = append(sectionsToDelete, s)
		}

		return false
	})

	for _, s := range sectionsToDelete {
		m.sections.Delete(s)
	}
}

func (c *contentTree) hasBelow(s1 string) bool {
	var t bool
	c.WalkBelow(s1, func(s2 string, v any) bool {
		t = true
		return true
	})
	return t
}
