package helpers

import (
	"fmt"
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/common/hugio"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/hugofs"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// SymbolicWalk is like filepath.Walk, but it follows symbolic links.
func SymbolicWalk(fs afero.Fs, root string, walker hugofs.WalkFunc) error {
	if _, isOs := fs.(*afero.OsFs); isOs {
		// Mainly to track symlinks.
		fs = hugofs.NewBaseFileDecorator(fs)
	}

	w := hugofs.NewWalkway(hugofs.WalkwayConfig{
		Fs:     fs,
		Root:   root,
		WalkFn: walker,
	})

	return w.Walk()
}

// OpenFileForWriting opens or creates the given file. If the target directory
// does not exist, it gets created.
func OpenFileForWriting(fs afero.Fs, filename string) (afero.File, error) {
	filename = filepath.Clean(filename)
	// Create will truncate if file already exists.
	// os.Create will create any new files with mode 0666 (before umask).
	f, err := fs.Create(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if err = fs.MkdirAll(filepath.Dir(filename), 0777); err != nil { //  before umask
			return nil, err
		}
		f, err = fs.Create(filename)
	}

	return f, err
}

// GetCacheDir returns a cache dir from the given filesystem and config.
// The dir will be created if it does not exist.
func GetCacheDir(fs afero.Fs, cfg config.Provider) (string, error) {
	cacheDir := getCacheDir(cfg)
	if cacheDir != "" {
		exists, err := DirExists(cacheDir, fs)
		if err != nil {
			return "", err
		}
		if !exists {
			err := fs.MkdirAll(cacheDir, 0777) // Before umask
			if err != nil {
				return "", fmt.Errorf("failed to create cache dir: %w", err)
			}
		}
		return cacheDir, nil
	}

	// Fall back to a cache in /tmp.
	return GetTempDir("hugo_cache", fs), nil
}

func getCacheDir(cfg config.Provider) string {
	// Always use the cacheDir config if set.
	cacheDir := cfg.GetString("cacheDir")
	if len(cacheDir) > 1 {
		return addTrailingFileSeparator(cacheDir)
	}

	// See Issue #8714.
	// Turns out that Cloudflare also sets NETLIFY=true in its build environment,
	// but all of these 3 should not give any false positives.
	if os.Getenv("NETLIFY") == "true" && os.Getenv("PULL_REQUEST") != "" && os.Getenv("DEPLOY_PRIME_URL") != "" {
		// Netlify's cache behaviour is not documented, the currently best example
		// is this project:
		// https://github.com/philhawksworth/content-shards/blob/master/gulpfile.js
		return "/opt/build/cache/hugo_cache/"
	}

	// This will fall back to an hugo_cache folder in the tmp dir, which should work fine for most CI
	// providers. See this for a working CircleCI setup:
	// https://github.com/bep/hugo-sass-test/blob/6c3960a8f4b90e8938228688bc49bdcdd6b2d99e/.circleci/config.yml
	// If not, they can set the HUGO_CACHEDIR environment variable or cacheDir config key.
	return ""
}

func addTrailingFileSeparator(s string) string {
	if !strings.HasSuffix(s, FilePathSeparator) {
		s = s + FilePathSeparator
	}
	return s
}

// DirExists checks if a path exists and is a directory.
func DirExists(path string, fs afero.Fs) (bool, error) {
	return afero.DirExists(fs, path)
}

// GetTempDir returns a temporary directory with the given sub path.
func GetTempDir(subPath string, fs afero.Fs) string {
	return afero.GetTempDir(fs, subPath)
}

// MakePath takes a string with any characters and replace it
// so the string could be used in a path.
// It does so by creating a Unicode-sanitized string, with the spaces replaced,
// whilst preserving the original casing of the string.
// E.g. Social Media -> Social-Media
func (p *PathSpec) MakePath(s string) string {
	return p.UnicodeSanitize(s)
}

// UnicodeSanitize sanitizes string to be used in Hugo URL's, allowing only
// a predefined set of special Unicode characters.
// If RemovePathAccents configuration flag is enabled, Unicode accents
// are also removed.
// Hyphens in the original input are maintained.
// Spaces will be replaced with a single hyphen, and sequential replacement hyphens will be reduced to one.
func (p *PathSpec) UnicodeSanitize(s string) string {

	source := []rune(s)
	target := make([]rune, 0, len(source))
	var (
		prependHyphen bool
		wasHyphen     bool
	)

	for i, r := range source {
		isAllowed := r == '.' || r == '/' || r == '\\' || r == '_' || r == '#' || r == '+' || r == '~' || r == '-'
		isAllowed = isAllowed || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
		isAllowed = isAllowed || (r == '%' && i+2 < len(source) && ishex(source[i+1]) && ishex(source[i+2]))

		if isAllowed {
			// track explicit hyphen in input; no need to add a new hyphen if
			// we just saw one.
			wasHyphen = r == '-'

			if prependHyphen {
				// if currently have a hyphen, don't prepend an extra one
				if !wasHyphen {
					target = append(target, '-')
				}
				prependHyphen = false
			}
			target = append(target, r)
		} else if len(target) > 0 && !wasHyphen && unicode.IsSpace(r) {
			prependHyphen = true
		}
	}

	return string(target)
}

// From https://golang.org/src/net/url/url.go
func ishex(c rune) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

// AddTrailingSlash adds a trailing Unix styled slash (/) if not already
// there.
func AddTrailingSlash(path string) string {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

// WriteToDisk writes content to disk.
func WriteToDisk(inpath string, r io.Reader, fs afero.Fs) (err error) {
	return afero.WriteReader(fs, inpath, r)
}

// MakePathsSanitized applies MakePathSanitized on every item in the slice
func (p *PathSpec) MakePathsSanitized(paths []string) {
	for i, path := range paths {
		paths[i] = p.MakePathSanitized(path)
	}
}

// Should be good enough for Hugo.
var isFileRe = regexp.MustCompile(`.*\..{1,6}$`)

// GetDottedRelativePath expects a relative path starting after the content directory.
// It returns a relative path with dots ("..") navigating up the path structure.
func GetDottedRelativePath(inPath string) string {
	inPath = filepath.Clean(filepath.FromSlash(inPath))

	if inPath == "." {
		return "./"
	}

	if !isFileRe.MatchString(inPath) && !strings.HasSuffix(inPath, FilePathSeparator) {
		inPath += FilePathSeparator
	}

	if !strings.HasPrefix(inPath, FilePathSeparator) {
		inPath = FilePathSeparator + inPath
	}

	dir, _ := filepath.Split(inPath)

	sectionCount := strings.Count(dir, FilePathSeparator)

	if sectionCount == 0 || dir == FilePathSeparator {
		return "./"
	}

	var dottedPath string

	for i := 1; i < sectionCount; i++ {
		dottedPath += "../"
	}

	return dottedPath
}

// ToSlashTrimLeading is just a filepath.ToSlaas with an added / prefix trimmer.
func ToSlashTrimLeading(s string) string {
	return strings.TrimPrefix(filepath.ToSlash(s), "/")
}

// OpenFilesForWriting opens all the given filenames for writing.
func OpenFilesForWriting(fs afero.Fs, filenames ...string) (io.WriteCloser, error) {
	var writeClosers []io.WriteCloser
	for _, filename := range filenames {
		f, err := OpenFileForWriting(fs, filename)
		if err != nil {
			for _, wc := range writeClosers {
				wc.Close()
			}
			return nil, err
		}
		writeClosers = append(writeClosers, f)
	}

	return hugio.NewMultiWriteCloser(writeClosers...), nil
}
