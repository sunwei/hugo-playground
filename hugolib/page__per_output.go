package hugolib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"github.com/sunwei/hugo-playground/common/text"
	"github.com/sunwei/hugo-playground/common/types/hstring"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/lazy"
	"github.com/sunwei/hugo-playground/markup/converter"
	"github.com/sunwei/hugo-playground/markup/converter/hooks"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/resources/page"
	"github.com/sunwei/hugo-playground/resources/resource"
	"github.com/sunwei/hugo-playground/tpl"
	"html/template"
	"runtime/debug"
	"strings"
	"sync"
	"unicode/utf8"
)

var (
	nopTargetPath    = targetPathsHolder{}
	nopPagePerOutput = struct {
		resource.ResourceLinksProvider
		page.ContentProvider
		page.PageRenderProvider
		page.PaginatorProvider
		page.TableOfContentsProvider
		page.AlternativeOutputFormatsProvider

		targetPather
	}{
		page.NopPage,
		page.NopPage,
		page.NopPage,
		page.NopPage,
		page.NopPage,
		page.NopPage,
		nopTargetPath,
	}
)

// these will be shifted out when rendering a given output format.
type pagePerOutputProviders interface {
	targetPather
	page.PaginatorProvider
	resource.ResourceLinksProvider
}

type targetPather interface {
	targetPaths() page.TargetPaths
}

// pageContentOutput represents the Page content for a given output format.
type pageContentOutput struct {
	f output.Format

	p *pageState

	// Lazy load dependencies
	initMain  *lazy.Init
	initPlain *lazy.Init

	placeholdersEnabled     bool
	placeholdersEnabledInit sync.Once

	// Renders Markdown hooks.
	renderHooks *renderHooks

	workContent       []byte
	dependencyTracker identity.Manager // Set in server mode.

	// Content sections
	content         template.HTML
	summary         template.HTML
	tableOfContents template.HTML

	truncated bool

	plainWords     []string
	plain          string
	fuzzyWordCount int
	wordCount      int
	readingTime    int
}

type renderHooks struct {
	getRenderer hooks.GetRendererFunc
	init        sync.Once
}

type targetPathsHolder struct {
	paths page.TargetPaths
	page.OutputFormat
}

func (t targetPathsHolder) targetPaths() page.TargetPaths {
	return t.paths
}

func newPageContentOutput(p *pageState, po *pageOutput) (*pageContentOutput, error) {
	parent := p.init

	var dependencyTracker identity.Manager
	cp := &pageContentOutput{
		dependencyTracker: dependencyTracker,
		p:                 p,
		f:                 po.f,
		renderHooks:       &renderHooks{},
	}

	initContent := func() (err error) {
		if p.cmap == nil {
			// Nothing to do.
			return nil
		}
		defer func() {
			// See https://github.com/gohugoio/hugo/issues/6210
			if r := recover(); r != nil {
				err = fmt.Errorf("%s", r)
				fmt.Printf("[BUG] Got panic:\n%s\n%s", r, string(debug.Stack()))
			}
		}()

		if err := po.cp.initRenderHooks(); err != nil {
			return err
		}

		cp.workContent = p.contentToRender(p.source.parsed, p.cmap)

		isHTML := cp.p.m.markup == "html"

		if !isHTML {
			r, err := cp.renderContent(cp.workContent, true)
			if err != nil {
				return err
			}

			cp.workContent = r.Bytes()

			if tocProvider, ok := r.(converter.TableOfContentsProvider); ok {
				cfg := p.s.ContentSpec.Converters.GetMarkupConfig()
				cp.tableOfContents = template.HTML(
					tocProvider.TableOfContents().ToHTML(
						cfg.TableOfContents.StartLevel,
						cfg.TableOfContents.EndLevel,
						cfg.TableOfContents.Ordered,
					),
				)
			} else {
				tmpContent, tmpTableOfContents := helpers.ExtractTOC(cp.workContent)
				cp.tableOfContents = helpers.BytesToHTML(tmpTableOfContents)
				cp.workContent = tmpContent
			}
		}

		if cp.p.source.hasSummaryDivider {
			if isHTML {
				src := p.source.parsed.Input()

				// Use the summary sections as they are provided by the user.
				if p.source.posSummaryEnd != -1 {
					cp.summary = helpers.BytesToHTML(src[p.source.posMainContent:p.source.posSummaryEnd])
				}

				if cp.p.source.posBodyStart != -1 {
					cp.workContent = src[cp.p.source.posBodyStart:]
				}

			} else {
				summary, content, err := splitUserDefinedSummaryAndContent(cp.p.m.markup, cp.workContent)
				if err != nil {
					fmt.Printf("Failed to set user defined summary for page %q: %s", cp.p.pathOrTitle(), err)
				} else {
					cp.workContent = content
					cp.summary = helpers.BytesToHTML(summary)
				}
			}
		} else if cp.p.m.summary != "" {
			b, err := cp.renderContent([]byte(cp.p.m.summary), false)
			if err != nil {
				return err
			}
			html := cp.p.s.ContentSpec.TrimShortHTML(b.Bytes())
			cp.summary = helpers.BytesToHTML(html)
		}

		cp.content = helpers.BytesToHTML(cp.workContent)

		return nil
	}

	// There may be recursive loops in shortcodes and render hooks.
	cp.initMain = parent.BranchWithTimeout(p.s.siteCfg.timeout, func(ctx context.Context) (any, error) {
		return nil, initContent()
	})

	cp.initPlain = cp.initMain.Branch(func() (any, error) {
		cp.plain = tpl.StripHTML(string(cp.content))
		cp.plainWords = strings.Fields(cp.plain)
		cp.setWordCounts(p.m.isCJKLanguage)

		if err := cp.setAutoSummary(); err != nil {
			return err, nil
		}

		return nil, nil
	})

	return cp, nil
}

