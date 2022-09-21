package tplimpl

import (
	"strings"
)

func resolveTemplateType(name string) templateType {
	if isShortcode(name) {
		return templateShortcode
	}

	if strings.Contains(name, "partials/") {
		return templatePartial
	}

	return templateUndefined
}

func isShortcode(name string) bool {
	return strings.Contains(name, shortcodesPathPrefix)
}
