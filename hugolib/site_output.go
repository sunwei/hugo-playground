package hugolib

import (
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/resources/page"
)

func createDefaultOutputFormats(allFormats output.Formats) map[string]output.Formats {
	htmlOut, _ := allFormats.GetByName(output.HTMLFormat.Name)

	defaultListTypes := output.Formats{htmlOut}

	m := map[string]output.Formats{
		page.KindPage:     {htmlOut},
		page.KindHome:     defaultListTypes,
		page.KindSection:  defaultListTypes,
		page.KindTerm:     defaultListTypes,
		page.KindTaxonomy: defaultListTypes,
		kind404:           {htmlOut},
	}

	return m
}

func createSiteOutputFormats(allFormats output.Formats, outputs map[string]any, rssDisabled bool) (map[string]output.Formats, error) {
	defaultOutputFormats := createDefaultOutputFormats(allFormats)
	return defaultOutputFormats, nil
}