func (p *pageContentOutput) initRenderHooks() error {
	if p == nil {
		return nil
	}

	p.renderHooks.init.Do(func() {
		if p.p.pageOutputTemplateVariationsState.Load() == 0 {
			p.p.pageOutputTemplateVariationsState.Store(1)
		}

		type cacheKey struct {
			tp hooks.RendererType
			id any
			f  output.Format
		}

		renderCache := make(map[cacheKey]any)
		var renderCacheMu sync.Mutex

		resolvePosition := func(ctx any) text.Position {
			var offset int

			switch v := ctx.(type) {
			case hooks.CodeblockContext:
				offset = bytes.Index(p.p.source.parsed.Input(), []byte(v.Inner()))
			}

			pos := p.p.posFromInput(p.p.source.parsed.Input(), offset)

			if pos.LineNumber > 0 {
				// Move up to the code fence delimiter.
				// This is in line with how we report on shortcodes.
				pos.LineNumber = pos.LineNumber - 1
			}

			return pos
		}

		p.renderHooks.getRenderer = func(tp hooks.RendererType, id any) any {
			renderCacheMu.Lock()
			defer renderCacheMu.Unlock()

			key := cacheKey{tp: tp, id: id, f: p.f}
			if r, ok := renderCache[key]; ok {
				return r
			}

			layoutDescriptor := p.p.getLayoutDescriptor()
			layoutDescriptor.RenderingHook = true
			layoutDescriptor.LayoutOverride = false
			layoutDescriptor.Layout = ""

			switch tp {
			case hooks.LinkRendererType:
				layoutDescriptor.Kind = "render-link"
			case hooks.ImageRendererType:
				layoutDescriptor.Kind = "render-image"
			case hooks.HeadingRendererType:
				layoutDescriptor.Kind = "render-heading"
			case hooks.CodeBlockRendererType:
				layoutDescriptor.Kind = "render-codeblock"
				if id != nil {
					lang := id.(string)
					lexer := lexers.Get(lang)
					if lexer != nil {
						layoutDescriptor.KindVariants = strings.Join(lexer.Config().Aliases, ",")
					} else {
						layoutDescriptor.KindVariants = lang
					}
				}
			}

			getHookTemplate := func(f output.Format) (tpl.Template, bool) {
				templ, found, err := p.p.s.Tmpl().LookupLayout(layoutDescriptor, f)
				if err != nil {
					panic(err)
				}
				return templ, found
			}

			templ, found1 := getHookTemplate(p.f)

			if p.p.reusePageOutputContent() {
				// Check if some of the other output formats would give a different template.
				for _, f := range p.p.s.renderFormats {
					if f.Name == p.f.Name {
						continue
					}
					templ2, found2 := getHookTemplate(f)
					if found2 {
						if !found1 {
							templ = templ2
							found1 = true
							break
						}

						if templ != templ2 {
							p.p.pageOutputTemplateVariationsState.Store(2)
							break
						}
					}
				}
			}
			if !found1 {
				if tp == hooks.CodeBlockRendererType {
					// No user provided tempplate for code blocks, so we use the native Go code version -- which is also faster.
					r := p.p.s.ContentSpec.Converters.GetHighlighter()
					renderCache[key] = r
					return r
				}
				return nil
			}

			r := hookRendererTemplate{
				templateHandler: p.p.s.Tmpl(),
				SearchProvider:  templ.(identity.SearchProvider),
				templ:           templ,
				resolvePosition: resolvePosition,
			}
			renderCache[key] = r
			return r
		}
	})

	return nil
}

