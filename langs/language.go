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

package langs

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sunwei/hugo-playground/common/maps"
	"github.com/sunwei/hugo-playground/config"
	"golang.org/x/text/collate"
)

// These are the settings that should only be looked up in the global Viper
// config and not per language.
// This list may not be complete, but contains only settings that we know
// will be looked up in both.
// This isn't perfect, but it is ultimately the user who shoots him/herself in
// the foot.
// See the pathSpec.
var globalOnlySettings = map[string]bool{
	strings.ToLower("defaultContentLanguage"): true,
	strings.ToLower("multilingual"):           true,
	strings.ToLower("assetDir"):               true,
	strings.ToLower("resourceDir"):            true,
	strings.ToLower("build"):                  true,
}

// Language manages specific-language configuration.
type Language struct {
	Lang   string
	Weight int // for sort

	// If set per language, this tells Hugo that all content files without any
	// language indicator (e.g. my-page.en.md) is in this language.
	// This is usually a path relative to the working dir, but it can be an
	// absolute directory reference. It is what we get.
	// For internal use.
	ContentDir string

	// Global config.
	// For internal use.
	Cfg config.Provider

	// Language specific config.
	// For internal use.
	LocalCfg config.Provider

	// Composite config.
	// For internal use.
	config.Provider

	location *time.Location

	// Error during initialization. Will fail the buld.
	initErr error
}

// For internal use.
func (l *Language) String() string {
	return l.Lang
}

// NewLanguage creates a new language.
func NewLanguage(lang string, cfg config.Provider) *Language {
	localCfg := config.New()
	compositeConfig := config.NewCompositeConfig(cfg, localCfg)

	l := &Language{
		Lang:       lang,
		ContentDir: cfg.GetString("contentDir"),
		Cfg:        cfg,
		LocalCfg:   localCfg,
		Provider:   compositeConfig,
	}

	if err := l.loadLocation(cfg.GetString("timeZone")); err != nil {
		l.initErr = err
	}

	return l
}

// NewDefaultLanguage creates the default language for a config.Provider.
// If not otherwise specified the default is "en".
func NewDefaultLanguage(cfg config.Provider) *Language {
	defaultLang := cfg.GetString("defaultContentLanguage")

	if defaultLang == "" {
		defaultLang = "en"
	}

	return NewLanguage(defaultLang, cfg)
}

// Languages is a sortable list of languages.
type Languages []*Language

func (l Languages) Len() int { return len(l) }
func (l Languages) Less(i, j int) bool {
	wi, wj := l[i].Weight, l[j].Weight

	if wi == wj {
		return l[i].Lang < l[j].Lang
	}

	return wj == 0 || wi < wj
}

func (l Languages) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

// Params returns language-specific params merged with the global params.
func (l *Language) Params() maps.Params {
	params := make(map[string]any)
	return params
}

func (l Languages) AsSet() map[string]bool {
	m := make(map[string]bool)
	for _, lang := range l {
		m[lang.Lang] = true
	}

	return m
}

func (l Languages) AsOrdinalSet() map[string]int {
	m := make(map[string]int)
	for i, lang := range l {
		m[lang.Lang] = i
	}

	return m
}

// IsMultihost returns whether there are more than one language and at least one of
// the languages has baseURL specificed on the language level.
func (l Languages) IsMultihost() bool {
	if len(l) <= 1 {
		return false
	}

	for _, lang := range l {
		if lang.GetLocal("baseURL") != nil {
			return true
		}
	}
	return false
}

// SetParam sets a param with the given key and value.
// SetParam is case-insensitive.
// For internal use.
func (l *Language) SetParam(k string, v any) {
	panic("params cannot be set")
}

// GetLocal gets a configuration value set on language level. It will
// not fall back to any global value.
// It will return nil if a value with the given key cannot be found.
// For internal use.
func (l *Language) GetLocal(key string) any {
	if l == nil {
		panic("language not set")
	}
	key = strings.ToLower(key)
	if !globalOnlySettings[key] {
		return l.LocalCfg.Get(key)
	}
	return nil
}

// For internal use.
func (l *Language) Set(k string, v any) {
	k = strings.ToLower(k)
	if globalOnlySettings[k] {
		return
	}
	l.Provider.Set(k, v)
}

// Merge is currently not supported for Language.
// For internal use.
func (l *Language) Merge(key string, value any) {
	panic("Not supported")
}

// IsSet checks whether the key is set in the language or the related config store.
// For internal use.
func (l *Language) IsSet(key string) bool {
	key = strings.ToLower(key)
	if !globalOnlySettings[key] {
		return l.Provider.IsSet(key)
	}
	return l.Cfg.IsSet(key)
}

func GetLocation(l *Language) *time.Location {
	return l.location
}

func (l *Language) loadLocation(tzStr string) error {
	location, err := time.LoadLocation(tzStr)
	if err != nil {
		return fmt.Errorf("invalid timeZone for language %q: %w", l.Lang, err)
	}
	l.location = location

	return nil
}

type Collator struct {
	sync.Mutex
	c *collate.Collator
}

// CompareStrings compares a and b.
// It returns -1 if a < b, 1 if a > b and 0 if a == b.
// Note that the Collator is not thread safe, so you may want
// to aquire a lock on it before calling this method.
func (c *Collator) CompareStrings(a, b string) int {
	return c.c.CompareString(a, b)
}
