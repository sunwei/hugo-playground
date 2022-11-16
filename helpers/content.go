package helpers

import (
	"bytes"
	"github.com/sunwei/hugo-playground/markup"
	"html/template"
	"strings"
	"unicode"
	"unicode/utf8"
)

// NewContentSpec returns a ContentSpec initialized
// with the appropriate fields from the given config.Provider.
func NewContentSpec() (*ContentSpec, error) {
	spec := &ContentSpec{
		summaryLength: 1000,
	}

	// markdown
	converterProvider, err := markup.NewConverterProvider()
	if err != nil {
		return nil, err
	}

	spec.Converters = converterProvider

	return spec, nil
}

// ContentSpec provides functionality to render markdown content.
type ContentSpec struct {
	Converters markup.ConverterProvider

	// SummaryLength is the length of the summary that Hugo extracts from a content.
	summaryLength int
}

func (c *ContentSpec) ResolveMarkup(in string) string {
	in = strings.ToLower(in)
	switch in {
	case "md", "markdown", "mdown":
		return "markdown"
	case "html", "htm":
		return "html"
	default:
		if conv := c.Converters.Get(in); conv != nil {
			return conv.Name()
		}
	}
	return ""
}

// ExtractTOC extracts Table of Contents from content.
func ExtractTOC(content []byte) (newcontent []byte, toc []byte) {
	if !bytes.Contains(content, []byte("<nav>")) {
		return content, nil
	}
	origContent := make([]byte, len(content))
	copy(origContent, content)
	first := []byte(`<nav>
<ul>`)

	last := []byte(`</ul>
</nav>`)

	replacement := []byte(`<nav id="TableOfContents">
<ul>`)

	startOfTOC := bytes.Index(content, first)

	peekEnd := len(content)
	if peekEnd > 70+startOfTOC {
		peekEnd = 70 + startOfTOC
	}

	if startOfTOC < 0 {
		return stripEmptyNav(content), toc
	}
	// Need to peek ahead to see if this nav element is actually the right one.
	correctNav := bytes.Index(content[startOfTOC:peekEnd], []byte(`<li><a href="#`))
	if correctNav < 0 { // no match found
		return content, toc
	}
	lengthOfTOC := bytes.Index(content[startOfTOC:], last) + len(last)
	endOfTOC := startOfTOC + lengthOfTOC

	newcontent = append(content[:startOfTOC], content[endOfTOC:]...)
	toc = append(replacement, origContent[startOfTOC+len(first):endOfTOC]...)
	return
}

// stripEmptyNav strips out empty <nav> tags from content.
func stripEmptyNav(in []byte) []byte {
	return bytes.Replace(in, []byte("<nav>\n</nav>\n\n"), []byte(``), -1)
}

// BytesToHTML converts bytes to type template.HTML.
func BytesToHTML(b []byte) template.HTML {
	return template.HTML(string(b))
}

var (
	openingPTag        = []byte("<p>")
	closingPTag        = []byte("</p>")
	paragraphIndicator = []byte("<p")
	closingIndicator   = []byte("</")
)

// TrimShortHTML removes the <p>/</p> tags from HTML input in the situation
// where said tags are the only <p> tags in the input and enclose the content
// of the input (whitespace excluded).
func (c *ContentSpec) TrimShortHTML(input []byte) []byte {
	firstOpeningP := bytes.Index(input, paragraphIndicator)
	lastOpeningP := bytes.LastIndex(input, paragraphIndicator)

	lastClosingP := bytes.LastIndex(input, closingPTag)
	lastClosing := bytes.LastIndex(input, closingIndicator)

	if firstOpeningP == lastOpeningP && lastClosingP == lastClosing {
		input = bytes.TrimSpace(input)
		input = bytes.TrimPrefix(input, openingPTag)
		input = bytes.TrimSuffix(input, closingPTag)
		input = bytes.TrimSpace(input)
	}
	return input
}

// TruncateWordsByRune truncates words by runes.
func (c *ContentSpec) TruncateWordsByRune(in []string) (string, bool) {
	words := make([]string, len(in))
	copy(words, in)

	count := 0
	for index, word := range words {
		if count >= c.summaryLength {
			return strings.Join(words[:index], " "), true
		}
		runeCount := utf8.RuneCountInString(word)
		if len(word) == runeCount {
			count++
		} else if count+runeCount < c.summaryLength {
			count += runeCount
		} else {
			for ri := range word {
				if count >= c.summaryLength {
					truncatedWords := append(words[:index], word[:ri])
					return strings.Join(truncatedWords, " "), true
				}
				count++
			}
		}
	}

	return strings.Join(words, " "), false
}

// TruncateWordsToWholeSentence takes content and truncates to whole sentence
// limited by max number of words. It also returns whether it is truncated.
func (c *ContentSpec) TruncateWordsToWholeSentence(s string) (string, bool) {
	var (
		wordCount     = 0
		lastWordIndex = -1
	)

	for i, r := range s {
		if unicode.IsSpace(r) {
			wordCount++
			lastWordIndex = i

			if wordCount >= c.summaryLength {
				break
			}

		}
	}

	if lastWordIndex == -1 {
		return s, false
	}

	endIndex := -1

	for j, r := range s[lastWordIndex:] {
		if isEndOfSentence(r) {
			endIndex = j + lastWordIndex + utf8.RuneLen(r)
			break
		}
	}

	if endIndex == -1 {
		return s, false
	}

	return strings.TrimSpace(s[:endIndex]), endIndex < len(s)
}

func isEndOfSentence(r rune) bool {
	return r == '.' || r == '?' || r == '!' || r == '"' || r == '\n'
}

// TotalWords counts instance of one or more consecutive white space
// characters, as defined by unicode.IsSpace, in s.
// This is a cheaper way of word counting than the obvious len(strings.Fields(s)).
func TotalWords(s string) int {
	n := 0
	inWord := false
	for _, r := range s {
		wasInWord := inWord
		inWord = !unicode.IsSpace(r)
		if inWord && !wasInWord {
			n++
		}
	}
	return n
}
