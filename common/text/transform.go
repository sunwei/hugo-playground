package text

import (
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"strings"
	"sync"
	"unicode"
)

// Puts adds a trailing \n none found.
func Puts(s string) string {
	if s == "" || s[len(s)-1] == '\n' {
		return s
	}
	return s + "\n"
}

// RemoveAccents removes all accents from b.
func RemoveAccents(b []byte) []byte {
	t := accentTransformerPool.Get().(transform.Transformer)
	b, _, _ = transform.Bytes(t, b)
	t.Reset()
	accentTransformerPool.Put(t)
	return b
}

var accentTransformerPool = &sync.Pool{
	New: func() any {
		return transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	},
}

// Chomp removes trailing newline characters from s.
func Chomp(s string) string {
	return strings.TrimRightFunc(s, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}

// VisitLinesAfter calls the given function for each line, including newlines, in the given string.
func VisitLinesAfter(s string, fn func(line string)) {
	high := strings.IndexRune(s, '\n')
	for high != -1 {
		fn(s[:high+1])
		s = s[high+1:]

		high = strings.IndexRune(s, '\n')
	}

	if s != "" {
		fn(s)
	}
}
