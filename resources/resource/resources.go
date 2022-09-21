package resource

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/sunwei/hugo-playground/hugofs/glob"
	"strings"
)

// Resources represents a slice of resources, which can be a mix of different types.
// I.e. both pages and images etc.
type Resources []Resource

// Source is an internal template and not meant for use in the templates. It
// may change without notice.
type Source interface {
	Publish() error
}

// ByType returns resources of a given resource type (e.g. "image").
func (r Resources) ByType(typ any) Resources {
	tpstr, err := cast.ToStringE(typ)
	if err != nil {
		panic(err)
	}
	var filtered Resources

	for _, resource := range r {
		if resource.ResourceType() == tpstr {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// Get locates the name given in Resources.
// The search is case insensitive.
func (r Resources) Get(name any) Resource {
	namestr, err := cast.ToStringE(name)
	if err != nil {
		panic(err)
	}
	namestr = strings.ToLower(namestr)
	for _, resource := range r {
		if strings.EqualFold(namestr, resource.Name()) {
			return resource
		}
	}
	return nil
}

// GetMatch finds the first Resource matching the given pattern, or nil if none found.
// See Match for a more complete explanation about the rules used.
func (r Resources) GetMatch(pattern any) Resource {
	patternstr, err := cast.ToStringE(pattern)
	if err != nil {
		panic(err)
	}

	g, err := glob.GetGlob(patternstr)
	if err != nil {
		panic(err)
	}

	for _, resource := range r {
		if g.Match(strings.ToLower(resource.Name())) {
			return resource
		}
	}

	return nil
}

// Match gets all resources matching the given base filename prefix, e.g
// "*.png" will match all png files. The "*" does not match path delimiters (/),
// so if you organize your resources in sub-folders, you need to be explicit about it, e.g.:
// "images/*.png". To match any PNG image anywhere in the bundle you can do "**.png", and
// to match all PNG images below the images folder, use "images/**.jpg".
// The matching is case insensitive.
// Match matches by using the value of Resource.Name, which, by default, is a filename with
// path relative to the bundle root with Unix style slashes (/) and no leading slash, e.g. "images/logo.png".
// See https://github.com/gobwas/glob for the full rules set.
func (r Resources) Match(pattern any) Resources {
	patternstr, err := cast.ToStringE(pattern)
	if err != nil {
		panic(err)
	}

	g, err := glob.GetGlob(patternstr)
	if err != nil {
		panic(err)
	}

	var matches Resources
	for _, resource := range r {
		if g.Match(strings.ToLower(resource.Name())) {
			matches = append(matches, resource)
		}
	}
	return matches
}

type translatedResource interface {
	TranslationKey() string
}

// MergeByLanguage adds missing translations in r1 from r2.
func (r Resources) MergeByLanguage(r2 Resources) Resources {
	result := append(Resources(nil), r...)
	m := make(map[string]bool)
	for _, rr := range r {
		if translated, ok := rr.(translatedResource); ok {
			m[translated.TranslationKey()] = true
		}
	}

	for _, rr := range r2 {
		if translated, ok := rr.(translatedResource); ok {
			if _, found := m[translated.TranslationKey()]; !found {
				result = append(result, rr)
			}
		}
	}
	return result
}

// MergeByLanguageInterface is the generic version of MergeByLanguage. It
// is here just so it can be called from the tpl package.
func (r Resources) MergeByLanguageInterface(in any) (any, error) {
	r2, ok := in.(Resources)
	if !ok {
		return nil, fmt.Errorf("%T cannot be merged by language", in)
	}
	return r.MergeByLanguage(r2), nil
}