// A callback to signal that we have inserted a placeholder into the rendered
// content. This avoids doing extra replacement work.
func (p *pageContentOutput) enablePlaceholders() {
	p.placeholdersEnabledInit.Do(func() {
		p.placeholdersEnabled = true
	})
}

func (cp *pageContentOutput) renderContent(content []byte, renderTOC bool) (converter.Result, error) {
	if err := cp.initRenderHooks(); err != nil {
		return nil, err
	}
	c := cp.p.getContentConverter()
	return cp.renderContentWithConverter(c, content, renderTOC)
}

func (cp *pageContentOutput) renderContentWithConverter(c converter.Converter, content []byte, renderTOC bool) (converter.Result, error) {
	r, err := c.Convert(
		converter.RenderContext{
			Src:         content,
			RenderTOC:   renderTOC,
			GetRenderer: cp.renderHooks.getRenderer,
		})

	if err == nil {
		if ids, ok := r.(identity.IdentitiesProvider); ok {
			for _, v := range ids.GetIdentities() {
				cp.trackDependency(v)
			}
		}
	}

	return r, err
}

func (p *pageContentOutput) trackDependency(id identity.Provider) {
	if p.dependencyTracker != nil {
		p.dependencyTracker.Add(id)
	}
}

func splitUserDefinedSummaryAndContent(markup string, c []byte) (summary []byte, content []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("summary split failed: %s", r)
		}
	}()

	startDivider := bytes.Index(c, internalSummaryDividerBaseBytes)

	if startDivider == -1 {
		return
	}

	startTag := "p"
	switch markup {
	case "asciidocext":
		startTag = "div"
	}

	// Walk back and forward to the surrounding tags.
	start := bytes.LastIndex(c[:startDivider], []byte("<"+startTag))
	end := bytes.Index(c[startDivider:], []byte("</"+startTag))

	if start == -1 {
		start = startDivider
	} else {
		start = startDivider - (startDivider - start)
	}

	if end == -1 {
		end = startDivider + len(internalSummaryDividerBase)
	} else {
		end = startDivider + end + len(startTag) + 3
	}

	var addDiv bool

	switch markup {
	case "rst":
		addDiv = true
	}

	withoutDivider := append(c[:start], bytes.Trim(c[end:], "\n")...)

	if len(withoutDivider) > 0 {
		summary = bytes.TrimSpace(withoutDivider[:start])
	}

	if addDiv {
		// For the rst
		summary = append(append([]byte(nil), summary...), []byte("</div>")...)
	}

	if err != nil {
		return
	}

	content = bytes.TrimSpace(withoutDivider)

	return
}

func (p *pageContentOutput) setWordCounts(isCJKLanguage bool) {
	if isCJKLanguage {
		p.wordCount = 0
		for _, word := range p.plainWords {
			runeCount := utf8.RuneCountInString(word)
			if len(word) == runeCount {
				p.wordCount++
			} else {
				p.wordCount += runeCount
			}
		}
	} else {
		p.wordCount = helpers.TotalWords(p.plain)
	}

	// TODO(bep) is set in a test. Fix that.
	if p.fuzzyWordCount == 0 {
		p.fuzzyWordCount = (p.wordCount + 100) / 100 * 100
	}

	if isCJKLanguage {
		p.readingTime = (p.wordCount + 500) / 501
	} else {
		p.readingTime = (p.wordCount + 212) / 213
	}
}

