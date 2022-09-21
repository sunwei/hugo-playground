package modules

import (
	"fmt"
	"github.com/spf13/afero"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var fileSeparator = string(os.PathSeparator)

const (
	goModFilename = "go.mod"
)

// ClientConfig configures the module Client.
type ClientConfig struct {
	// Fs to get the source
	Fs afero.Fs

	// If set, it will be run before we do any duplicate checks for modules
	// etc.
	// It must be set in our case, for default project structure
	HookBeforeFinalize func(m *ModulesConfig) error

	// Absolute path to the project dir.
	WorkingDir string

	// Absolute path to the project's themes dir.
	ThemesDir string

	// Read from config file and transferred
	ModuleConfig Config
}

func (c ClientConfig) shouldIgnoreVendor(path string) bool {
	return false
}

// Client contains most of the API provided by this package.
type Client struct {
	fs afero.Fs

	ccfg ClientConfig

	// The top level module config
	moduleConfig Config

	// Set when Go modules are initialized in the current repo, that is:
	// a go.mod file exists.
	GoModulesFilename string
}

// NewClient creates a new Client that can be used to manage the Hugo Components
// in a given workingDir.
// The Client will resolve the dependencies recursively, but needs the top
// level imports to start out.
func NewClient(cfg ClientConfig) *Client {
	fs := cfg.Fs
	n := filepath.Join(cfg.WorkingDir, goModFilename)
	goModEnabled, _ := afero.Exists(fs, n)
	var goModFilename string
	if goModEnabled {
		goModFilename = n
	}

	mcfg := cfg.ModuleConfig

	return &Client{
		fs:                fs,
		ccfg:              cfg,
		moduleConfig:      mcfg,
		GoModulesFilename: goModFilename,
	}
}

func (h *Client) Collect() (ModulesConfig, error) {
	mc, coll := h.collect(true)
	if coll.err != nil {
		return mc, coll.err
	}

	if err := (&mc).setActiveMods(); err != nil {
		return mc, err
	}

	if h.ccfg.HookBeforeFinalize != nil {
		if err := h.ccfg.HookBeforeFinalize(&mc); err != nil {
			return mc, err
		}
	}

	fmt.Println("collect done 222...")
	for _, m := range mc.ActiveModules {
		fmt.Println(len(m.Mounts()))
	}

	if err := (&mc).finalize(); err != nil {
		return mc, err
	}

	return mc, nil
}

func (h *Client) collect(tidy bool) (ModulesConfig, *collector) {
	c := &collector{
		Client: h,
	}

	c.collect()
	if c.err != nil {
		return ModulesConfig{}, c
	}

	return ModulesConfig{
		AllModules:        c.modules,
		GoModulesFilename: c.GoModulesFilename,
	}, c
}

func (c *Client) createThemeDirname(modulePath string, isProjectMod bool) (string, error) {
	invalid := fmt.Errorf("invalid module path %q; must be relative to themesDir when defined outside of the project", modulePath)

	modulePath = filepath.Clean(modulePath)
	if filepath.IsAbs(modulePath) {
		if isProjectMod {
			return modulePath, nil
		}
		return "", invalid
	}

	moduleDir := filepath.Join(c.ccfg.ThemesDir, modulePath)
	if !isProjectMod && !strings.HasPrefix(moduleDir, c.ccfg.ThemesDir) {
		return "", invalid
	}
	return moduleDir, nil
}

type collected struct {
	// Pick the first and prevent circular loops.
	seen map[string]bool

	// Set if a Go modules enabled project.
	gomods goModules

	// Ordered list of collected modules, including Go Modules and theme
	// components stored below /themes.
	modules Modules
}

type goModule struct {
	Path     string         // module path
	Version  string         // module version
	Versions []string       // available module versions (with -versions)
	Replace  *goModule      // replaced by this module
	Time     *time.Time     // time version was created
	Update   *goModule      // available update, if any (with -u)
	Main     bool           // is this the main module?
	Indirect bool           // is this module only an indirect dependency of main module?
	Dir      string         // directory holding files for this module, if any
	GoMod    string         // path to go.mod file for this module, if any
	Error    *goModuleError // error loading module
}

type goModuleError struct {
	Err string // the error itself
}

type goModules []*goModule

func (modules goModules) GetMain() *goModule {
	for _, m := range modules {
		if m.Main {
			return m
		}
	}

	return nil
}

func (modules goModules) GetByPath(p string) *goModule {
	if modules == nil {
		return nil
	}

	for _, m := range modules {
		if strings.EqualFold(p, m.Path) {
			return m
		}
	}

	return nil
}
