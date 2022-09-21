package hugolib

import (
	"context"
	"fmt"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/hugofs"
	"github.com/sunwei/hugo-playground/hugofs/files"
	"github.com/sunwei/hugo-playground/source"
	"golang.org/x/sync/errgroup"
)

func newPagesProcessor(h *HugoSites, sp *source.SourceSpec) *pagesProcessor {
	procs := make(map[string]pagesCollectorProcessorProvider)
	for _, s := range h.Sites {
		procs[s.Lang()] = &sitePagesProcessor{
			m:        s.pageMap,
			itemChan: make(chan interface{}, config.GetNumWorkerMultiplier()*2),
		}
	}
	return &pagesProcessor{
		procs: procs,
	}
}

type pagesProcessor struct {
	// Per language/Site
	procs map[string]pagesCollectorProcessorProvider
}

type pageBundles map[string]*fileinfoBundle

func (proc *pagesProcessor) Process(item any) error {
	fmt.Println("page processor ")
	fmt.Printf("%#v", item)
	fmt.Println("?????")

	switch v := item.(type) {
	// Page bundles mapped to their language.
	case pageBundles:
		for _, vv := range v {
			proc.getProcFromFi(vv.header).Process(vv)
		}
	case hugofs.FileMetaInfo:
		proc.getProcFromFi(v).Process(v)
	default:
		panic(fmt.Sprintf("unrecognized item type in Process: %T", item))

	}

	return nil
}

func (proc *pagesProcessor) Start(ctx context.Context) context.Context {
	for _, p := range proc.procs {
		ctx = p.Start(ctx)
	}
	return ctx
}

func (proc *pagesProcessor) Wait() error {
	var err error
	for _, p := range proc.procs {
		if e := p.Wait(); e != nil {
			err = e
		}
	}
	return err
}

type pagesCollectorProcessorProvider interface {
	Process(item any) error
	Start(ctx context.Context) context.Context
	Wait() error
}

type sitePagesProcessor struct {
	m         *pageMap
	ctx       context.Context
	itemChan  chan any
	itemGroup *errgroup.Group
}

func (p *sitePagesProcessor) Process(item any) error {
	select {
	case <-p.ctx.Done():
		return nil
	default:
		p.itemChan <- item
	}
	return nil
}

func (p *sitePagesProcessor) Start(ctx context.Context) context.Context {
	p.itemGroup, ctx = errgroup.WithContext(ctx)
	p.ctx = ctx
	p.itemGroup.Go(func() error {
		for item := range p.itemChan {
			if err := p.doProcess(item); err != nil {
				return err
			}
		}
		return nil
	})
	return ctx
}

func (p *sitePagesProcessor) Wait() error {
	close(p.itemChan)
	return p.itemGroup.Wait()
}

func (p *sitePagesProcessor) doProcess(item any) error {
	m := p.m
	switch v := item.(type) {
	case hugofs.FileMetaInfo:
		meta := v.Meta()

		classifier := meta.Classifier
		switch classifier {
		case files.ContentClassContent: // basefs.go createOverlayFs
			if err := m.AddFilesBundle(v); err != nil {
				return err
			}
		case files.ContentClassFile:
			panic("doProcess not support ContentClassFile yet")
		default:
			panic(fmt.Sprintf("invalid classifier: %q", classifier))
		}
	default:
		panic(fmt.Sprintf("unrecognized item type in Process: %T", item))
	}
	return nil
}

var defaultPageProcessor = new(nopPageProcessor)

func (proc *pagesProcessor) getProcFromFi(fi hugofs.FileMetaInfo) pagesCollectorProcessorProvider {
	if p, found := proc.procs["en"]; found {
		return p
	}
	return defaultPageProcessor
}

type nopPageProcessor int

func (nopPageProcessor) Process(item any) error {
	return nil
}

func (nopPageProcessor) Start(ctx context.Context) context.Context {
	return context.Background()
}

func (nopPageProcessor) Wait() error {
	return nil
}
