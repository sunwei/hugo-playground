package tplimpl

import (
	"fmt"
	"github.com/spf13/afero"
)

type templateInfo struct {
	name     string
	template string
	isText   bool // HTML or plain text template.

	// Used to create some error context in error situations
	fs afero.Fs

	// The filename relative to the fs above.
	filename string

	// The real filename (if possible). Used for logging.
	realFilename string
}

func (t templateInfo) resolveType() templateType {
	return resolveTemplateType(t.name)
}

func (t templateInfo) errWithFileContext(what string, err error) error {
	return fmt.Errorf(what+": %w", err)
}

func (t templateInfo) IsZero() bool {
	return t.name == ""
}
