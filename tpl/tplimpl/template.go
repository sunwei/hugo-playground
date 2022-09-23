package tplimpl

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/common/types"
	"github.com/sunwei/hugo-playground/deps"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugofs/files"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/tpl"
	htmltemplate "github.com/sunwei/hugo-playground/tpl/internal/go_templates/htmltemplate"
	"github.com/sunwei/hugo-playground/tpl/internal/go_templates/texttemplate"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

const (
	textTmplNamePrefix = "_text/"

	shortcodesPathPrefix = "shortcodes/"
	internalPathPrefix   = "_internal/"
	baseFileBase         = "baseof"
)

type templateType int

type templateState struct {
	tpl.Template

	typ       templateType
	parseInfo tpl.ParseInfo
	identity.Manager

	info     templateInfo
	baseInfo templateInfo // Set when a base template is used.
}

type templateStateMap struct {
	mu        sync.RWMutex
	templates map[string]*templateState
}

type templateNamespace struct {
	prototypeText      *texttemplate.Template
	prototypeHTML      *htmltemplate.Template
	prototypeTextClone *texttemplate.Template
	prototypeHTMLClone *htmltemplate.Template

	*templateStateMap
}

type templateHandler struct {
	main          *templateNamespace
	readyInit     sync.Once
	layoutHandler *output.LayoutHandler
	*deps.Deps
}

type templateExec struct {
	d        *deps.Deps
	executor texttemplate.Executer
	funcs    map[string]reflect.Value

	*templateHandler
}

func newTemplateExec(d *deps.Deps) (*templateExec, error) {
	exec, funcs := newTemplateExecuter(d)
	funcMap := make(map[string]any)
	for k, v := range funcs {
		funcMap[k] = v.Interface()
	}

	h := &templateHandler{
		main: newTemplateNamespace(funcMap),

		Deps:          d,
		layoutHandler: output.NewLayoutHandler(),
	}

	if err := h.loadEmbedded(); err != nil {
		fmt.Println("error load embedded")
		return nil, err
	}

	if err := h.loadTemplates(); err != nil {
		fmt.Println("error load templates")
		fmt.Printf("%#v", err)
		return nil, err
	}

	e := &templateExec{
		d:               d,
		executor:        exec,
		funcs:           funcs,
		templateHandler: h,
	}

	d.SetTmpl(e)
	d.SetTextTmpl(newStandaloneTextTemplate(funcMap))

	return e, nil
}

func newTemplateNamespace(funcs map[string]any) *templateNamespace {
	return &templateNamespace{
		prototypeHTML: htmltemplate.New("").Funcs(funcs),
		prototypeText: texttemplate.New("").Funcs(funcs),
		templateStateMap: &templateStateMap{
			templates: make(map[string]*templateState),
		},
	}
}

//go:embed embedded/templates/*
//go:embed embedded/templates/_default/*
//go:embed embedded/templates/_server/*
var embededTemplatesFs embed.FS

func (t *templateHandler) loadEmbedded() error {
	return fs.WalkDir(embededTemplatesFs, ".", func(path string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}

		templb, err := embededTemplatesFs.ReadFile(path)
		if err != nil {
			return err
		}

		// Get the newlines on Windows in line with how we had it back when we used Go Generate
		// to write the templates to Go files.
		templ := string(bytes.ReplaceAll(templb, []byte("\r\n"), []byte("\n")))
		name := strings.TrimPrefix(filepath.ToSlash(path), "embedded/templates/")
		templateName := name

		// For the render hooks and the server templates it does not make sense to preseve the
		// double _indternal double book-keeping,
		// just add it if its now provided by the user.
		if !strings.Contains(path, "_default/_markup") && !strings.HasPrefix(name, "_server/") {
			templateName = internalPathPrefix + name
		}

		if _, found := t.Lookup(templateName); !found {
			// parse template to tree
			if err := t.AddTemplate(templateName, templ); err != nil {
				fmt.Println("add template err:")
				fmt.Println(err)
				return err
			}
		}

		return nil
	})
}

func (t *templateHandler) Lookup(name string) (tpl.Template, bool) {
	templ, found := t.main.Lookup(name)
	if found {
		return templ, true
	}

	return nil, false
}

func (t *templateNamespace) Lookup(name string) (tpl.Template, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	templ, found := t.templates[name]
	if !found {
		return nil, false
	}

	return templ, found
}

