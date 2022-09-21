package hugo

import "time"

// Dependency is a single dependency, which can be either a Hugo Module or a local theme.
type Dependency struct {
	// Returns the path to this module.
	// This will either be the module path, e.g. "github.com/gohugoio/myshortcodes",
	// or the path below your /theme folder, e.g. "mytheme".
	Path string

	// The module version.
	Version string

	// Whether this dependency is vendored.
	Vendor bool

	// Time version was created.
	Time time.Time

	// In the dependency tree, this is the first module that defines this module
	// as a dependency.
	Owner *Dependency

	// Replaced by this dependency.
	Replace *Dependency
}

// Info contains information about the current Hugo environment
type Info struct {
	CommitHash string
	BuildDate  string

	// The build environment.
	// Defaults are "production" (hugo) and "development" (hugo server).
	// This can also be set by the user.
	// It can be any string, but it will be all lower case.
	Environment string

	// version of go that the Hugo binary was built with
	GoVersion string

	deps []*Dependency
}
