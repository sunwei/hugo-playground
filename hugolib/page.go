package hugolib

import (
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/hugofs/files"
	"github.com/sunwei/hugo-playground/resources/page"
	"github.com/sunwei/hugo-playground/resources/resource"
	"github.com/sunwei/hugo-playground/source"
	"strings"
)

var (
	_ page.Page = (*pageState)(nil)
)

type pageState struct {
}

func (p *pageState) Err() resource.ResourceError {
	return nil
}

func (s *Site) sectionsFromFile(fi source.File) []string {
	dirname := fi.Dir()

	dirname = strings.Trim(dirname, helpers.FilePathSeparator)
	if dirname == "" {
		return nil
	}
	parts := strings.Split(dirname, helpers.FilePathSeparator)

	if fii, ok := fi.(*fileInfo); ok {
		if len(parts) > 0 && fii.FileInfo().Meta().Classifier == files.ContentClassLeaf {
			// my-section/mybundle/index.md => my-section
			return parts[:len(parts)-1]
		}
	}

	return parts
}