// AddTemplate parses and adds a template to the collection.
// Templates with name prefixed with "_text" will be handled as plain
// text templates.
func (t *templateHandler) AddTemplate(name, tpl string) error {
	templ, err := t.addTemplateTo(t.newTemplateInfo(name, tpl), t.main)
	if err == nil {
		_, _ = t.applyTemplateTransformers(t.main, templ)
	}
	return err
}

func (t *templateHandler) newTemplateInfo(name, tpl string) templateInfo {
	var isText bool
	name, isText = t.nameIsText(name)
	return templateInfo{
		name:     name,
		isText:   isText,
		template: tpl,
	}
}

func (t *templateHandler) nameIsText(name string) (string, bool) {
	isText := strings.HasPrefix(name, textTmplNamePrefix)
	if isText {
		name = strings.TrimPrefix(name, textTmplNamePrefix)
	}
	return name, isText
}

func (t *templateHandler) addTemplateTo(info templateInfo, to *templateNamespace) (*templateState, error) {
	return to.parse(info)
}

func (t *templateNamespace) parse(info templateInfo) (*templateState, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if info.isText {
		prototype := t.prototypeText

		templ, err := prototype.New(info.name).Parse(info.template)
		if err != nil {
			return nil, err
		}

		ts := newTemplateState(templ, info)

		t.templates[info.name] = ts

		return ts, nil
	}

	prototype := t.prototypeHTML

	templ, err := prototype.New(info.name).Parse(info.template)
	if err != nil {
		return nil, err
	}

	ts := newTemplateState(templ, info)

	t.templates[info.name] = ts

	return ts, nil
}

func newTemplateState(templ tpl.Template, info templateInfo) *templateState {
	return &templateState{
		info:      info,
		typ:       info.resolveType(),
		Template:  templ,
		Manager:   newIdentity(info.name),
		parseInfo: tpl.DefaultParseInfo,
	}
}

func newIdentity(name string) identity.Manager {
	return identity.NewManager(identity.NewPathIdentity(files.ComponentFolderLayouts, name))
}

func (t *templateHandler) applyTemplateTransformers(ns *templateNamespace, ts *templateState) (*templateContext, error) {
	c, err := applyTemplateTransformers(ts, ns.newTemplateLookup(ts))
	if err != nil {
		return nil, err
	}

	return c, err
}

func (t *templateNamespace) newTemplateLookup(in *templateState) func(name string) *templateState {
	return func(name string) *templateState {
		if templ, found := t.templates[name]; found {
			if templ.isText() != in.isText() {
				return nil
			}
			return templ
		}
		if templ, found := findTemplateIn(name, in); found {
			return newTemplateState(templ, templateInfo{name: templ.Name()})
		}
		return nil
	}
}

func (t *templateState) isText() bool {
	return isText(t.Template)
}

func isText(templ tpl.Template) bool {
	_, isText := templ.(*texttemplate.Template)
	return isText
}

func unwrap(templ tpl.Template) tpl.Template {
	if ts, ok := templ.(*templateState); ok {
		return ts.Template
	}
	return templ
}

func (t *templateHandler) loadTemplates() error {
	walker := func(path string, fi hugofs.FileMetaInfo, err error) error {
		if err != nil || fi.IsDir() {
			fmt.Println("walker err 1")
			return err
		}

		if isDotFile(path) || isBackupFile(path) {
			fmt.Println("walker err 2")
			return nil
		}

		name := strings.TrimPrefix(filepath.ToSlash(path), "/")
		filename := filepath.Base(path)
		outputFormat, found := t.OutputFormatsConfig.FromFilename(filename)

		if found && outputFormat.IsPlainText {
			name = textTmplNamePrefix + name
		}

		if err := t.addTemplateFile(name, path); err != nil {
			return err
		}

		return nil
	}

	if err := helpers.SymbolicWalk(t.Layouts.Fs, "", walker); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	return nil
}

func isDotFile(path string) bool {
	return filepath.Base(path)[0] == '.'
}

func isBackupFile(path string) bool {
	return path[len(path)-1] == '~'
}

