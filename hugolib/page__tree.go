package hugolib

import (
	"github.com/sunwei/hugo-playground/common/types"
	"github.com/sunwei/hugo-playground/resources/page"
	"path"
	"strings"
)

type pageTree struct {
	p *pageState
}

func (pt pageTree) IsAncestor(other any) (bool, error) {
	if pt.p == nil {
		return false, nil
	}

	tp, ok := other.(treeRefProvider)
	if !ok {
		return false, nil
	}

	ref1, ref2 := pt.p.getTreeRef(), tp.getTreeRef()
	if ref1 != nil && ref2 != nil && ref1.key == ref2.key {
		return false, nil
	}

	if ref1 != nil && ref1.key == "/" {
		return true, nil
	}

	if ref1 == nil || ref2 == nil {
		if ref1 == nil {
			// A 404 or other similar standalone page.
			return false, nil
		}

		return ref1.n.p.IsHome(), nil
	}

	if strings.HasPrefix(ref2.key, ref1.key) {
		return true, nil
	}

	return strings.HasPrefix(ref2.key, ref1.key+cmBranchSeparator), nil
}

func (pt pageTree) CurrentSection() page.Page {
	p := pt.p

	if p.IsHome() || p.IsSection() {
		return p
	}

	return p.Parent()
}

func (pt pageTree) IsDescendant(other any) (bool, error) {
	if pt.p == nil {
		return false, nil
	}

	tp, ok := other.(treeRefProvider)
	if !ok {
		return false, nil
	}

	ref1, ref2 := pt.p.getTreeRef(), tp.getTreeRef()
	if ref1 != nil && ref2 != nil && ref1.key == ref2.key {
		return false, nil
	}

	if ref2 != nil && ref2.key == "/" {
		return true, nil
	}

	if ref1 == nil || ref2 == nil {
		if ref2 == nil {
			// A 404 or other similar standalone page.
			return false, nil
		}

		return ref2.n.p.IsHome(), nil
	}

	if strings.HasPrefix(ref1.key, ref2.key) {
		return true, nil
	}

	return strings.HasPrefix(ref1.key, ref2.key+cmBranchSeparator), nil
}

func (pt pageTree) FirstSection() page.Page {
	ref := pt.p.getTreeRef()
	if ref == nil {
		return pt.p.s.home
	}
	key := ref.key

	if !ref.isSection() {
		key = path.Dir(key)
	}

	_, b := ref.m.getFirstSection(key)
	if b == nil {
		return nil
	}
	return b.p
}

func (pt pageTree) InSection(other any) (bool, error) {
	if pt.p == nil || types.IsNil(other) {
		return false, nil
	}

	tp, ok := other.(treeRefProvider)
	if !ok {
		return false, nil
	}

	ref1, ref2 := pt.p.getTreeRef(), tp.getTreeRef()

	if ref1 == nil || ref2 == nil {
		if ref1 == nil {
			// A 404 or other similar standalone page.
			return false, nil
		}
		return ref1.n.p.IsHome(), nil
	}

	s1, _ := ref1.getCurrentSection()
	s2, _ := ref2.getCurrentSection()

	return s1 == s2, nil
}

func (pt pageTree) Page() page.Page {
	return pt.p
}

func (pt pageTree) Parent() page.Page {
	p := pt.p

	if p.parent != nil {
		return p.parent
	}

	if pt.p.IsHome() {
		return nil
	}

	tree := p.getTreeRef()

	if tree == nil || pt.p.Kind() == page.KindTaxonomy {
		return pt.p.s.home
	}

	_, b := tree.getSection()
	if b == nil {
		return nil
	}

	return b.p
}

func (pt pageTree) Sections() page.Pages {
	if pt.p.bucket == nil {
		return nil
	}

	return pt.p.bucket.getSections()
}
