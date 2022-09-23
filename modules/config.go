package modules

import (
	"fmt"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/hugofs/files"
	"github.com/sunwei/hugo-playground/langs"
	"github.com/sunwei/hugo-playground/log"
	"strings"
)

type Import struct {
	Path                string // Module path
	pathProjectReplaced bool   // Set when Path is replaced in project config.
	IgnoreConfig        bool   // Ignore any config in config.toml (will still follow imports).
	IgnoreImports       bool   // Do not follow any configured imports.
	NoMounts            bool   // Do not mount any folder in this import.
	NoVendor            bool   // Never vendor this import (only allowed in main project).
	Disable             bool   // Turn off this module.
	Mounts              []Mount
}

type Mount struct {
	Source string // relative path in source repo, e.g. "scss"
	Target string // relative target path, e.g. "assets/bootstrap/scss"

	Lang string // any language code associated with this mount.
}

// Used as key to remove duplicates.
func (m Mount) key() string {
	return strings.Join([]string{"", m.Source, m.Target}, "/")
}

// Config holds a module config.
type Config struct {
	Mounts  []Mount
	Imports []Import

	// Meta info about this module (license information etc.).
	Params map[string]any
}

// DecodeConfig creates a modules Config from a given Hugo configuration.
func DecodeConfig(cfg config.Provider) (Config, error) {
	return decodeConfig(cfg)
}

func decodeConfig(cfg config.Provider) (Config, error) {
	c := DefaultModuleConfig

	if cfg == nil {
		// applyThemeConfig
		return c, nil
	}

	themeSet := cfg.IsSet("theme")

	if themeSet {
		// [mytheme]
		log.Process("decodeConfig", "set mytheme as Imports in DefaultModuleConfig, Config{}")
		imports := config.GetStringSlicePreserveString(cfg, "theme")
		for _, imp := range imports {
			c.Imports = append(c.Imports, Import{
				Path: imp,
			})
		}
	}

	return c, nil
}

var DefaultModuleConfig = Config{}

// ApplyProjectConfigDefaults applies default/missing module configuration for
// the main project.
func ApplyProjectConfigDefaults(cfg config.Provider, mod Module) error {
	moda := mod.(*moduleAdapter)

	// Map legacy directory config into the new module.
	languages := cfg.Get("languagesSortedDefaultFirst").(langs.Languages)
	isMultiHost := languages.IsMultihost()

	fmt.Println("add default configuration 1")
	fmt.Printf("%#v", moda.mounts)
	fmt.Printf("")
	// To bridge between old and new configuration format we need
	// a way to make sure all of the core components are configured on
	// the basic level.
	componentsConfigured := make(map[string]bool)
	for _, mnt := range moda.mounts {
		if !strings.HasPrefix(mnt.Target, files.JsConfigFolderMountPrefix) {
			componentsConfigured[mnt.Component()] = true
		}
	}

	type dirKeyComponent struct {
		key          string
		component    string
		multilingual bool
	}

	dirKeys := []dirKeyComponent{
		{"contentDir", files.ComponentFolderContent, true},
		{"dataDir", files.ComponentFolderData, false},
		{"layoutDir", files.ComponentFolderLayouts, false},
		{"i18nDir", files.ComponentFolderI18n, false},
		{"archetypeDir", files.ComponentFolderArchetypes, false},
		{"assetDir", files.ComponentFolderAssets, false},
		{"", files.ComponentFolderStatic, isMultiHost},
	}

	createMountsFor := func(d dirKeyComponent, cfg config.Provider) []Mount {
		var lang string
		if language, ok := cfg.(*langs.Language); ok {
			lang = language.Lang
		}

		// Static mounts are a little special.
		if d.component == files.ComponentFolderStatic {
			var mounts []Mount
			staticDirs := getStaticDirs(cfg)
			if len(staticDirs) > 0 {
				componentsConfigured[d.component] = true
			}

			for _, dir := range staticDirs {
				mounts = append(mounts, Mount{Lang: lang, Source: dir, Target: d.component})
			}

			return mounts

		}

		// We set `contentDir` only
		if cfg.IsSet(d.key) {
			source := cfg.GetString(d.key)
			componentsConfigured[d.component] = true

			return []Mount{{
				// No lang set for layouts etc.
				Source: source,
				Target: d.component,
			}}
		}

		return nil
	}

	createMounts := func(d dirKeyComponent) []Mount {
		var mounts []Mount
		if d.multilingual {
			if d.component == files.ComponentFolderContent {
				seen := make(map[string]bool)
				hasContentDir := false
				for _, language := range languages {
					if language.ContentDir != "" {
						hasContentDir = true
						break
					}
				}

				if hasContentDir {
					for _, language := range languages {
						contentDir := language.ContentDir
						if contentDir == "" {
							contentDir = files.ComponentFolderContent
						}
						if contentDir == "" || seen[contentDir] {
							continue
						}
						seen[contentDir] = true
						mounts = append(mounts, Mount{Lang: language.Lang, Source: contentDir, Target: d.component})
					}
				}

				componentsConfigured[d.component] = len(seen) > 0

			} else {
				for _, language := range languages {
					mounts = append(mounts, createMountsFor(d, language)...)
				}
			}
		} else {
			mounts = append(mounts, createMountsFor(d, cfg)...)
		}

		return mounts
	}

	var mounts []Mount
	for _, dirKey := range dirKeys {
		if componentsConfigured[dirKey.component] {
			continue
		}

		mounts = append(mounts, createMounts(dirKey)...)

	}

	// Add default configuration
	// Real mount created place, except 'contentDir'
	for _, dirKey := range dirKeys {
		if componentsConfigured[dirKey.component] {
			continue
		}
		mounts = append(mounts, Mount{Source: dirKey.component, Target: dirKey.component})
	}

	// Prepend the mounts from configuration.
	mounts = append(moda.mounts, mounts...)

	moda.mounts = mounts

	return nil
}

func (m Mount) Component() string {
	return strings.Split(m.Target, fileSeparator)[0]
}

func getStaticDirs(cfg config.Provider) []string {
	var staticDirs []string
	for i := -1; i <= 10; i++ {
		staticDirs = append(staticDirs, getStringOrStringSlice(cfg, "staticDir", i)...)
	}
	return staticDirs
}

func getStringOrStringSlice(cfg config.Provider, key string, id int) []string {
	if id >= 0 {
		key = fmt.Sprintf("%s%d", key, id)
	}

	return config.GetStringSlicePreserveString(cfg, key)
}
