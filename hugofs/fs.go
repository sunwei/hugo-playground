// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package hugofs provides the file systems used by Hugo.
package hugofs

import (
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/common/paths"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/log"
	"os"
)

// Os points to the (real) Os filesystem.
var Os = &afero.OsFs{}

// Fs holds the core filesystems used by Hugo.
type Fs struct {
	// Source is Hugo's source file system.
	// Note that this will always be a "plain" Afero filesystem:
	// * afero.OsFs when running in production
	// * afero.MemMapFs for many of the tests.
	Source afero.Fs

	// PublishDir is where Hugo publishes its rendered content.
	// It's mounted inside publishDir (default /public).
	PublishDir afero.Fs

	// WorkingDirReadOnly is a read-only file system
	// restricted to the project working dir.
	WorkingDirReadOnly afero.Fs
}

// NewFrom creates a new Fs based on the provided Afero Fs
// as source and destination file systems.
// Useful for testing.
func NewFrom(fs afero.Fs, cfg config.Provider, wd string) *Fs {
	return newFs(fs, fs, cfg, wd)
}

func newFs(source, destination afero.Fs, cfg config.Provider, wd string) *Fs {
	cfg.Set("workingDir", wd)
	workingDir := cfg.GetString("workingDir")
	publishDir := cfg.GetString("publishDir")

	absPublishDir := paths.AbsPathify(workingDir, publishDir)

	// Make sure we always have the /public folder ready to use.
	if err := source.MkdirAll(absPublishDir, 0777); err != nil && !os.IsExist(err) {
		panic(err)
	}
	log.Process("newFs", "create /public folder")

	log.Process("newFs", "new base path fs &BasePathFs{}")
	pubFs := afero.NewBasePathFs(destination, absPublishDir)

	return &Fs{
		Source:             source,
		PublishDir:         pubFs,
		WorkingDirReadOnly: getWorkingDirFsReadOnly(source, workingDir),
	}
}

func getWorkingDirFsReadOnly(base afero.Fs, workingDir string) afero.Fs {
	if workingDir == "" {
		return afero.NewReadOnlyFs(base)
	}
	return afero.NewBasePathFs(afero.NewReadOnlyFs(base), workingDir)
}

func isWrite(flag int) bool {
	return flag&os.O_RDWR != 0 || flag&os.O_WRONLY != 0
}

// FilesystemsUnwrapper returns the underlying filesystems.
type FilesystemsUnwrapper interface {
	UnwrapFilesystems() []afero.Fs
}

// FilesystemsProvider returns the underlying filesystem.
type FilesystemUnwrapper interface {
	UnwrapFilesystem() afero.Fs
}

// WalkFn is the walk func for WalkFilesystems.
type WalkFn func(fs afero.Fs) bool
