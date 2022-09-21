package helpers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/jdkato/prose/transform"
	"github.com/mitchellh/hashstructure"
	bp "github.com/sunwei/hugo-playground/bufferpool"
	"github.com/sunwei/hugo-playground/common/loggers"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// FilePathSeparator as defined by os.Separator.
const FilePathSeparator = string(filepath.Separator)

// GetTitleFunc returns a func that can be used to transform a string to
// title case.
//
// The supported styles are
//
// - "Go" (strings.Title)
// - "AP" (see https://www.apstylebook.com/)
// - "Chicago" (see http://www.chicagomanualofstyle.org/home.html)
//
// If an unknown or empty style is provided, AP style is what you get.
func GetTitleFunc(style string) func(s string) string {
	switch strings.ToLower(style) {
	case "go":
		return strings.Title
	default:
		tc := transform.NewTitleConverter(transform.APStyle)
		return tc.Title
	}
}

// MD5String takes a string and returns its MD5 hash.
func MD5String(f string) string {
	h := md5.New()
	h.Write([]byte(f))
	return hex.EncodeToString(h.Sum([]byte{}))
}

// Deprecated informs about a deprecation, but only once for a given set of arguments' values.
// If the err flag is enabled, it logs as an ERROR (will exit with -1) and the text will
// point at the next Hugo release.
// The idea is two remove an item in two Hugo releases to give users and theme authors
// plenty of time to fix their templates.
func Deprecated(item, alternative string, err bool) {
	if err {
		fmt.Printf("%s is deprecated and will be removed in Hugo %s. %s", item, "0.1", alternative)
	} else {
		fmt.Printf("%s is deprecated and will be removed in a future release. %s%s", item, alternative, "warnPanicMessage")
	}
}

// ReaderToBytes takes an io.Reader argument, reads from it
// and returns bytes.
func ReaderToBytes(lines io.Reader) []byte {
	if lines == nil {
		return []byte{}
	}
	b := bp.GetBuffer()
	defer bp.PutBuffer(b)

	b.ReadFrom(lines)

	bc := make([]byte, b.Len())
	copy(bc, b.Bytes())
	return bc
}

// UniqueStringsReuse returns a slice with any duplicates removed.
// It will modify the input slice.
func UniqueStringsReuse(s []string) []string {
	result := s[:0]
	for i, val := range s {
		var seen bool

		for j := 0; j < i; j++ {
			if s[j] == val {
				seen = true
				break
			}
		}

		if !seen {
			result = append(result, val)
		}
	}
	return result
}

// UniqueStringsSorted UniqueStringsReuse returns a sorted slice with any duplicates removed.
// It will modify the input slice.
func UniqueStringsSorted(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	ss := sort.StringSlice(s)
	ss.Sort()
	i := 0
	for j := 1; j < len(s); j++ {
		if !ss.Less(i, j) {
			continue
		}
		i++
		s[i] = s[j]
	}

	return s[:i+1]
}

// FirstUpper returns a string with the first character as upper case.
func FirstUpper(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// NewDistinctErrorLogger creates a new DistinctLogger that logs ERRORs
func NewDistinctErrorLogger() loggers.Logger {
	return &DistinctLogger{m: make(map[string]bool), Logger: loggers.NewErrorLogger()}
}

// DistinctLogger ignores duplicate log statements.
type DistinctLogger struct {
	loggers.Logger
	sync.RWMutex
	m map[string]bool
}

// HashString returns a hash from the given elements.
// It will panic if the hash cannot be calculated.
func HashString(elements ...any) string {
	var o any
	if len(elements) == 1 {
		o = elements[0]
	} else {
		o = elements
	}

	hash, err := hashstructure.Hash(o, nil)
	if err != nil {
		panic(err)
	}
	return strconv.FormatUint(hash, 10)
}

// MD5FromFileFast creates a MD5 hash from the given file. It only reads parts of
// the file for speed, so don't use it if the files are very subtly different.
// It will not close the file.
func MD5FromFileFast(r io.ReadSeeker) (string, error) {
	const (
		// Do not change once set in stone!
		maxChunks = 8
		peekSize  = 64
		seek      = 2048
	)

	h := md5.New()
	buff := make([]byte, peekSize)

	for i := 0; i < maxChunks; i++ {
		if i > 0 {
			_, err := r.Seek(seek, 0)
			if err != nil {
				if err == io.EOF {
					break
				}
				return "", err
			}
		}

		_, err := io.ReadAtLeast(r, buff, peekSize)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				h.Write(buff)
				break
			}
			return "", err
		}
		h.Write(buff)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
