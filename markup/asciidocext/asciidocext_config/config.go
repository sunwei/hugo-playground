package asciidocext_config

// Config configures asciidoc.
type Config struct {
	Backend              string
	Extensions           []string
	Attributes           map[string]string
	NoHeaderOrFooter     bool
	SafeMode             string
	SectionNumbers       bool
	Verbose              bool
	Trace                bool
	FailureLevel         string
	WorkingFolderCurrent bool
	PreserveTOC          bool
}

var (
	// Default holds Hugo's default asciidoc configuration.
	Default = Config{
		Backend:              "html5",
		Extensions:           []string{},
		Attributes:           map[string]string{},
		NoHeaderOrFooter:     true,
		SafeMode:             "unsafe",
		SectionNumbers:       false,
		Verbose:              false,
		Trace:                false,
		FailureLevel:         "fatal",
		WorkingFolderCurrent: false,
		PreserveTOC:          false,
	}
)
