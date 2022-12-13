package main

import (
	"bytes"
	"fmt"
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/deps"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugolib"
	"github.com/sunwei/hugo-playground/log"
	"golang.org/x/tools/txtar"
	"os"
	"path/filepath"
)

func main() {
	// 0. local example contents

	log.Process("main", "prepare example project file systems")
	tempDir, clean, err := CreateTempDir(hugofs.Os, "go-hugo-temp-dir")
	if err != nil {
		clean()
		os.Exit(-1)
	}

	var afs afero.Fs
	afs = afero.NewOsFs()
	prepareFS(tempDir, afs)

	// 1. config
	log.Process("main", "load configurations from config.toml and themes")
	cfg, _, err := hugolib.LoadConfig(
		hugolib.ConfigSourceDescriptor{
			WorkingDir: tempDir,
			Fs:         afs,
			Filename:   "config.toml",
		},
	)

	// 2. hugo file system
	log.Process("main", "setup hugo file systems based on machine file system and configurations")
	fs := hugofs.NewFrom(afs, cfg, tempDir)

	// 3. dependencies management
	depsCfg := deps.DepsCfg{Cfg: cfg, Fs: fs}

	// 4. hugo sites
	log.Process("main", "create hugo sites based on deps")
	sites, err := hugolib.NewHugoSites(depsCfg)

	// 5. build
	log.Process("main", "hugo building...")
	err = sites.Build(hugolib.BuildCfg{})
	if err != nil {
		fmt.Println("Sites build err")
		fmt.Printf("%#v", err)
		os.Exit(-1)
	}

	fmt.Println("===temp dir at last > ...")
	fmt.Println(tempDir)
}

func prepareFS(workingDir string, afs afero.Fs) {
	files := `
-- config.toml --
theme = "mytheme"
contentDir = "mycontent"
-- myproject.txt --
Hello project!
-- themes/mytheme/mytheme.txt --
Hello theme!
-- mycontent/blog/post.md --
---
title: "Post Title"
---
### first blog
Hello Blog
-- layouts/index.html --
{{ $entries := (readDir ".") }}
START:|{{ range $entry := $entries }}{{ if not $entry.IsDir }}{{ $entry.Name }}|{{ end }}{{ end }}:END:
-- layouts/_default/single.html --
<p>hello single page</p>
{{ .Content }}
===
Static Content
===

  `
	data := txtar.Parse([]byte(files))
	for _, f := range data.Files {
		filename := filepath.Join(workingDir, f.Name)
		data := bytes.TrimSuffix(f.Data, []byte("\n"))

		err := afs.MkdirAll(filepath.Dir(filename), 0777)
		if err != nil {
			fmt.Println(err)
		}
		err = afero.WriteFile(afs, filename, data, 0666)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// CreateTempDir creates a temp dir in the given filesystem and
// returns the dirnam and a func that removes it when done.
func CreateTempDir(fs afero.Fs, prefix string) (string, func(), error) {
	tempDir, err := afero.TempDir(fs, "", prefix)
	if err != nil {
		return "", nil, err
	}

	return tempDir, func() { fs.RemoveAll(tempDir) }, nil
}
