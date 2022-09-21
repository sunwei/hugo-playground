package hugolib

import (
	"github.com/sunwei/hugo-playground/resources/page"
	"strings"
)

const (

	// The following are (currently) temporary nodes,
	// i.e. nodes we create just to render in isolation.
	kind404 = "404"

	// Temporary state.
	kindUnknown = "unknown"

	pageResourceType = "page"

	kindRobotsTXT = "robotsTXT"
)

var kindMap = map[string]string{
	strings.ToLower(kind404): kind404,
}

func getKind(s string) string {
	if pkind := page.GetKind(s); pkind != "" {
		return pkind
	}
	return kindMap[strings.ToLower(s)]
}
