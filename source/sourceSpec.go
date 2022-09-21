package source

import (
	"fmt"
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/common/paths"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugofs/glob"
	"path/filepath"
	"strings"
)

// SourceSpec abstracts language-specific file creation.
// TODO(bep) rename to Spec
type SourceSpec struct {
	*helpers.PathSpec

	SourceFs afero.Fs

	Languages              map[string]any
	DefaultContentLanguage string
}

// NewSourceSpec initializes SourceSpec using languages the given filesystem and PathSpec.
func NewSourceSpec(ps *helpers.PathSpec, inclusionFilter *glob.FilenameFilter, fs afero.Fs) *SourceSpec {
	cfg := ps.Cfg
	defaultLang := cfg.GetString("defaultContentLanguage")
	languages := cfg.GetStringMap("languages")

	return &SourceSpec{
		PathSpec:               ps,
		SourceFs:               fs,
		Languages:              languages,
		DefaultContentLanguage: defaultLang,
	}
}

// IgnoreFile returns whether a given file should be ignored.
func (s *SourceSpec) IgnoreFile(filename string) bool {
	if filename == "" {
		if _, ok := s.SourceFs.(*afero.OsFs); ok {
			return true
		}
		return false
	}

	base := filepath.Base(filename)

	if len(base) > 0 {
		first := base[0]
		last := base[len(base)-1]
		if first == '.' ||
			first == '#' ||
			last == '~' {
			return true
		}
	}

	return false
}

func (sp *SourceSpec) NewFileInfo(fi hugofs.FileMetaInfo) (*FileInfo, error) {
	m := fi.Meta()

	filename := m.Filename
	relPath := m.Path

	if relPath == "" {
		return nil, fmt.Errorf("no Path provided by %v (%T)", m, m.Fs)
	}

	if filename == "" {
		return nil, fmt.Errorf("no Filename provided by %v (%T)", m, m.Fs)
	}

	relDir := filepath.Dir(relPath)
	if relDir == "." {
		relDir = ""
	}
	if !strings.HasSuffix(relDir, helpers.FilePathSeparator) {
		relDir = relDir + helpers.FilePathSeparator
	}

	dir, name := filepath.Split(relPath)
	if !strings.HasSuffix(dir, helpers.FilePathSeparator) {
		dir = dir + helpers.FilePathSeparator
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	baseName := paths.Filename(name)

	f := &FileInfo{
		sp:         sp,
		filename:   filename,
		fi:         fi,
		ext:        ext,
		dir:        dir,
		relDir:     relDir,  // Dir()
		relPath:    relPath, // Path()
		name:       name,
		baseName:   baseName, // BaseFileName()
		classifier: m.Classifier,
	}

	return f, nil
}
