package internal

import "github.com/sunwei/hugo-playground/helpers"

// ResourceTransformationKey are provided by the different transformation implementations.
// It identifies the transformation (name) and its configuration (elements).
// We combine this in a chain with the rest of the transformations
// with the target filename and a content hash of the origin to use as cache key.
type ResourceTransformationKey struct {
	Name     string
	elements []any
}

// Value returns the Key as a string.
// Do not change this without good reasons.
func (k ResourceTransformationKey) Value() string {
	if len(k.elements) == 0 {
		return k.Name
	}

	return k.Name + "_" + helpers.HashString(k.elements...)
}
