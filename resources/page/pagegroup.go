package page

import "fmt"

// ToPagesGroup tries to convert seq into a PagesGroup.
func ToPagesGroup(seq any) (PagesGroup, error) {
	switch v := seq.(type) {
	case nil:
		return nil, nil
	case PagesGroup:
		return v, nil
	case []PageGroup:
		return PagesGroup(v), nil
	case []any:
		l := len(v)
		if l == 0 {
			break
		}
		switch v[0].(type) {
		case PageGroup:
			pagesGroup := make(PagesGroup, l)
			for i, ipg := range v {
				if pg, ok := ipg.(PageGroup); ok {
					pagesGroup[i] = pg
				} else {
					return nil, fmt.Errorf("unsupported type in paginate from slice, got %T instead of PageGroup", ipg)
				}
			}
			return pagesGroup, nil
		}
	}

	return nil, nil
}

// PagesGroup represents a list of page groups.
// This is what you get when doing page grouping in the templates.
type PagesGroup []PageGroup

// PageGroup represents a group of pages, grouped by the key.
// The key is typically a year or similar.
type PageGroup struct {
	// The key, typically a year or similar.
	Key any

	// The Pages in this group.
	Pages
}

// Len returns the number of pages in the page group.
func (psg PagesGroup) Len() int {
	l := 0
	for _, pg := range psg {
		l += len(pg.Pages)
	}
	return l
}
