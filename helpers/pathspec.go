// Copyright 2016-present The Hugo Authors. All rights reserved.
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

package helpers

import (
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugolib/filesystems"
	"github.com/sunwei/hugo-playground/hugolib/paths"
	"strings"
)

// PathSpec holds methods that decides how paths in URLs and files in Hugo should look like.
type PathSpec struct {
	*paths.Paths
	*filesystems.BaseFs

	// The file systems to use
	Fs *hugofs.Fs

	// The config provider to use
	Cfg config.Provider
}

// NewPathSpec creates a new PathSpec from the given filesystems and language.
func NewPathSpec(fs *hugofs.Fs, cfg config.Provider) (*PathSpec, error) {
	return NewPathSpecWithBaseBaseFsProvided(fs, cfg, nil)
}

// NewPathSpecWithBaseBaseFsProvided creats a new PathSpec from the given filesystems and language.
// If an existing BaseFs is provided, parts of that is reused.
func NewPathSpecWithBaseBaseFsProvided(fs *hugofs.Fs, cfg config.Provider, baseBaseFs *filesystems.BaseFs) (*PathSpec, error) {
	p, err := paths.New(fs, cfg)
	if err != nil {
		return nil, err
	}

	bfs, err := filesystems.NewBase(p)
	if err != nil {
		return nil, err
	}

	ps := &PathSpec{
		Paths:  p,
		BaseFs: bfs,
		Fs:     fs,
		Cfg:    cfg,
	}

	basePath := ps.BaseURL.Path()
	if basePath != "" && basePath != "/" {
		ps.BasePath = basePath
	}

	return ps, nil
}

// PermalinkForBaseURL creates a permalink from the given link and baseURL.
func (p *PathSpec) PermalinkForBaseURL(link, baseURL string) string {
	link = strings.TrimPrefix(link, "/")
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL + link
}
