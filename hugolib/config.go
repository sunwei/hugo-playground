package hugolib

import (
	"fmt"
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/common/maps"
	cpaths "github.com/sunwei/hugo-playground/common/paths"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/langs"
	"github.com/sunwei/hugo-playground/modules"
	"path/filepath"
)

// ConfigSourceDescriptor describes where to find the config (e.g. config.toml etc.).
type ConfigSourceDescriptor struct {
	Fs afero.Fs

	// Path to the config file to use, e.g. /my/project/config.toml
	// Multiple config files supported, e.g. 'config.toml, abc.toml'
	// In our case, one config file is just right
	Filename string

	// The project's working dir. Is used to look for additional theme config.
	WorkingDir string
}

func (d ConfigSourceDescriptor) configFileDir() string {
	return d.WorkingDir
}

type configLoader struct {
	cfg config.Provider
	ConfigSourceDescriptor
}

func (l configLoader) loadConfig(configName string) (string, error) {
	baseDir := l.configFileDir()
	filename := filepath.Join(baseDir, configName)

	m, err := config.FromFileToMap(l.Fs, filename)
	if err != nil {
		return filename, err
	}

	// Set overwrites keys of the same name, recursively.
	l.cfg.Set("", m)

	return filename, nil
}

func (l configLoader) applyConfigDefaults() error {
	defaultSettings := maps.Params{
		"cleanDestinationDir":                  false,
		"watch":                                false,
		"resourceDir":                          "resources",
		"publishDir":                           "public",
		"themesDir":                            "themes",
		"buildDrafts":                          false,
		"buildFuture":                          false,
		"buildExpired":                         false,
		"environment":                          "production",
		"uglyURLs":                             false,
		"verbose":                              false,
		"ignoreCache":                          false,
		"canonifyURLs":                         false,
		"relativeURLs":                         false,
		"removePathAccents":                    false,
		"titleCaseStyle":                       "AP",
		"taxonomies":                           maps.Params{"tag": "tags", "category": "categories"},
		"permalinks":                           maps.Params{"a": "b"},
		"sitemap":                              maps.Params{"priority": -1, "filename": "sitemap.xml"},
		"disableLiveReload":                    false,
		"pluralizeListTitles":                  true,
		"forceSyncStatic":                      false,
		"footnoteAnchorPrefix":                 "",
		"footnoteReturnLinkContents":           "",
		"newContentEditor":                     "",
		"paginate":                             10,
		"paginatePath":                         "page",
		"summaryLength":                        70,
		"rssLimit":                             -1,
		"sectionPagesMenu":                     "",
		"disablePathToLower":                   false,
		"hasCJKLanguage":                       false,
		"enableEmoji":                          false,
		"defaultContentLanguage":               "en",
		"enableMissingTranslationPlaceholders": false,
		"enableGitInfo":                        false,
		"ignoreFiles":                          make([]string, 0),
		"disableAliases":                       false,
		"debug":                                false,
		"disableFastRender":                    false,
		"timeout":                              "30s",
		"enableInlineShortcodes":               false,
	}

	l.cfg.SetDefaults(defaultSettings)

	return nil
}

func (l configLoader) loadModulesConfig() (modules.Config, error) {
	modConfig, err := modules.DecodeConfig(l.cfg)
	if err != nil {
		return modules.Config{}, err
	}

	return modConfig, nil
}

func (l configLoader) collectModules(modConfig modules.Config, v1 config.Provider, hookBeforeFinalize func(m *modules.ModulesConfig) error) (modules.Modules, []string, error) {
	workingDir := l.WorkingDir
	themesDir := cpaths.AbsPathify(l.WorkingDir, v1.GetString("themesDir"))

	var configFilenames []string

	hook := func(m *modules.ModulesConfig) error {
		if hookBeforeFinalize != nil {
			return hookBeforeFinalize(m)
		}
		return nil
	}

	modulesClient := modules.NewClient(modules.ClientConfig{
		Fs:                 l.Fs,
		HookBeforeFinalize: hook,
		WorkingDir:         workingDir,
		ThemesDir:          themesDir,
		ModuleConfig:       modConfig,
	})

	v1.Set("modulesClient", modulesClient)

	moduleConfig, err := modulesClient.Collect()

	// Avoid recreating these later.
	v1.Set("allModules", moduleConfig.ActiveModules)

	if moduleConfig.GoModulesFilename != "" {
		// We want to watch this for changes and trigger rebuild on version
		// changes etc.
		configFilenames = append(configFilenames, moduleConfig.GoModulesFilename)
	}

	return moduleConfig.ActiveModules, configFilenames, err
}

// LoadConfig loads Hugo configuration into a new Viper and then adds
// a set of defaults.
func LoadConfig(d ConfigSourceDescriptor) (config.Provider, []string, error) {
	var configFiles []string
	l := configLoader{ConfigSourceDescriptor: d, cfg: config.New()}
	filename, err := l.loadConfig(d.Filename)
	if err == nil {
		configFiles = append(configFiles, filename)
	} else {
		return nil, nil, err
	}

	if err := l.applyConfigDefaults(); err != nil {
		return l.cfg, configFiles, err
	}

	modulesConfig, err := l.loadModulesConfig()
	if err != nil {
		fmt.Println("load modules config err...")
		return l.cfg, configFiles, err
	}
	fmt.Println("LoadConfig: modulesConfig >>>")
	fmt.Printf("%#v", modulesConfig)

	// Need to run these after the modules are loaded, but before
	// they are finalized.
	collectHook := func(m *modules.ModulesConfig) error {
		mods := m.ActiveModules

		//cfg.Set("languagesSorted", c.Languages)                    // ["en"]
		//cfg.Set("languagesSortedDefaultFirst", sortedDefaultFirst) // ["en"]
		//cfg.Set("multilingual", len(languages2) > 1)               // false
		if err := l.loadLanguageSettings(nil); err != nil {
			return err
		}

		// Apply default project mounts.
		// Default folder structure for hugo project
		if err := modules.ApplyProjectConfigDefaults(l.cfg, mods[0]); err != nil {
			return err
		}

		return nil
	}

	_, modulesConfigFiles, modulesCollectErr := l.collectModules(modulesConfig, l.cfg, collectHook)
	if modulesCollectErr != nil {
		return l.cfg, configFiles, modulesCollectErr
	}

	configFiles = append(configFiles, modulesConfigFiles...)

	return l.cfg, configFiles, err
}

func (l configLoader) loadLanguageSettings(oldLangs langs.Languages) error {
	_, err := langs.LoadLanguageSettings(l.cfg, oldLangs)
	return err
}
