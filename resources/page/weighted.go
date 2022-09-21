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

// WeightedPages is a list of Pages with their corresponding (and relative) weight
// [{Weight: 30, Page: *1}, {Weight: 40, Page: *2}]
type WeightedPages []WeightedPage

// Page will return the Page (of Kind taxonomyList) that represents this set
// of pages. This method will panic if p is empty, as that should never happen.
func (p WeightedPages) Page() Page {
	if len(p) == 0 {
		panic("WeightedPages is empty")
	}

	first := p[0]

	// TODO(bep) fix tests
	if first.owner == nil {
		return nil
	}

	return first.owner
}

// A WeightedPage is a Page with a weight.
type WeightedPage struct {
	Weight int
	Page

	// Reference to the owning Page. This avoids having to do
	// manual .Site.GetPage lookups. It is implemented in this roundabout way
	// because we cannot add additional state to the WeightedPages slice
	// without breaking lots of templates in the wild.
	owner Page
}

// Pages returns the Pages in this weighted page set.
func (wp WeightedPages) Pages() Pages {
	pages := make(Pages, len(wp))
	for i := range wp {
		pages[i] = wp[i].Page
	}
	return pages
}
