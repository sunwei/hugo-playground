package page

import (
	"html/template"
)

// Site represents a site in the build. This is currently a very narrow interface,
// but the actual implementation will be richer, see hugolib.SiteInfo.
type Site interface {
	// RegularPages Returns all the regular Pages in this Site.
	RegularPages() Pages

	// Pages Returns all Pages in this Site.
	Pages() Pages

	// Home A shortcut to the home page.
	Home() Page

	// Title Returns the configured title for this Site.
	Title() string

	// Current Returns Site currently rendering.
	Current() Site

	// BaseURL Returns the BaseURL for this Site.
	BaseURL() template.URL

	// Data Returns a map of all the data inside /data.
	Data() map[string]any
}

// Sites represents an ordered list of sites (languages).
type Sites []Site
