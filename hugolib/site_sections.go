package hugolib

import "github.com/sunwei/hugo-playground/resources/page"

// Sections returns the top level sections.
func (s *SiteInfo) Sections() page.Pages {
	return s.Home().Sections()
}

// Home is a shortcut to the home page, equivalent to .Site.GetPage "home".
func (s *SiteInfo) Home() page.Page {
	return s.s.home
}
