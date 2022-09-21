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
	"sort"
	"strings"

	"github.com/sunwei/hugo-playground/config"
)

type LanguagesConfig struct {
	Languages Languages
}

func LoadLanguageSettings(cfg config.Provider, oldLangs Languages) (c LanguagesConfig, err error) {
	defaultLang := strings.ToLower(cfg.GetString("defaultContentLanguage"))
	if defaultLang == "" {
		defaultLang = "en"
		cfg.Set("defaultContentLanguage", defaultLang)
	}

	var languages map[string]any

	languagesFromConfig := cfg.GetParams("languages")
	disableLanguages := cfg.GetStringSlice("disableLanguages")

	if len(disableLanguages) == 0 {
		languages = languagesFromConfig
	} else {
		panic("there is no disabled language")
	}

	var languages2 Languages
	if len(languages) == 0 {
		languages2 = append(languages2, NewDefaultLanguage(cfg))
	} else {
		panic("no languages config params supported")
	}

	// oldLangs is nil

	// The defaultContentLanguage is something the user has to decide, but it needs
	// to match a language in the language definition list.
	langExists := false
	for _, lang := range languages2 {
		if lang.Lang == defaultLang {
			langExists = true
			break
		}
	}

	if !langExists {
		return c, fmt.Errorf("site config value %q for defaultContentLanguage does not match any language definition", defaultLang)
	}

	c.Languages = languages2

	sortedDefaultFirst := make(Languages, len(c.Languages))
	for i, v := range c.Languages {
		sortedDefaultFirst[i] = v
	}
	sort.Slice(sortedDefaultFirst, func(i, j int) bool {
		li, lj := sortedDefaultFirst[i], sortedDefaultFirst[j]
		if li.Lang == defaultLang {
			return true
		}

		if lj.Lang == defaultLang {
			return false
		}

		return i < j
	})

	cfg.Set("languagesSorted", c.Languages)                    // ["en"]
	cfg.Set("languagesSortedDefaultFirst", sortedDefaultFirst) // ["en"]
	cfg.Set("multilingual", len(languages2) > 1)               // false

	for _, language := range c.Languages {
		if language.initErr != nil {
			return c, language.initErr
		}
	}

	return c, nil
}
