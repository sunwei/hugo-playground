package hugolib

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/sunwei/hugo-playground/common/text"
)

func newPageRef(p *pageState) pageRef {
	return pageRef{p: p}
}

type pageRef struct {
	p *pageState
}

func (p pageRef) Ref(argsm map[string]any) (string, error) {
	return p.ref(argsm, p.p)
}

func (p pageRef) RefFrom(argsm map[string]any, source any) (string, error) {
	return p.ref(argsm, source)
}

func (p pageRef) RelRef(argsm map[string]any) (string, error) {
	return p.relRef(argsm, p.p)
}

func (p pageRef) RelRefFrom(argsm map[string]any, source any) (string, error) {
	return p.relRef(argsm, source)
}

func (p pageRef) decodeRefArgs(args map[string]any) (refArgs, *Site, error) {
	var ra refArgs
	err := mapstructure.WeakDecode(args, &ra)
	if err != nil {
		return ra, nil, nil
	}

	s := p.p.s

	if ra.Lang != "" && ra.Lang != p.p.s.Language().Lang {
		// Find correct site
		found := false
		for _, ss := range p.p.s.h.Sites {
			if ss.Lang() == ra.Lang {
				found = true
				s = ss
			}
		}

		if !found {
			fmt.Printf(ra.Path, fmt.Sprintf("no site found with lang %q", ra.Lang), nil, text.Position{})
			return ra, nil, nil
		}
	}

	return ra, s, nil
}

func (p pageRef) ref(argsm map[string]any, source any) (string, error) {
	args, s, err := p.decodeRefArgs(argsm)
	if err != nil {
		return "", fmt.Errorf("invalid arguments to Ref: %w", err)
	}

	if s == nil {
		return p.p.s.siteRefLinker.notFoundURL, nil
	}

	if args.Path == "" {
		return "", nil
	}

	return s.refLink(args.Path, source, false, args.OutputFormat)
}

func (p pageRef) relRef(argsm map[string]any, source any) (string, error) {
	args, s, err := p.decodeRefArgs(argsm)
	if err != nil {
		return "", fmt.Errorf("invalid arguments to Ref: %w", err)
	}

	if s == nil {
		return p.p.s.siteRefLinker.notFoundURL, nil
	}

	if args.Path == "" {
		return "", nil
	}

	return s.refLink(args.Path, source, true, args.OutputFormat)
}

type refArgs struct {
	Path         string
	Lang         string
	OutputFormat string
}
