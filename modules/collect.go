package modules

import (
	"fmt"
	"github.com/rogpeppe/go-internal/module"
	"github.com/spf13/afero"
	"strings"
)

type ModulesConfig struct {
	// All modules, including any disabled.
	AllModules Modules

	// All active modules.
	ActiveModules Modules

	// Set if this is a Go modules enabled project.
	GoModulesFilename string
}

func (m *ModulesConfig) setActiveMods() error {
	var activeMods Modules
	for _, mod := range m.AllModules {
		activeMods = append(activeMods, mod)
	}

	m.ActiveModules = activeMods

	return nil
}

func (m *ModulesConfig) finalize() error {
	for _, mod := range m.AllModules {
		m := mod.(*moduleAdapter)
		m.mounts = filterUnwantedMounts(m.mounts)
	}
	return nil
}

func filterUnwantedMounts(mounts []Mount) []Mount {
	// Remove duplicates
	seen := make(map[string]bool)
	tmp := mounts[:0]
	for _, m := range mounts {
		if !seen[m.key()] {
			tmp = append(tmp, m)
		}
		seen[m.key()] = true
	}
	return tmp
}

// Collects and creates a module tree.
type collector struct {
	*Client

	// Store away any non-fatal error and return at the end.
	err error

	// Set to disable any Tidy operation in the end.
	skipTidy bool

	*collected
}

func (c *collector) collect() {
	c.collected = &collected{
		seen:   make(map[string]bool),
		gomods: goModules{},
	}

	// c.gomods is [], GetMain() returns nil
	projectMod := createProjectModule(c.gomods.GetMain(), c.ccfg.WorkingDir, c.moduleConfig)

	// module structure, [project, others...]
	if err := c.addAndRecurse(projectMod, false); err != nil {
		c.err = err
		return
	}

	// Add the project mod on top.
	c.modules = append(Modules{projectMod}, c.modules...)
}

// addAndRecurse Project Imports -> Import imports
func (c *collector) addAndRecurse(owner *moduleAdapter, disabled bool) error {
	moduleConfig := owner.Config()

	for _, moduleImport := range moduleConfig.Imports {
		disabled := disabled || moduleImport.Disable

		if !c.isSeen(moduleImport.Path) {
			tc, err := c.add(owner, moduleImport, disabled)
			fmt.Println("not seen...")
			fmt.Printf("%#v", tc)
			fmt.Println("_")
			if err != nil {
				return err
			}
			if tc == nil || moduleImport.IgnoreImports {
				continue
			}
			// tc is mytheme with no config file
			if err := c.addAndRecurse(tc, disabled); err != nil {
				return err
			}
		}
	}
	return nil
}

// add owner is project module
func (c *collector) add(owner *moduleAdapter, moduleImport Import, disabled bool) (*moduleAdapter, error) {
	var (
		mod       *goModule
		moduleDir string
		version   string
		vendored  bool
	)

	modulePath := moduleImport.Path
	var realOwner Module = owner

	if moduleDir == "" {
		mod = c.gomods.GetByPath(modulePath)
		if mod != nil {
			moduleDir = mod.Dir
		}

		if moduleDir == "" {
			// Fall back to project/themes/<mymodule>
			if moduleDir == "" {
				var err error
				moduleDir, err = c.createThemeDirname(modulePath, owner.projectMod || moduleImport.pathProjectReplaced)
				if err != nil {
					c.err = err
					return nil, nil
				}
				if found, _ := afero.Exists(c.fs, moduleDir); !found {
					c.err = fmt.Errorf(`module %q not found; either add it as a Hugo Module or store it in %q`, modulePath, c.ccfg.ThemesDir)
					return nil, nil
				}
			}
		}
	}

	if found, _ := afero.Exists(c.fs, moduleDir); !found {
		c.err = fmt.Errorf("%q not found", moduleDir)
		return nil, nil
	}

	if !strings.HasSuffix(moduleDir, fileSeparator) {
		moduleDir += fileSeparator
	}

	ma := &moduleAdapter{
		dir:      moduleDir,
		vendor:   vendored,
		disabled: disabled,
		gomod:    mod,
		version:  version,
		// This may be the owner of the _vendor dir
		owner: realOwner,
	}

	if mod == nil {
		ma.path = modulePath
	}

	if !moduleImport.IgnoreConfig {
		if err := c.applyThemeConfig(ma); err != nil {
			return nil, err
		}
	}

	// remove applyMounts for mytheme, because there is no component folder in our example

	c.modules = append(c.modules, ma)
	return ma, nil
}

func (c *collector) applyThemeConfig(tc *moduleAdapter) error {
	// tc.cfg is nil
	// mytheme has no config file
	configT, err := decodeConfig(tc.cfg)
	if err != nil {
		return err
	}

	tc.config = configT

	return nil
}

func (c *collector) isSeen(path string) bool {
	key := pathKey(path)
	if c.seen[key] {
		return true
	}
	c.seen[key] = true
	return false
}

// In the first iteration of Hugo Modules, we do not support multiple
// major versions running at the same time, so we pick the first (upper most).
// We will investigate namespaces in future versions.
// TODO(bep) add a warning when the above happens.
func pathKey(p string) string {
	prefix, _, _ := module.SplitPathVersion(p)

	return strings.ToLower(prefix)
}

func createProjectModule(gomod *goModule, workingDir string, conf Config) *moduleAdapter {
	// Create a pseudo module for the main project.
	var path string
	if gomod == nil {
		path = "project"
	}

	return &moduleAdapter{
		path:       path,
		dir:        workingDir,
		gomod:      gomod,
		projectMod: true,
		config:     conf,
	}
}