func (p *pageContentOutput) setAutoSummary() error {
	if p.p.source.hasSummaryDivider || p.p.m.summary != "" {
		return nil
	}

	var summary string
	var truncated bool

	if p.p.m.isCJKLanguage {
		summary, truncated = p.p.s.ContentSpec.TruncateWordsByRune(p.plainWords)
	} else {
		summary, truncated = p.p.s.ContentSpec.TruncateWordsToWholeSentence(p.plain)
	}
	p.summary = template.HTML(summary)

	p.truncated = truncated

	return nil
}

func (p *pageContentOutput) Content() (any, error) {
	if p.p.s.initInit(p.initMain, p.p) {
		return p.content, nil
	}
	return nil, nil
}

func (p *pageContentOutput) Plain() string {
	p.p.s.initInit(p.initPlain, p.p)
	return p.plain
}

func (p *pageContentOutput) TableOfContents() template.HTML {
	p.p.s.initInit(p.initMain, p.p)
	return p.tableOfContents
}

func (p *pageContentOutput) PlainWords() []string {
	p.p.s.initInit(p.initPlain, p.p)
	return p.plainWords
}

func (p *pageContentOutput) Summary() template.HTML {
	p.p.s.initInit(p.initMain, p.p)
	if !p.p.source.hasSummaryDivider {
		p.p.s.initInit(p.initPlain, p.p)
	}
	return p.summary
}

func (p *pageContentOutput) Truncated() bool {
	if p.p.truncated {
		return true
	}
	p.p.s.initInit(p.initPlain, p.p)
	return p.truncated
}

func (p *pageContentOutput) WordCount() int {
	p.p.s.initInit(p.initPlain, p.p)
	return p.wordCount
}

func (p *pageContentOutput) FuzzyWordCount() int {
	p.p.s.initInit(p.initPlain, p.p)
	return p.fuzzyWordCount
}

func (p *pageContentOutput) ReadingTime() int {
	p.p.s.initInit(p.initPlain, p.p)
	return p.readingTime
}

func (p *pageContentOutput) Len() int {
	p.p.s.initInit(p.initMain, p.p)
	return len(p.content)
}

type renderStringOpts struct {
	Display string
	Markup  string
}

var defaultRenderStringOpts = renderStringOpts{
	Display: "inline",
	Markup:  "", // Will inherit the page's value when not set.
}

func (p *pageContentOutput) RenderString(args ...any) (template.HTML, error) {
	if len(args) < 1 || len(args) > 2 {
		return "", errors.New("want 1 or 2 arguments")
	}

	var contentToRender string
	opts := defaultRenderStringOpts
	sidx := 1

	if len(args) == 1 {
		sidx = 0
	} else {
		m, ok := args[0].(map[string]any)
		if !ok {
			return "", errors.New("first argument must be a map")
		}

		if err := mapstructure.WeakDecode(m, &opts); err != nil {
			return "", fmt.Errorf("failed to decode options: %w", err)
		}
	}

	contentToRenderv := args[sidx]

	if _, ok := contentToRenderv.(hstring.RenderedString); ok {
		// This content is already rendered, this is potentially
		// a infinite recursion.
		return "", errors.New("text is already rendered, repeating it may cause infinite recursion")
	}

	var err error
	contentToRender, err = cast.ToStringE(contentToRenderv)
	if err != nil {
		return "", err
	}

	if err = p.initRenderHooks(); err != nil {
		return "", err
	}

	conv := p.p.getContentConverter()
	if opts.Markup != "" && opts.Markup != p.p.m.markup {
		var err error
		// TODO(bep) consider cache
		conv, err = p.p.m.newContentConverter(p.p, opts.Markup)
		if err != nil {
			return "", err
		}
	}

	var rendered []byte

	c, err := p.renderContentWithConverter(conv, []byte(contentToRender), false)
	if err != nil {
		return "", err
	}

	rendered = c.Bytes()

	if opts.Display == "inline" {
		// We may have to rethink this in the future when we get other
		// renderers.
		rendered = p.p.s.ContentSpec.TrimShortHTML(rendered)
	}

	return template.HTML(string(rendered)), nil
}
