package goldmark

import (
	"bytes"
	bp "github.com/sunwei/hugo-playground/bufferpool"
	"github.com/sunwei/hugo-playground/common/text"
	"github.com/sunwei/hugo-playground/markup/blackfriday"
	"github.com/sunwei/hugo-playground/markup/goldmark/goldmark_config"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
	"strconv"
	"unicode"
	"unicode/utf8"
)

func sanitizeAnchorNameString(s string, idType string) string {
	return string(sanitizeAnchorName([]byte(s), idType))
}

func sanitizeAnchorName(b []byte, idType string) []byte {
	return sanitizeAnchorNameWithHook(b, idType, nil)
}

func sanitizeAnchorNameWithHook(b []byte, idType string, hook func(buf *bytes.Buffer)) []byte {
	buf := bp.GetBuffer()

	if idType == goldmark_config.AutoHeadingIDTypeBlackfriday {
		// TODO(bep) make it more efficient.
		buf.WriteString(blackfriday.SanitizedAnchorName(string(b)))
	} else {
		asciiOnly := idType == goldmark_config.AutoHeadingIDTypeGitHubAscii

		if asciiOnly {
			// Normalize it to preserve accents if possible.
			b = text.RemoveAccents(b)
		}

		b = bytes.TrimSpace(b)

		for len(b) > 0 {
			r, size := utf8.DecodeRune(b)
			switch {
			case asciiOnly && size != 1:
			case r == '-' || r == ' ':
				buf.WriteRune('-')
			case isAlphaNumeric(r):
				buf.WriteRune(unicode.ToLower(r))
			default:
			}

			b = b[size:]
		}
	}

	if hook != nil {
		hook(buf)
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())

	bp.PutBuffer(buf)

	return result
}

func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func newIDFactory(idType string) *idFactory {
	return &idFactory{
		vals:   make(map[string]struct{}),
		idType: idType,
	}
}

type idFactory struct {
	idType string
	vals   map[string]struct{}
}

func (ids *idFactory) Generate(value []byte, kind ast.NodeKind) []byte {
	return sanitizeAnchorNameWithHook(value, ids.idType, func(buf *bytes.Buffer) {
		if buf.Len() == 0 {
			if kind == ast.KindHeading {
				buf.WriteString("heading")
			} else {
				buf.WriteString("id")
			}
		}

		if _, found := ids.vals[util.BytesToReadOnlyString(buf.Bytes())]; found {
			// Append a hypen and a number, starting with 1.
			buf.WriteRune('-')
			pos := buf.Len()
			for i := 1; ; i++ {
				buf.WriteString(strconv.Itoa(i))
				if _, found := ids.vals[util.BytesToReadOnlyString(buf.Bytes())]; !found {
					break
				}
				buf.Truncate(pos)
			}
		}

		ids.vals[buf.String()] = struct{}{}
	})
}

func (ids *idFactory) Put(value []byte) {
	ids.vals[util.BytesToReadOnlyString(value)] = struct{}{}
}
