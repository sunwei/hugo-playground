package pagemeta

// BuildConfig holds configuration options about how to handle a Page in Hugo's
// build process.
type BuildConfig struct {
	// Whether to add it to any of the page collections.
	// Note that the page can always be found with .Site.GetPage.
	// Valid values: never, always, local.
	// Setting it to 'local' means they will be available via the local
	// page collections, e.g. $section.Pages.
	// Note: before 0.57.2 this was a bool, so we accept those too.
	List string

	// Whether to render it.
	// Valid values: never, always, link.
	// The value link means it will not be rendered, but it will get a RelPermalink/Permalink.
	// Note that before 0.76.0 this was a bool, so we accept those too.
	Render string

	set bool // BuildCfg is non-zero if this is set to true.
}

type URLPath struct {
	URL       string
	Permalink string
	Slug      string
	Section   string
}

const (
	Never       = "never"
	Always      = "always"
	ListLocally = "local"
	Link        = "link"
)

var defaultBuildConfig = BuildConfig{
	List:   Always,
	Render: Always,
	set:    true,
}

func DecodeBuildConfig(m any) (BuildConfig, error) {
	b := defaultBuildConfig
	if m == nil {
		return b, nil
	}

	return BuildConfig{}, nil
}

func (b BuildConfig) IsZero() bool {
	return !b.set
}

// Disable sets all options to their off value.
func (b *BuildConfig) Disable() {
	b.List = Never
	b.Render = Never
	b.set = true
}
