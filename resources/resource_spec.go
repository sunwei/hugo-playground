package resources

import (
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/media"
	"github.com/sunwei/hugo-playground/output"
)

type Spec struct {
	*helpers.PathSpec

	MediaTypes    media.Types
	OutputFormats output.Formats
}

func NewSpec(
	s *helpers.PathSpec,
	outputFormats output.Formats,
	mimeTypes media.Types) (*Spec, error) {

	rs := &Spec{
		PathSpec:      s,
		MediaTypes:    mimeTypes,
		OutputFormats: outputFormats,
	}

	return rs, nil
}
