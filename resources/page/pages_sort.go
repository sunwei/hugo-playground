package page

import (
	"github.com/sunwei/hugo-playground/common/collections"
	"github.com/sunwei/hugo-playground/compare"
	"sort"
)

// SortByDefault sorts pages by the default sort.
func SortByDefault(pages Pages) {
	pageBy(DefaultPageSort).Sort(pages)
}

var (
	// DefaultPageSort is the default sort func for pages in Hugo:
	// Order by Ordinal, Weight, Date, LinkTitle and then full file path.
	DefaultPageSort = func(p1, p2 Page) bool {
		o1, o2 := getOrdinals(p1, p2)
		if o1 != o2 && o1 != -1 && o2 != -1 {
			return o1 < o2
		}
		if p1.Weight() == p2.Weight() {
			if p1.Date().Unix() == p2.Date().Unix() {
				c := collatorStringCompare(func(p Page) string { return p.LinkTitle() }, p1, p2)
				if c == 0 {
					if p1.File().IsZero() || p2.File().IsZero() {
						return p1.File().IsZero()
					}
					return compare.LessStrings(p1.File().Filename(), p2.File().Filename())
				}
				return c < 0
			}
			return p1.Date().Unix() > p2.Date().Unix()
		}

		if p2.Weight() == 0 {
			return true
		}

		if p1.Weight() == 0 {
			return false
		}

		return p1.Weight() < p2.Weight()
	}
)

func getOrdinals(p1, p2 Page) (int, int) {
	p1o, ok1 := p1.(collections.Order)
	if !ok1 {
		return -1, -1
	}
	p2o, ok2 := p2.(collections.Order)
	if !ok2 {
		return -1, -1
	}

	return p1o.Ordinal(), p2o.Ordinal()
}

// pageBy is a closure used in the Sort.Less method.
type pageBy func(p1, p2 Page) bool

var collatorStringCompare = func(getString func(Page) string, p1, p2 Page) int {
	return 0
}

// Sort stable sorts the pages given the receiver's sort order.
func (by pageBy) Sort(pages Pages) {
	ps := &pageSorter{
		pages: pages,
		by:    by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Stable(ps)
}

// A pageSorter implements the sort interface for Pages
type pageSorter struct {
	pages Pages
	by    pageBy
}

func (ps *pageSorter) Len() int      { return len(ps.pages) }
func (ps *pageSorter) Swap(i, j int) { ps.pages[i], ps.pages[j] = ps.pages[j], ps.pages[i] }

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (ps *pageSorter) Less(i, j int) bool { return ps.by(ps.pages[i], ps.pages[j]) }
