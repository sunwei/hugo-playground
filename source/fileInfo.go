package source

import (
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugofs/files"
	"path/filepath"
	"strings"
	"sync"
)

// File represents a source file.
// This is a temporary construct until we resolve page.Page conflicts.
// TODO(bep) remove this construct once we have resolved page deprecations
type File interface {
	fileOverlap
	FileWithoutOverlap
}

// Temporary to solve duplicate/deprecated names in page.Page
type fileOverlap interface {
	// Path gets the relative path including file name and extension.
	// The directory is relative to the content root.
	Path() string

	// Section is first directory below the content root.
	// For page bundles in root, the Section will be empty.
	Section() string

	IsZero() bool
}

type FileWithoutOverlap interface {

	// Filename gets the full path and filename to the file.
	Filename() string

	// Dir gets the name of the directory that contains this file.
	// The directory is relative to the content root.
	Dir() string

	// Extension is an alias to Ext().
	// Deprecated: Use Ext instead.
	Extension() string

	// Ext gets the file extension, i.e "myblogpost.md" will return "md".
	Ext() string

	// LogicalName is filename and extension of the file.
	LogicalName() string

	// BaseFileName is a filename without extension.
	BaseFileName() string

	// TranslationBaseName is a filename with no extension,
	// not even the optional language extension part.
	TranslationBaseName() string

	// ContentBaseName is a either TranslationBaseName or name of containing folder
	// if file is a leaf bundle.
	ContentBaseName() string

	// UniqueID is the MD5 hash of the file's path and is for most practical applications,
	// Hugo content files being one of them, considered to be unique.
	UniqueID() string

	FileInfo() hugofs.FileMetaInfo
}

// FileInfo describes a source file.
type FileInfo struct {

	// Absolute filename to the file on disk.
	filename string

	sp *SourceSpec

	fi hugofs.FileMetaInfo

	// Derived from filename
	ext string // Extension without any "."

	name string

	dir                 string
	relDir              string
	relPath             string
	baseName            string
	translationBaseName string
	contentBaseName     string
	section             string
	classifier          files.ContentClass

	uniqueID string

	lazyInit sync.Once
}

// Path gets the relative path including file name and extension.  The directory
// is relative to the content root.
func (fi *FileInfo) Path() string { return fi.relPath }

// Section returns a file's section.
func (fi *FileInfo) Section() string {
	fi.init()
	return fi.section
}

// We create a lot of these FileInfo objects, but there are parts of it used only
// in some cases that is slightly expensive to construct.
func (fi *FileInfo) init() {
	fi.lazyInit.Do(func() {
		relDir := strings.Trim(fi.relDir, helpers.FilePathSeparator)
		parts := strings.Split(relDir, helpers.FilePathSeparator)
		var section string
		if (fi.classifier != files.ContentClassLeaf && len(parts) == 1) || len(parts) > 1 {
			section = parts[0]
		}
		fi.section = section

		if fi.classifier.IsBundle() && len(parts) > 0 {
			fi.contentBaseName = parts[len(parts)-1]
		} else {
			fi.contentBaseName = fi.translationBaseName
		}

		fi.uniqueID = helpers.MD5String(filepath.ToSlash(fi.relPath))
	})
}

func (fi *FileInfo) IsZero() bool {
	return fi == nil
}

// Dir gets the name of the directory that contains this file.  The directory is
// relative to the content root.
func (fi *FileInfo) Dir() string { return fi.relDir }

// Extension is an alias to Ext().
func (fi *FileInfo) Extension() string {
	helpers.Deprecated(".File.Extension", "Use .File.Ext instead. ", false)
	return fi.Ext()
}

// Ext returns a file's extension without the leading period (ie. "md").
func (fi *FileInfo) Ext() string { return fi.ext }

// Filename returns a file's absolute path and filename on disk.
func (fi *FileInfo) Filename() string { return fi.filename }

// LogicalName returns a file's name and extension (ie. "page.sv.md").
func (fi *FileInfo) LogicalName() string { return fi.name }

// BaseFileName returns a file's name without extension (ie. "page.sv").
func (fi *FileInfo) BaseFileName() string { return fi.baseName }

// TranslationBaseName returns a file's translation base name without the
// language segment (ie. "page").
func (fi *FileInfo) TranslationBaseName() string { return fi.translationBaseName }

// ContentBaseName is a either TranslationBaseName or name of containing folder
// if file is a leaf bundle.
func (fi *FileInfo) ContentBaseName() string {
	fi.init()
	return fi.contentBaseName
}

// UniqueID returns a file's unique, MD5 hash identifier.
func (fi *FileInfo) UniqueID() string {
	fi.init()
	return fi.uniqueID
}

// FileInfo returns a file's underlying os.FileInfo.
func (fi *FileInfo) FileInfo() hugofs.FileMetaInfo { return fi.fi }
