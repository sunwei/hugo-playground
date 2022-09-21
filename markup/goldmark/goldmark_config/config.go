package goldmark_config

// Config configures Goldmark.
type Config struct {
	Renderer   Renderer
	Parser     Parser
	Extensions Extensions
}

type Renderer struct {
	// Whether softline breaks should be rendered as '<br>'
	HardWraps bool

	// XHTML instead of HTML5.
	XHTML bool

	// Allow raw HTML etc.
	Unsafe bool
}

type Parser struct {
	// Enables custom heading ids and
	// auto generated heading ids.
	AutoHeadingID bool

	// The strategy to use when generating heading IDs.
	// Available options are "github", "github-ascii".
	// Default is "github", which will create GitHub-compatible anchor names.
	AutoHeadingIDType string

	// Enables custom attributes.
	Attribute ParserAttribute
}

type ParserAttribute struct {
	// Enables custom attributes for titles.
	Title bool
	// Enables custom attributeds for blocks.
	Block bool
}

type Extensions struct {
	Typographer    bool
	Footnote       bool
	DefinitionList bool

	// GitHub flavored markdown
	Table           bool
	Strikethrough   bool
	Linkify         bool
	LinkifyProtocol string
	TaskList        bool
}

const (
	AutoHeadingIDTypeGitHub      = "github"
	AutoHeadingIDTypeGitHubAscii = "github-ascii"
	AutoHeadingIDTypeBlackfriday = "blackfriday"
)

// Default DefaultConfig holds the default Goldmark configuration.
var Default = Config{
	Extensions: Extensions{
		Typographer:     true,
		Footnote:        true,
		DefinitionList:  true,
		Table:           true,
		Strikethrough:   true,
		Linkify:         true,
		LinkifyProtocol: "https",
		TaskList:        true,
	},
	Renderer: Renderer{
		Unsafe: false,
	},
	Parser: Parser{
		AutoHeadingID:     true,
		AutoHeadingIDType: AutoHeadingIDTypeGitHub,
		Attribute: ParserAttribute{
			Title: true,
			Block: false,
		},
	},
}
