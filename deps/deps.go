package deps

import (
	"fmt"
	"github.com/sunwei/hugo-playground/common/loggers"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/langs"
	"github.com/sunwei/hugo-playground/log"
	"github.com/sunwei/hugo-playground/media"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/resources"
	"github.com/sunwei/hugo-playground/resources/page"
	"github.com/sunwei/hugo-playground/source"
	"github.com/sunwei/hugo-playground/tpl"
)

// Deps holds dependencies used by many.
// There will be normally only one instance of deps in play
// at a given time, i.e. one per Site built.
type Deps struct {
	// The logger to use.
	Log loggers.Logger `json:"-"`

	// The PathSpec to use
	*helpers.PathSpec `json:"-"`

	// The templates to use. This will usually implement the full tpl.TemplateManager.
	tmpl tpl.TemplateHandler

	// We use this to parse and execute ad-hoc text templates.
	textTmpl tpl.TemplateParseFinder

	// All the output formats available for the current site.
	OutputFormatsConfig output.Formats

	// The Resource Spec to use
	ResourceSpec *resources.Spec

	// The SourceSpec to use
	SourceSpec *source.SourceSpec `json:"-"`

	// The ContentSpec to use
	*helpers.ContentSpec `json:"-"`

	// The site building.
	Site page.Site

	// The file systems to use.
	Fs *hugofs.Fs `json:"-"`

	templateProvider ResourceProvider

	// The configuration to use
	Cfg config.Provider `json:"-"`

	// The language in use. TODO(bep) consolidate with site
	Language *langs.Language

	// The translation func to use
	Translate func(translationID string, templateData any) string `json:"-"`
}

// DepsCfg contains configuration options that can be used to configure Hugo
// on a global level, i.e. logging etc.
// Nil values will be given default values.
type DepsCfg struct {
	// The language to use.
	Language *langs.Language

	// The file systems to use
	Fs *hugofs.Fs

	// The Site in use
	Site page.Site

	// The configuration to use.
	Cfg config.Provider

	// The media types configured.
	MediaTypes media.Types

	// The output formats configured.
	OutputFormats output.Formats

	// Template handling.
	TemplateProvider ResourceProvider
}

// ResourceProvider is used to create and refresh, and clone resources needed.
type ResourceProvider interface {
	Update(deps *Deps) error
	Clone(deps *Deps) error
}

func (d *Deps) Tmpl() tpl.TemplateHandler {
	return d.tmpl
}

func (d *Deps) SetTmpl(tmpl tpl.TemplateHandler) {
	d.tmpl = tmpl
}

func (d *Deps) SetTextTmpl(tmpl tpl.TemplateParseFinder) {
	d.textTmpl = tmpl
}

// New initializes a Dep struct.
// Defaults are set for nil values,
// but TemplateProvider, TranslationProvider and Language are always required.
func New(cfg DepsCfg) (*Deps, error) {
	var (
		fs = cfg.Fs
	)

	if cfg.TemplateProvider == nil {
		panic("Must have a TemplateProvider")
	}
	if fs == nil {
		// Default to the production file system.
		panic("Must get fs ready: deps.New")
	}

	if cfg.MediaTypes == nil {
		cfg.MediaTypes = media.DefaultTypes
	}

	if cfg.OutputFormats == nil {
		cfg.OutputFormats = output.DefaultFormats
	}

	log.Process("New PathSpec", "new PathSpec with all source filesystem built")
	ps, err := helpers.NewPathSpec(fs, cfg.Language)
	if err != nil {
		return nil, fmt.Errorf("create PathSpec: %w", err)
	}

	log.Process("New resources Spec", "with pathSpec, outputFormats, MediaTypes")
	resourceSpec, err := resources.NewSpec(ps, cfg.OutputFormats, cfg.MediaTypes)
	if err != nil {
		return nil, err
	}

	log.Process("New content Spec", "content converter provider inside")
	contentSpec, err := helpers.NewContentSpec(cfg.Language, ps.BaseFs.Content.Fs)
	if err != nil {
		return nil, err
	}

	log.Process("New source Spec", "with source filesystem and language")
	sp := source.NewSourceSpec(ps, nil, fs.Source)

	d := &Deps{
		Fs:               fs,
		templateProvider: cfg.TemplateProvider,
		PathSpec:         ps,
		ContentSpec:      contentSpec,
		SourceSpec:       sp,
		ResourceSpec:     resourceSpec,
		Cfg:              cfg.Language,
		Language:         cfg.Language,
		Site:             cfg.Site,
	}

	return d, nil
}

// LoadResources loads translations and templates.
func (d *Deps) LoadResources() error {
	if err := d.templateProvider.Update(d); err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}

	return nil
}

func (d *Deps) TextTmpl() tpl.TemplateParseFinder {
	return d.textTmpl
}
