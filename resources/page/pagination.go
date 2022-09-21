// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package page

import (
	"errors"
	"fmt"
	"github.com/spf13/cast"
	"github.com/sunwei/hugo-playground/config"
	"math"
)

// PaginatorProvider provides two ways to create a page paginator.
type PaginatorProvider interface {
	Paginator(options ...any) (*Pager, error)
	Paginate(seq any, options ...any) (*Pager, error)
}

// Pager represents one of the elements in a paginator.
// The number, starting on 1, represents its place.
type Pager struct {
	number int
	*Paginator
}

func (p Pager) String() string {
	return fmt.Sprintf("Pager %d", p.number)
}

type paginatedElement interface {
	Len() int
}

type pagers []*Pager

type Paginator struct {
	paginatedElements []paginatedElement
	pagers
	paginationURLFactory
	total int
	size  int
}

type paginationURLFactory func(int) string

func ResolvePagerSize(cfg config.Provider, options ...any) (int, error) {
	if len(options) == 0 {
		return cfg.GetInt("paginate"), nil
	}

	if len(options) > 1 {
		return -1, errors.New("too many arguments, 'pager size' is currently the only option")
	}

	pas, err := cast.ToIntE(options[0])

	if err != nil || pas <= 0 {
		return -1, errors.New("'pager size' must be a positive integer")
	}

	return pas, nil
}

func Paginate(td TargetPathDescriptor, seq any, pagerSize int) (*Paginator, error) {
	if pagerSize <= 0 {
		return nil, errors.New("'paginate' configuration setting must be positive to paginate")
	}

	urlFactory := newPaginationURLFactory(td)

	var paginator *Paginator

	groups, err := ToPagesGroup(seq)
	if err != nil {
		return nil, err
	}
	if groups != nil {
		paginator, _ = newPaginatorFromPageGroups(groups, pagerSize, urlFactory)
	} else {
		pages, err := ToPages(seq)
		if err != nil {
			return nil, err
		}
		paginator, _ = newPaginatorFromPages(pages, pagerSize, urlFactory)
	}

	return paginator, nil
}

func newPaginationURLFactory(d TargetPathDescriptor) paginationURLFactory {
	return func(pageNumber int) string {
		pathDescriptor := d
		var rel string
		if pageNumber > 1 {
			rel = fmt.Sprintf("/%s/%d/", d.PathSpec.PaginatePath, pageNumber)
			pathDescriptor.Addends = rel
		}

		return CreateTargetPaths(pathDescriptor).RelPermalink(d.PathSpec)
	}
}

func newPaginatorFromPageGroups(pageGroups PagesGroup, size int, urlFactory paginationURLFactory) (*Paginator, error) {
	if size <= 0 {
		return nil, errors.New("Paginator size must be positive")
	}

	split := splitPageGroups(pageGroups, size)

	return newPaginator(split, pageGroups.Len(), size, urlFactory)
}

func splitPageGroups(pageGroups PagesGroup, size int) []paginatedElement {
	type keyPage struct {
		key  any
		page Page
	}

	var (
		split     []paginatedElement
		flattened []keyPage
	)

	for _, g := range pageGroups {
		for _, p := range g.Pages {
			flattened = append(flattened, keyPage{g.Key, p})
		}
	}

	numPages := len(flattened)

	for low, j := 0, numPages; low < j; low += size {
		high := int(math.Min(float64(low+size), float64(numPages)))

		var (
			pg         PagesGroup
			key        any
			groupIndex = -1
		)

		for k := low; k < high; k++ {
			kp := flattened[k]
			if key == nil || key != kp.key {
				key = kp.key
				pg = append(pg, PageGroup{Key: key})
				groupIndex++
			}
			pg[groupIndex].Pages = append(pg[groupIndex].Pages, kp.page)
		}
		split = append(split, pg)
	}

	return split
}

func newPaginator(elements []paginatedElement, total, size int, urlFactory paginationURLFactory) (*Paginator, error) {
	p := &Paginator{total: total, paginatedElements: elements, size: size, paginationURLFactory: urlFactory}

	var ps pagers

	if len(elements) > 0 {
		ps = make(pagers, len(elements))
		for i := range p.paginatedElements {
			ps[i] = &Pager{number: (i + 1), Paginator: p}
		}
	} else {
		ps = make(pagers, 1)
		ps[0] = &Pager{number: 1, Paginator: p}
	}

	p.pagers = ps

	return p, nil
}

func newPaginatorFromPages(pages Pages, size int, urlFactory paginationURLFactory) (*Paginator, error) {
	if size <= 0 {
		return nil, errors.New("Paginator size must be positive")
	}

	split := splitPages(pages, size)

	return newPaginator(split, len(pages), size, urlFactory)
}

func splitPages(pages Pages, size int) []paginatedElement {
	var split []paginatedElement
	for low, j := 0, len(pages); low < j; low += size {
		high := int(math.Min(float64(low+size), float64(len(pages))))
		split = append(split, pages[low:high])
	}

	return split
}

// Pagers returns a list of pagers that can be used to build a pagination menu.
func (p *Paginator) Pagers() pagers {
	return p.pagers
}
