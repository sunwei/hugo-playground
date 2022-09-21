// Copyright 2018 The Hugo Authors. All rights reserved.
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

// Package filesystems provides the fine grained file systems used by Hugo. These
// are typically virtual filesystems that are composites of project and theme content.
package filesystems

import (
	"fmt"
	"github.com/bep/overlayfs"
	"github.com/sunwei/hugo-playground/hugofs/glob"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sunwei/hugo-playground/hugofs/files"

	"github.com/sunwei/hugo-playground/modules"

	"github.com/spf13/afero"
	hpaths "github.com/sunwei/hugo-playground/common/paths"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugolib/paths"
)

const (
	// Used to control concurrency between multiple Hugo instances, e.g.
	// a running server and building new content with 'hugo new'.
	// It's placed in the project root.
	lockFileBuild = ".hugo_build.lock"
)

var filePathSeparator = string(filepath.Separator)

// BaseFs contains the core base filesystems used by Hugo. The name "base" is used
// to underline that even if they can be composites, they all have a base path set to a specific
// resource folder, e.g "/my-project/content". So, no absolute filenames needed.
type BaseFs struct {

	// SourceFilesystems contains the different source file systems.
	*SourceFilesystems

	// The project source.
	SourceFs afero.Fs

	// The filesystem used to publish the rendered site.
	// This usually maps to /my-project/public.
	PublishFs afero.Fs

	// A read-only filesystem starting from the project workDir.
	WorkDir afero.Fs

	theBigFs *filesystemsCollector
}

type Lockable interface {
	Lock() (unlock func(), err error)
}

