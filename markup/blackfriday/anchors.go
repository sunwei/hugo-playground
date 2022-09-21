package blackfriday

import "unicode"

// SanitizedAnchorName is how Blackfriday sanitizes anchor names.
// Implementation borrowed from https://github.com/russross/blackfriday/blob/a477dd1646916742841ed20379f941cfa6c5bb6f/block.go#L1464
// Note that Hugo removed its Blackfriday support in v0.100.0, but you can still use this strategy for
// auto ID generation.
func SanitizedAnchorName(text string) string {
	var anchorName []rune
	futureDash := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			if futureDash && len(anchorName) > 0 {
				anchorName = append(anchorName, '-')
			}
			futureDash = false
			anchorName = append(anchorName, unicode.ToLower(r))
		default:
			futureDash = true
		}
	}
	return string(anchorName)
}
