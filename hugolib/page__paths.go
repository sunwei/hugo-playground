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

package hugolib

import (
	"github.com/sunwei/hugo-playground/resources/page"
)

func newPagePaths(
	s *Site,
	p page.Page,
	pm *pageMeta) (pagePaths, error) {
	targetPathDescriptor, err := createTargetPathDescriptor(s, p, pm)
	if err != nil {
		return pagePaths{}, err
	}

	outputFormats := pm.outputFormats()
	if len(outputFormats) == 0 {
		return pagePaths{}, nil
	}

	if pm.noRender() {
		outputFormats = outputFormats[:1]
	}

	pageOutputFormats := make(page.OutputFormats, len(outputFormats))
	targets := make(map[string]targetPathsHolder)

	for i, f := range outputFormats {
		desc := targetPathDescriptor
		desc.Type = f
		paths := page.CreateTargetPaths(desc)

		var relPermalink, permalink string

		// If a page is headless or bundled in another,
		// it will not get published on its own and it will have no links.
		// We also check the build options if it's set to not render or have
		// a link.
		if !pm.noLink() && !pm.bundled {
			relPermalink = paths.RelPermalink(s.PathSpec)
			permalink = paths.PermalinkForOutputFormat(s.PathSpec, f)
		}

		pageOutputFormats[i] = page.NewOutputFormat(relPermalink, permalink, len(outputFormats) == 1, f)

		// Use the main format for permalinks, usually HTML.
		permalinksIndex := 0
		if f.Permalinkable {
			// Unless it's permalinkable
			permalinksIndex = i
		}

		targets[f.Name] = targetPathsHolder{
			paths:        paths,
			OutputFormat: pageOutputFormats[permalinksIndex],
		}

	}

	var out page.OutputFormats
	if !pm.noLink() {
		out = pageOutputFormats
	}

	return pagePaths{
		outputFormats:        out,
		firstOutputFormat:    pageOutputFormats[0],
		targetPaths:          targets,
		targetPathDescriptor: targetPathDescriptor,
	}, nil
}

type pagePaths struct {
	outputFormats     page.OutputFormats
	firstOutputFormat page.OutputFormat

	targetPaths          map[string]targetPathsHolder
	targetPathDescriptor page.TargetPathDescriptor
}

func (l pagePaths) OutputFormats() page.OutputFormats {
	return l.outputFormats
}

func createTargetPathDescriptor(s *Site, p page.Page, pm *pageMeta) (page.TargetPathDescriptor, error) {
	var (
		dir      string
		baseName string
	)

	d := s.Deps

	if !p.File().IsZero() {
		dir = p.File().Dir()
		baseName = p.File().TranslationBaseName()
	}

	desc := page.TargetPathDescriptor{
		PathSpec:    d.PathSpec,
		Kind:        p.Kind(),
		Sections:    p.SectionsEntries(),
		ForcePrefix: false,
		Dir:         dir,
		URL:         pm.urlPaths.URL,
		BaseName:    baseName,
	}

	return desc, nil
}
