package source

import (
	"fmt"
	"github.com/sunwei/hugo-playground/hugofs"
	"path/filepath"
	"sync"
)

// Filesystem represents a source filesystem.
type Filesystem struct {
	files        []File
	filesInit    sync.Once
	filesInitErr error

	Base string

	fi hugofs.FileMetaInfo

	SourceSpec
}

func (sp SourceSpec) NewFilesystemFromFileMetaInfo(fi hugofs.FileMetaInfo) *Filesystem {
	return &Filesystem{SourceSpec: sp, fi: fi}
}

// Files returns a slice of readable files.
func (f *Filesystem) Files() ([]File, error) {
	f.filesInit.Do(func() {
		err := f.captureFiles()
		if err != nil {
			f.filesInitErr = fmt.Errorf("capture files: %w", err)
		}
	})
	return f.files, f.filesInitErr
}

func (f *Filesystem) captureFiles() error {
	walker := func(path string, fi hugofs.FileMetaInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		meta := fi.Meta()
		filename := meta.Filename

		b, err := f.shouldRead(filename, fi)
		if err != nil {
			return err
		}

		if b {
			err = f.add(filename, fi)
		}

		return err
	}

	w := hugofs.NewWalkway(hugofs.WalkwayConfig{
		Fs:     f.SourceFs,
		Info:   f.fi,
		Root:   f.Base,
		WalkFn: walker,
	})

	return w.Walk()
}

func (f *Filesystem) shouldRead(filename string, fi hugofs.FileMetaInfo) (bool, error) {
	fmt.Println("file system should read")
	fmt.Println(fi.Meta().Filename)
	fmt.Printf("==")
	ignore := f.SourceSpec.IgnoreFile(fi.Meta().Filename)

	if fi.IsDir() {
		if ignore {
			return false, filepath.SkipDir
		}
		return false, nil
	}

	if ignore {
		return false, nil
	}

	return true, nil
}

// add populates a file in the Filesystem.files
func (f *Filesystem) add(name string, fi hugofs.FileMetaInfo) (err error) {
	var file File

	file, err = f.SourceSpec.NewFileInfo(fi)
	if err != nil {
		return err
	}

	f.files = append(f.files, file)

	return err
}