// TODO(bep) we can get regular files in here and that is fine, but
// we need to clean up the naming.
func (fs *BaseFs) WatchDirs() []hugofs.FileMetaInfo {
	var dirs []hugofs.FileMetaInfo
	for _, dir := range fs.AllDirs() {
		if dir.Meta().Watch {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func (fs *BaseFs) AllDirs() []hugofs.FileMetaInfo {
	var dirs []hugofs.FileMetaInfo
	for _, dirSet := range [][]hugofs.FileMetaInfo{
		fs.Archetypes.Dirs,
		fs.I18n.Dirs,
		fs.Data.Dirs,
		fs.Content.Dirs,
		fs.Assets.Dirs,
		fs.Layouts.Dirs,
		// fs.Resources.Dirs,
		fs.StaticDirs,
	} {
		dirs = append(dirs, dirSet...)
	}

	return dirs
}

// RelContentDir tries to create a path relative to the content root from
// the given filename. The return value is the path and language code.
func (b *BaseFs) RelContentDir(filename string) string {
	for _, dir := range b.SourceFilesystems.Content.Dirs {
		dirname := dir.Meta().Filename
		if strings.HasPrefix(filename, dirname) {
			rel := path.Join(dir.Meta().Path, strings.TrimPrefix(filename, dirname))
			return strings.TrimPrefix(rel, filePathSeparator)
		}
	}
	// Either not a content dir or already relative.
	return filename
}

// AbsProjectContentDir tries to construct a filename below the most
// relevant content directory.
func (b *BaseFs) AbsProjectContentDir(filename string) (string, string, error) {
	isAbs := filepath.IsAbs(filename)
	for _, dir := range b.SourceFilesystems.Content.Dirs {
		meta := dir.Meta()
		if !meta.IsProject {
			continue
		}

		if isAbs {
			if strings.HasPrefix(filename, meta.Filename) {
				return strings.TrimPrefix(filename, meta.Filename), filename, nil
			}
		} else {
			contentDir := strings.TrimPrefix(strings.TrimPrefix(meta.Filename, meta.BaseDir), filePathSeparator) + filePathSeparator

			if strings.HasPrefix(filename, contentDir) {
				relFilename := strings.TrimPrefix(filename, contentDir)
				absFilename := filepath.Join(meta.Filename, relFilename)
				return relFilename, absFilename, nil
			}
		}

	}

	if !isAbs {
		// A filename on the form "posts/mypage.md", put it inside
		// the first content folder, usually <workDir>/content.
		// Pick the first project dir (which is probably the most important one).
		for _, dir := range b.SourceFilesystems.Content.Dirs {
			meta := dir.Meta()
			if meta.IsProject {
				return filename, filepath.Join(meta.Filename, filename), nil
			}
		}

	}

	return "", "", fmt.Errorf("could not determine content directory for %q", filename)
}

// ResolveJSConfigFile resolves the JS-related config file to a absolute
// filename. One example of such would be postcss.config.js.
func (fs *BaseFs) ResolveJSConfigFile(name string) string {
	// First look in assets/_jsconfig
	fi, err := fs.Assets.Fs.Stat(filepath.Join(files.FolderJSConfig, name))
	if err == nil {
		return fi.(hugofs.FileMetaInfo).Meta().Filename
	}
	// Fall back to the work dir.
	fi, err = fs.Work.Stat(name)
	if err == nil {
		return fi.(hugofs.FileMetaInfo).Meta().Filename
	}

	return ""
}

// SourceFilesystems contains the different source file systems. These can be
// composite file systems (theme and project etc.), and they have all root
// set to the source type the provides: data, i18n, static, layouts.
type SourceFilesystems struct {
	Content    *SourceFilesystem
	Data       *SourceFilesystem
	I18n       *SourceFilesystem
	Layouts    *SourceFilesystem
	Archetypes *SourceFilesystem
	Assets     *SourceFilesystem

	// Writable filesystem on top the project's resources directory,
	// with any sub module's resource fs layered below.
	ResourcesCache afero.Fs

	// The work folder (may be a composite of project and theme components).
	Work afero.Fs

	// When in multihost we have one static filesystem per language. The sync
	// static files is currently done outside of the Hugo build (where there is
	// a concept of a site per language).
	// When in non-multihost mode there will be one entry in this map with a blank key.
	Static map[string]*SourceFilesystem

	// All the /static dirs (including themes/modules).
	StaticDirs []hugofs.FileMetaInfo
}

// FileSystems returns the FileSystems relevant for the change detection
// in server mode.
// Note: This does currently not return any static fs.
func (s *SourceFilesystems) FileSystems() []*SourceFilesystem {
	return []*SourceFilesystem{
		s.Content,
		s.Data,
		s.I18n,
		s.Layouts,
		s.Archetypes,
		// TODO(bep) static
	}
}

// A SourceFilesystem holds the filesystem for a given source type in Hugo (data,
// i18n, layouts, static) and additional metadata to be able to use that filesystem
// in server mode.
type SourceFilesystem struct {
	// Name matches one in files.ComponentFolders
	Name string

	// This is a virtual composite filesystem. It expects path relative to a context.
	Fs afero.Fs

	// This filesystem as separate root directories, starting from project and down
	// to the themes/modules.
	Dirs []hugofs.FileMetaInfo

	// When syncing a source folder to the target (e.g. /public), this may
	// be set to publish into a subfolder. This is used for static syncing
	// in multihost mode.
	PublishFolder string
}

// ContentStaticAssetFs will create a new composite filesystem from the content,
// static, and asset filesystems. The site language is needed to pick the correct static filesystem.
// The order is content, static and then assets.
// TODO(bep) check usage
func (s SourceFilesystems) ContentStaticAssetFs(lang string) afero.Fs {
	return overlayfs.New(
		overlayfs.Options{
			Fss: []afero.Fs{
				s.Content.Fs,
				s.StaticFs(lang),
				s.Assets.Fs,
			},
		},
	)

}

// StaticFs returns the static filesystem for the given language.
// This can be a composite filesystem.
func (s SourceFilesystems) StaticFs(lang string) afero.Fs {
	var staticFs afero.Fs = hugofs.NoOpFs

	if fs, ok := s.Static[lang]; ok {
		staticFs = fs.Fs
	} else if fs, ok := s.Static[""]; ok {
		staticFs = fs.Fs
	}

	return staticFs
}

// StatResource looks for a resource in these filesystems in order: static, assets and finally content.
// If found in any of them, it returns FileInfo and the relevant filesystem.
// Any non os.IsNotExist error will be returned.
// An os.IsNotExist error wil be returned only if all filesystems return such an error.
// Note that if we only wanted to find the file, we could create a composite Afero fs,
// but we also need to know which filesystem root it lives in.
func (s SourceFilesystems) StatResource(lang, filename string) (fi os.FileInfo, fs afero.Fs, err error) {
	for _, fsToCheck := range []afero.Fs{s.StaticFs(lang), s.Assets.Fs, s.Content.Fs} {
		fs = fsToCheck
		fi, err = fs.Stat(filename)
		if err == nil || !os.IsNotExist(err) {
			return
		}
	}
	// Not found.
	return
}

// IsStatic returns true if the given filename is a member of one of the static
// filesystems.
func (s SourceFilesystems) IsStatic(filename string) bool {
	for _, staticFs := range s.Static {
		if staticFs.Contains(filename) {
			return true
		}
	}
	return false
}

// IsContent returns true if the given filename is a member of the content filesystem.
func (s SourceFilesystems) IsContent(filename string) bool {
	return s.Content.Contains(filename)
}

// IsLayout returns true if the given filename is a member of the layouts filesystem.
func (s SourceFilesystems) IsLayout(filename string) bool {
	return s.Layouts.Contains(filename)
}

// IsData returns true if the given filename is a member of the data filesystem.
func (s SourceFilesystems) IsData(filename string) bool {
	return s.Data.Contains(filename)
}

// IsAsset returns true if the given filename is a member of the asset filesystem.
func (s SourceFilesystems) IsAsset(filename string) bool {
	return s.Assets.Contains(filename)
}

// IsI18n returns true if the given filename is a member of the i18n filesystem.
func (s SourceFilesystems) IsI18n(filename string) bool {
	return s.I18n.Contains(filename)
}

// MakeStaticPathRelative makes an absolute static filename into a relative one.
// It will return an empty string if the filename is not a member of a static filesystem.
func (s SourceFilesystems) MakeStaticPathRelative(filename string) string {
	for _, staticFs := range s.Static {
		rel, _ := staticFs.MakePathRelative(filename)
		if rel != "" {
			return rel
		}
	}
	return ""
}

// MakePathRelative creates a relative path from the given filename.
func (d *SourceFilesystem) MakePathRelative(filename string) (string, bool) {
	for _, dir := range d.Dirs {
		meta := dir.(hugofs.FileMetaInfo).Meta()
		currentPath := meta.Filename

		if strings.HasPrefix(filename, currentPath) {
			rel := strings.TrimPrefix(filename, currentPath)
			if mp := meta.Path; mp != "" {
				rel = filepath.Join(mp, rel)
			}
			return strings.TrimPrefix(rel, filePathSeparator), true
		}
	}
	return "", false
}

func (d *SourceFilesystem) RealFilename(rel string) string {
	fi, err := d.Fs.Stat(rel)
	if err != nil {
		return rel
	}
	if realfi, ok := fi.(hugofs.FileMetaInfo); ok {
		return realfi.Meta().Filename
	}

	return rel
}

// Contains returns whether the given filename is a member of the current filesystem.
func (d *SourceFilesystem) Contains(filename string) bool {
	for _, dir := range d.Dirs {
		if strings.HasPrefix(filename, dir.Meta().Filename) {
			return true
		}
	}
	return false
}

// Path returns the mount relative path to the given filename if it is a member of
// of the current filesystem, an empty string if not.
func (d *SourceFilesystem) Path(filename string) string {
	for _, dir := range d.Dirs {
		meta := dir.Meta()
		if strings.HasPrefix(filename, meta.Filename) {
			p := strings.TrimPrefix(strings.TrimPrefix(filename, meta.Filename), filePathSeparator)
			if mountRoot := meta.MountRoot; mountRoot != "" {
				return filepath.Join(mountRoot, p)
			}
			return p
		}
	}
	return ""
}

// RealDirs gets a list of absolute paths to directories starting from the given
// path.
func (d *SourceFilesystem) RealDirs(from string) []string {
	var dirnames []string
	for _, dir := range d.Dirs {
		meta := dir.Meta()
		dirname := filepath.Join(meta.Filename, from)
		_, err := meta.Fs.Stat(from)

		if err == nil {
			dirnames = append(dirnames, dirname)
		}
	}
	return dirnames
}

// NewBase builds the filesystems used by Hugo given the paths and options provided.NewBase
func NewBase(p *paths.Paths) (*BaseFs, error) {
	fs := p.Fs

	publishFs := hugofs.NewBaseFileDecorator(fs.PublishDir)
	sourceFs := hugofs.NewBaseFileDecorator(afero.NewBasePathFs(fs.Source, p.WorkingDir))

	b := &BaseFs{
		SourceFs:  sourceFs,
		WorkDir:   fs.WorkingDirReadOnly,
		PublishFs: publishFs,
	}

	if b.theBigFs != nil && b.SourceFilesystems != nil {
		return b, nil
	}

	builder := newSourceFilesystemsBuilder(p, b)
	sourceFilesystems, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("build filesystems: %w", err)
	}

	b.SourceFilesystems = sourceFilesystems
	b.theBigFs = builder.theBigFs

	return b, nil
}

type sourceFilesystemsBuilder struct {
	p        *paths.Paths
	sourceFs afero.Fs
	result   *SourceFilesystems
	theBigFs *filesystemsCollector
}

func newSourceFilesystemsBuilder(p *paths.Paths, b *BaseFs) *sourceFilesystemsBuilder {
	sourceFs := hugofs.NewBaseFileDecorator(p.Fs.Source)
	return &sourceFilesystemsBuilder{p: p, sourceFs: sourceFs, theBigFs: b.theBigFs, result: &SourceFilesystems{}}
}

func (b *sourceFilesystemsBuilder) newSourceFilesystem(name string, fs afero.Fs, dirs []hugofs.FileMetaInfo) *SourceFilesystem {
	return &SourceFilesystem{
		Name: name,
		Fs:   fs,
		Dirs: dirs,
	}
}

func (b *sourceFilesystemsBuilder) Build() (*SourceFilesystems, error) {
	if b.theBigFs == nil {
		// Modules - mounts <-> RootMappingFs - OverlayFS
		theBigFs, err := b.createMainOverlayFs(b.p)
		if err != nil {
			return nil, fmt.Errorf("create main fs: %w", err)
		}

		b.theBigFs = theBigFs
	}

	createView := func(componentID string) *SourceFilesystem {
		if b.theBigFs == nil || b.theBigFs.overlayMounts == nil {
			return b.newSourceFilesystem(componentID, hugofs.NoOpFs, nil)
		}

		dirs := b.theBigFs.overlayDirs[componentID]

		return b.newSourceFilesystem(componentID, afero.NewBasePathFs(b.theBigFs.overlayMounts, componentID), dirs)
	}

	b.result.Archetypes = createView(files.ComponentFolderArchetypes)
	b.result.Layouts = createView(files.ComponentFolderLayouts)
	b.result.Assets = createView(files.ComponentFolderAssets)
	b.result.ResourcesCache = b.theBigFs.overlayResources

	// Data, i18n and content cannot use the overlay fs
	dataDirs := b.theBigFs.overlayDirs[files.ComponentFolderData]
	dataFs, err := hugofs.NewSliceFs(dataDirs...)
	if err != nil {
		return nil, err
	}

	b.result.Data = b.newSourceFilesystem(files.ComponentFolderData, dataFs, dataDirs)

	i18nDirs := b.theBigFs.overlayDirs[files.ComponentFolderI18n]
	i18nFs, err := hugofs.NewSliceFs(i18nDirs...)
	if err != nil {
		return nil, err
	}
	b.result.I18n = b.newSourceFilesystem(files.ComponentFolderI18n, i18nFs, i18nDirs)

	contentDirs := b.theBigFs.overlayDirs[files.ComponentFolderContent]
	contentBfs := afero.NewBasePathFs(b.theBigFs.overlayMountsContent, files.ComponentFolderContent)

	contentFs, err := hugofs.NewLanguageFs(b.p.LanguagesDefaultFirst.AsOrdinalSet(), contentBfs)
	if err != nil {
		return nil, fmt.Errorf("create content filesystem: %w", err)
	}

	b.result.Content = b.newSourceFilesystem(files.ComponentFolderContent, contentFs, contentDirs)

	b.result.Work = afero.NewReadOnlyFs(b.theBigFs.overlayFull)

	// Create static filesystem(s)
	ms := make(map[string]*SourceFilesystem)
	b.result.Static = ms
	b.result.StaticDirs = b.theBigFs.overlayDirs[files.ComponentFolderStatic]

	bfs := afero.NewBasePathFs(b.theBigFs.overlayMountsStatic, files.ComponentFolderStatic)
	ms[""] = b.newSourceFilesystem(files.ComponentFolderStatic, bfs, b.result.StaticDirs)

	return b.result, nil
}

func (b *sourceFilesystemsBuilder) createMainOverlayFs(p *paths.Paths) (*filesystemsCollector, error) {
	collector := &filesystemsCollector{
		sourceProject: b.sourceFs,
		sourceModules: hugofs.NewNoSymlinkFs(b.sourceFs, false),
		overlayDirs:   make(map[string][]hugofs.FileMetaInfo),

		overlayMounts:        overlayfs.New(overlayfs.Options{}),
		overlayMountsContent: overlayfs.New(overlayfs.Options{DirsMerger: hugofs.LanguageDirsMerger}),
		overlayMountsStatic:  overlayfs.New(overlayfs.Options{DirsMerger: hugofs.LanguageDirsMerger}),
		overlayFull:          overlayfs.New(overlayfs.Options{}),
		overlayResources:     overlayfs.New(overlayfs.Options{FirstWritable: true}),
	}

	// Module has same folder structure
	// need to merge, like static files
	mods := p.AllModules
	mounts := make([]mountsDescriptor, len(mods))

	for i := 0; i < len(mods); i++ {
		mod := mods[i]
		dir := mod.Dir()

		isMainProject := mod.Owner() == nil
		mounts[i] = mountsDescriptor{
			Module:        mod,
			dir:           dir,
			isMainProject: isMainProject,
			ordinal:       i,
		}
	}

	err := b.createOverlayFs(collector, mounts)

	return collector, err
}

func (b *sourceFilesystemsBuilder) isContentMount(mnt modules.Mount) bool {
	return strings.HasPrefix(mnt.Target, files.ComponentFolderContent)
}

func (b *sourceFilesystemsBuilder) isStaticMount(mnt modules.Mount) bool {
	return strings.HasPrefix(mnt.Target, files.ComponentFolderStatic)
}

func (b *sourceFilesystemsBuilder) createOverlayFs(
	collector *filesystemsCollector,
	mounts []mountsDescriptor) error {

	for _, md := range mounts {
		var (
			fromTo        []hugofs.RootMapping
			fromToContent []hugofs.RootMapping
			fromToStatic  []hugofs.RootMapping
		)

		absPathify := func(path string) (string, string) {
			if filepath.IsAbs(path) {
				return "", path
			}
			return md.dir, hpaths.AbsPathify(md.dir, path)
		}

		// only project module
		// Mount -> RootMapping -> RootMappingFs -> OverlayFs
		for i, mount := range md.Mounts() {

			// Add more weight to early mounts.
			// When two mounts contain the same filename,
			// the first entry wins.
			mountWeight := (10 + md.ordinal) * (len(md.Mounts()) - i)

			inclusionFilter, err := glob.NewFilenameFilter([]string{}, []string{})
			if err != nil {
				return err
			}

			base, filename := absPathify(mount.Source)

			rm := hugofs.RootMapping{
				From:      mount.Target, // content
				To:        filename,     // mycontent
				ToBasedir: base,
				Module:    md.Module.Path(),
				IsProject: md.isMainProject,
				Meta: &hugofs.FileMeta{
					Watch:           false,
					Weight:          mountWeight,
					Classifier:      files.ContentClassContent,
					InclusionFilter: inclusionFilter,
				},
			}

			isContentMount := b.isContentMount(mount)

			lang := mount.Lang
			if lang == "" && isContentMount {
				lang = "en"
			}
			rm.Meta.Lang = lang

			if isContentMount {
				fromToContent = append(fromToContent, rm)
			} else if b.isStaticMount(mount) {
				fromToStatic = append(fromToStatic, rm)
			} else {
				fromTo = append(fromTo, rm)
			}
		}

		modBase := collector.sourceProject // source fs, project module
		if !md.isMainProject {             // never been here, mytheme has no mounts
			modBase = collector.sourceModules // no symlink source fs
		}

		sourceStatic := hugofs.NewNoSymlinkFs(modBase, true)

		rmfs, err := hugofs.NewRootMappingFs(modBase, fromTo...)
		if err != nil {
			return err
		}
		rmfsContent, err := hugofs.NewRootMappingFs(modBase, fromToContent...)
		if err != nil {
			return err
		}
		rmfsStatic, err := hugofs.NewRootMappingFs(sourceStatic, fromToStatic...)
		if err != nil {
			return err
		}

		// We need to keep the ordered list of directories for watching and
		// some special merge operations (data, i18n).
		collector.addDirs(rmfs)        // add other folders
		collector.addDirs(rmfsContent) // only has /content, why need to go through all components?
		collector.addDirs(rmfsStatic)

		getResourcesDir := func() string {
			if md.isMainProject {
				return b.p.AbsResourcesDir
			}
			_, filename := absPathify(files.FolderResources)
			return filename
		}

		collector.overlayMounts = collector.overlayMounts.Append(rmfs)
		collector.overlayMountsContent = collector.overlayMountsContent.Append(rmfsContent)
		collector.overlayMountsStatic = collector.overlayMountsStatic.Append(rmfsStatic)
		collector.overlayFull = collector.overlayFull.Append(afero.NewBasePathFs(modBase, md.dir))
		collector.overlayResources = collector.overlayResources.Append(afero.NewBasePathFs(modBase, getResourcesDir()))

		fmt.Println(">-=-=-??=")
		fmt.Println(md.dir)

		fmt.Println("print overlay fs:")
		printFs(rmfs, "/")

		fmt.Println("<-=-=-??=")
	}

	return nil
}

func printFs(fs afero.Fs, path string) {
	if fs == nil {
		return
	}
	afero.Walk(fs, path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			fmt.Println("123   all are dir...")
			fmt.Println(path)
			return nil
		}
		var filename string
		if fim, ok := info.(hugofs.FileMetaInfo); ok {
			filename = fim.Meta().Filename
		}
		fmt.Printf("123    %q %q\n", path, filename)
		return nil
	})
}