func (t *templateHandler) addTemplateFile(name, path string) error {
	getTemplate := func(filename string) (templateInfo, error) {
		afs := t.Layouts.Fs
		b, err := afero.ReadFile(afs, filename)
		if err != nil {
			return templateInfo{filename: filename, fs: afs}, err
		}

		s := removeLeadingBOM(string(b))

		realFilename := filename
		if fi, err := afs.Stat(filename); err == nil {
			if fim, ok := fi.(hugofs.FileMetaInfo); ok {
				realFilename = fim.Meta().Filename
			}
		}

		var isText bool
		name, isText = t.nameIsText(name)

		return templateInfo{
			name:         name,
			isText:       isText,
			template:     s,
			filename:     filename,
			realFilename: realFilename,
			fs:           afs,
		}, nil
	}

	tinfo, err := getTemplate(path)
	if err != nil {
		return err
	}

	templ, err := t.addTemplateTo(tinfo, t.main)
	if err != nil {
		return tinfo.errWithFileContext("parse failed", err)
	}
	_, err = t.applyTemplateTransformers(t.main, templ)
	if err != nil {
		return err
	}

	return nil
}

func removeLeadingBOM(s string) string {
	const bom = '\ufeff'

	for i, r := range s {
		if i == 0 && r != bom {
			return s
		}
		if i > 0 {
			return s[i:]
		}
	}

	return s
}

func (t templateExec) Clone(d *deps.Deps) *templateExec {
	exec, funcs := newTemplateExecuter(d)
	t.executor = exec
	t.funcs = funcs
	t.d = d
	return &t
}

func (t *templateExec) Execute(templ tpl.Template, wr io.Writer, data any) error {
	return t.ExecuteWithContext(context.Background(), templ, wr, data)
}

func (t *templateExec) ExecuteWithContext(ctx context.Context, templ tpl.Template, wr io.Writer, data any) error {
	if rlocker, ok := templ.(types.RLocker); ok {
		rlocker.RLock()
		defer rlocker.RUnlock()
	}

	execErr := t.executor.ExecuteWithContext(ctx, templ, wr, data)
	if execErr != nil {
		fmt.Println("ExecuteWithContext error happens ...")
		execErr = t.addFileContext(templ, execErr)
	}
	return execErr
}

func (t *templateHandler) addFileContext(templ tpl.Template, inerr error) error {
	if strings.HasPrefix(templ.Name(), "_internal") {
		return inerr
	}

	ts, ok := templ.(*templateState)
	if !ok {
		return inerr
	}

	//lint:ignore ST1008 the error is the main result
	checkFilename := func(info templateInfo, inErr error) (error, bool) {
		if info.filename == "" {
			return inErr, false
		}

		return inErr, true
	}

	inerr = fmt.Errorf("execute of template failed: %w", inerr)

	if err, ok := checkFilename(ts.info, inerr); ok {
		return err
	}

	err, _ := checkFilename(ts.baseInfo, inerr)

	return err
}

// The identifiers may be truncated in the log, e.g.
// "executing "main" at <$scaled.SRelPermalin...>: can't evaluate field SRelPermalink in type *resource.Image"
// We need this to identify position in templates with base templates applied.
var identifiersRe = regexp.MustCompile(`at \<(.*?)(\.{3})?\>:`)

func (t *templateHandler) extractIdentifiers(line string) []string {
	m := identifiersRe.FindAllStringSubmatch(line, -1)
	identifiers := make([]string, len(m))
	for i := 0; i < len(m); i++ {
		identifiers[i] = m[i][1]
	}
	return identifiers
}

func (t *templateHandler) LookupLayout(d output.LayoutDescriptor, f output.Format) (tpl.Template, bool, error) {
	return t.findLayout(d, f)
}

func (t *templateHandler) HasTemplate(name string) bool {
	_, found := t.Lookup(name)
	return found
}

func (t *templateHandler) findLayout(d output.LayoutDescriptor, f output.Format) (tpl.Template, bool, error) {
	layouts, _ := t.layoutHandler.For(d, f) // construct layouts name
	for _, name := range layouts {
		templ, found := t.main.Lookup(name)
		if found {
			return templ, true, nil
		}
	}

	return nil, false, nil
}

