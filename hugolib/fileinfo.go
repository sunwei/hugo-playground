package hugolib

import (
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/source"
)

// TODO(bep) rename
func newFileInfo(sp *source.SourceSpec, fi hugofs.FileMetaInfo) (*fileInfo, error) {
	baseFi, err := sp.NewFileInfo(fi)
	if err != nil {
		return nil, err
	}

	f := &fileInfo{
		File: baseFi,
	}

	return f, nil
}

type fileInfo struct {
	source.File

	overriddenLang string
}