type filesystemsCollector struct {
	sourceProject afero.Fs // Source for project folders
	sourceModules afero.Fs // Source for modules/themes

	overlayMounts        *overlayfs.OverlayFs
	overlayMountsContent *overlayfs.OverlayFs
	overlayMountsStatic  *overlayfs.OverlayFs
	overlayFull          *overlayfs.OverlayFs
	overlayResources     *overlayfs.OverlayFs

	// Maps component type (layouts, static, content etc.) an ordered list of
	// directories representing the overlay filesystems above.
	overlayDirs map[string][]hugofs.FileMetaInfo
}

func (c *filesystemsCollector) addDirs(rfs *hugofs.RootMappingFs) {
	for _, componentFolder := range files.ComponentFolders {
		c.addDir(rfs, componentFolder)
	}
}

func (c *filesystemsCollector) addDir(rfs *hugofs.RootMappingFs, componentFolder string) {
	dirs, err := rfs.Dirs(componentFolder)

	if err == nil { // event dirs is nil
		// merge all the same component folder from different rfs in the same array
		c.overlayDirs[componentFolder] = append(c.overlayDirs[componentFolder], dirs...)
	}
}

func (c *filesystemsCollector) reverseFis(fis []hugofs.FileMetaInfo) {
	for i := len(fis)/2 - 1; i >= 0; i-- {
		opp := len(fis) - 1 - i
		fis[i], fis[opp] = fis[opp], fis[i]
	}
}

type mountsDescriptor struct {
	modules.Module
	dir           string
	isMainProject bool
	ordinal       int
}