func (t *templateHandler) applyBaseTemplate(overlay, base templateInfo) (tpl.Template, error) {
	if overlay.isText {
		var (
			templ = t.main.prototypeTextClone.New(overlay.name)
			err   error
		)

		if !base.IsZero() {
			templ, err = templ.Parse(base.template)
			if err != nil {
				return nil, base.errWithFileContext("parse failed", err)
			}
		}

		templ, err = texttemplate.Must(templ.Clone()).Parse(overlay.template)
		if err != nil {
			return nil, overlay.errWithFileContext("parse failed", err)
		}

		// The extra lookup is a workaround, see
		// * https://github.com/golang/go/issues/16101
		// * https://github.com/gohugoio/hugo/issues/2549
		// templ = templ.Lookup(templ.Name())

		return templ, nil
	}

	var (
		templ = t.main.prototypeHTMLClone.New(overlay.name)
		err   error
	)

	if !base.IsZero() {
		templ, err = templ.Parse(base.template)
		if err != nil {
			return nil, base.errWithFileContext("parse failed", err)
		}
	}

	templ, err = htmltemplate.Must(templ.Clone()).Parse(overlay.template)
	if err != nil {
		return nil, overlay.errWithFileContext("parse failed", err)
	}

	// The extra lookup is a workaround, see
	// * https://github.com/golang/go/issues/16101
	// * https://github.com/gohugoio/hugo/issues/2549
	templ = templ.Lookup(templ.Name())

	return templ, err
}

func (t *templateHandler) extractPartials(templ tpl.Template) error {
	templs := templates(templ)
	for _, templ := range templs {
		if templ.Name() == "" || !strings.HasPrefix(templ.Name(), "partials/") {
			continue
		}

		panic("extract partials not supported yet")
	}

	return nil
}

func templates(in tpl.Template) []tpl.Template {
	var templs []tpl.Template
	in = unwrap(in)
	if textt, ok := in.(*texttemplate.Template); ok {
		for _, t := range textt.Templates() {
			templs = append(templs, t)
		}
	}

	if htmlt, ok := in.(*htmltemplate.Template); ok {
		for _, t := range htmlt.Templates() {
			templs = append(templs, t)
		}
	}

	return templs
}

func newStandaloneTextTemplate(funcs map[string]any) tpl.TemplateParseFinder {
	return &textTemplateWrapperWithLock{
		RWMutex:  &sync.RWMutex{},
		Template: texttemplate.New("").Funcs(funcs),
	}
}

func (t *textTemplateWrapperWithLock) Lookup(name string) (tpl.Template, bool) {
	t.RLock()
	templ := t.Template.Lookup(name)
	t.RUnlock()
	if templ == nil {
		return nil, false
	}
	return &textTemplateWrapperWithLock{
		RWMutex:  t.RWMutex,
		Template: templ,
	}, true
}

func (t *textTemplateWrapperWithLock) LookupVariant(name string, variants tpl.TemplateVariants) (tpl.Template, bool, bool) {
	panic("not supported")
}

func (t *textTemplateWrapperWithLock) LookupVariants(name string) []tpl.Template {
	panic("not supported")
}

type textTemplateWrapperWithLock struct {
	*sync.RWMutex
	*texttemplate.Template
}

func (t *textTemplateWrapperWithLock) Parse(name, tpl string) (tpl.Template, error) {
	t.Lock()
	defer t.Unlock()
	return t.Template.New(name).Parse(tpl)
}

func (t *templateHandler) postTransform() error {
	defineCheckedHTML := false
	defineCheckedText := false

	for _, v := range t.main.templates {
		if defineCheckedHTML && defineCheckedText {
			continue
		}

		isText := isText(v.Template)
		if isText {
			if defineCheckedText {
				continue
			}
			defineCheckedText = true
		} else {
			if defineCheckedHTML {
				continue
			}
			defineCheckedHTML = true
		}

		if err := t.extractPartials(v.Template); err != nil {
			return err
		}
	}

	return nil
}

func (t *templateHandler) findTemplate(name string) *templateState {
	if templ, found := t.Lookup(name); found {
		return templ.(*templateState)
	}
	return nil
}

func (t *templateExec) GetFunc(name string) (reflect.Value, bool) {
	v, found := t.funcs[name]
	return v, found
}

func (t *templateExec) MarkReady() error {
	var err error
	t.readyInit.Do(func() {
		fmt.Println("We only need the clones if base templates are in use.")
	})

	return err
}

func (t *templateNamespace) createPrototypes() error {
	t.prototypeTextClone = texttemplate.Must(t.prototypeText.Clone())
	t.prototypeHTMLClone = htmltemplate.Must(t.prototypeHTML.Clone())

	return nil
}
