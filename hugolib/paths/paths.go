// Copyright 2018 The Hugo Authors. All rights reserved.
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

package paths

import (
	"fmt"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/langs"
	"github.com/sunwei/hugo-playground/modules"
	"path/filepath"
	"strings"

	hpaths "github.com/sunwei/hugo-playground/common/paths"
)

var FilePathSeparator = string(filepath.Separator)

type Paths struct {
	Fs  *hugofs.Fs
	Cfg config.Provider

	BaseURL
	BaseURLString string

	// If the baseURL contains a base path, e.g. https://example.com/docs, then "/docs" will be the BasePath.
	BasePath string

	// Directories
	// TODO(bep) when we have trimmed down most of the dirs usage outside of this package, make
	// these into an interface.
	ThemesDir  string
	WorkingDir string

	// Directories to store Resource related artifacts.
	AbsResourcesDir string

	AbsPublishDir string

	// pagination path handling
	PaginatePath string

	Language              *langs.Language
	Languages             langs.Languages
	LanguagesDefaultFirst langs.Languages

	AllModules modules.Modules
}

func New(fs *hugofs.Fs, cfg config.Provider) (*Paths, error) {
	baseURLstr := cfg.GetString("baseURL")
	baseURL, err := newBaseURLFromString(baseURLstr)
	if err != nil {
		return nil, fmt.Errorf("failed to create baseURL from %q:: %w", baseURLstr, err)
	}

	workingDir := filepath.Clean(cfg.GetString("workingDir"))
	resourceDir := filepath.Clean(cfg.GetString("resourceDir"))
	publishDir := filepath.Clean(cfg.GetString("publishDir"))

	if publishDir == "" {
		return nil, fmt.Errorf("publishDir not set")
	}

	absPublishDir := hpaths.AbsPathify(workingDir, publishDir)
	if !strings.HasSuffix(absPublishDir, FilePathSeparator) {
		absPublishDir += FilePathSeparator
	}
	// If root, remove the second '/'
	if absPublishDir == "//" {
		absPublishDir = FilePathSeparator
	}
	absResourcesDir := hpaths.AbsPathify(workingDir, resourceDir)
	if !strings.HasSuffix(absResourcesDir, FilePathSeparator) {
		absResourcesDir += FilePathSeparator
	}
	if absResourcesDir == "//" {
		absResourcesDir = FilePathSeparator
	}

	var baseURLString = baseURL.String()

	p := &Paths{
		Fs:            fs,
		Cfg:           cfg,
		BaseURL:       baseURL,
		BaseURLString: baseURLString,

		ThemesDir:  cfg.GetString("themesDir"),
		WorkingDir: workingDir,

		AbsResourcesDir: absResourcesDir,
		AbsPublishDir:   absPublishDir,

		PaginatePath: cfg.GetString("paginatePath"),
	}

	if cfg.IsSet("allModules") {
		p.AllModules = cfg.Get("allModules").(modules.Modules)
	}

	return p, nil
}

func (p *Paths) Lang() string {
	return "en"
}

func (p *Paths) GetTargetLanguageBasePath() string {
	return p.GetLanguagePrefix()
}

func (p *Paths) GetLanguagePrefix() string {
	return "en"
}

// GetBasePath returns any path element in baseURL if needed.
func (p *Paths) GetBasePath(isRelativeURL bool) string {
	return p.BasePath
}

// RelPathify trims any WorkingDir prefix from the given filename. If
// the filename is not considered to be absolute, the path is just cleaned.
func (p *Paths) RelPathify(filename string) string {
	filename = filepath.Clean(filename)
	if !filepath.IsAbs(filename) {
		return filename
	}

	return strings.TrimPrefix(strings.TrimPrefix(filename, p.WorkingDir), FilePathSeparator)
}
