package modules

import (
	"github.com/sunwei/hugo-playground/config"
	"time"
)

type Module interface {

	// Config The decoded module config and mounts.
	Config() Config

	// Dir Directory holding files for this module.
	Dir() string

	// IsGoMod Returns whether this is a Go Module.
	IsGoMod() bool

	// Path Returns the path to this module.
	// This will either be the module path, e.g. "github.com/gohugoio/myshortcodes",
	// or the path below your /theme folder, e.g. "mytheme".
	Path() string

	// Owner In the dependency tree, this is the first module that defines this module
	// as a dependency.
	Owner() Module

	// Mounts Any directory remappings.
	Mounts() []Mount
}

type Modules []Module

type moduleAdapter struct {
	path       string
	dir        string
	version    string
	vendor     bool
	disabled   bool
	projectMod bool
	owner      Module

	mounts []Mount

	configFilenames []string
	cfg             config.Provider
	config          Config

	// Set if a Go module.
	gomod *goModule
}

func (m *moduleAdapter) Cfg() config.Provider {
	return m.cfg
}

func (m *moduleAdapter) Config() Config {
	return m.config
}

func (m *moduleAdapter) ConfigFilenames() []string {
	return m.configFilenames
}

func (m *moduleAdapter) Dir() string {
	// This may point to the _vendor dir.
	if !m.IsGoMod() || m.dir != "" {
		return m.dir
	}
	return m.gomod.Dir
}

func (m *moduleAdapter) Disabled() bool {
	return m.disabled
}

func (m *moduleAdapter) IsGoMod() bool {
	return m.gomod != nil
}

func (m *moduleAdapter) Mounts() []Mount {
	return m.mounts
}

func (m *moduleAdapter) Owner() Module {
	return m.owner
}

func (m *moduleAdapter) Path() string {
	if !m.IsGoMod() || m.path != "" {
		return m.path
	}
	return m.gomod.Path
}

func (m *moduleAdapter) Replace() Module {
	if m.IsGoMod() && !m.Vendor() && m.gomod.Replace != nil {
		return &moduleAdapter{
			gomod: m.gomod.Replace,
			owner: m.owner,
		}
	}
	return nil
}

func (m *moduleAdapter) Vendor() bool {
	return m.vendor
}

func (m *moduleAdapter) Version() string {
	if !m.IsGoMod() || m.version != "" {
		return m.version
	}
	return m.gomod.Version
}

func (m *moduleAdapter) Time() time.Time {
	if !m.IsGoMod() || m.gomod.Time == nil {
		return time.Time{}
	}

	return *m.gomod.Time

}
