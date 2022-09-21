package config

import (
	"github.com/spf13/cast"
	"github.com/sunwei/hugo-playground/common/maps"
	"strings"
	"sync"
)

type KeyParams struct {
	Key    string
	Params maps.Params
}

// New creates a Provider backed by an empty maps.Params.
func New() Provider {
	return &defaultConfigProvider{
		root: make(maps.Params),
	}
}

// NewFrom creates a Provider backed by params.
func NewFrom(params maps.Params) Provider {
	return &defaultConfigProvider{
		root: params,
	}
}

// defaultConfigProvider is a Provider backed by a map where all keys are lower case.
// All methods are thread safe.
type defaultConfigProvider struct {
	mu   sync.RWMutex
	root maps.Params

	keyCache sync.Map
}

func (c *defaultConfigProvider) SetDefaultMergeStrategy() {
}

func (c *defaultConfigProvider) Get(k string) any {
	if k == "" {
		return c.root
	}
	c.mu.RLock()
	key, m := c.getNestedKeyAndMap(strings.ToLower(k), false)
	if m == nil {
		c.mu.RUnlock()
		return nil
	}
	v := m[key]
	c.mu.RUnlock()
	return v
}

func (c *defaultConfigProvider) GetBool(k string) bool {
	return false
}

func (c *defaultConfigProvider) GetInt(k string) int {
	return 0
}

func (c *defaultConfigProvider) IsSet(k string) bool {
	var found bool
	c.mu.RLock()
	key, m := c.getNestedKeyAndMap(strings.ToLower(k), false)
	if m != nil {
		_, found = m[key]
	}
	c.mu.RUnlock()
	return found
}

func (c *defaultConfigProvider) GetString(k string) string {
	v := c.Get(k)
	return cast.ToString(v)
}

func (c *defaultConfigProvider) GetParams(k string) maps.Params {
	v := c.Get(k)
	if v == nil {
		return nil
	}
	return v.(maps.Params)
}

func (c *defaultConfigProvider) GetStringMap(k string) map[string]any {
	return nil
}

func (c *defaultConfigProvider) GetStringMapString(k string) map[string]string {
	v := c.Get(k)
	return maps.ToStringMapString(v)
}

func (c *defaultConfigProvider) GetStringSlice(k string) []string {
	return nil
}

func (c *defaultConfigProvider) Set(k string, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	k = strings.ToLower(k)

	if k == "" {
		if p, ok := maps.ToParamsAndPrepare(v); ok {
			// Set the values directly in root.
			c.root.Set(p)
		} else {
			c.root[k] = v
		}

		return
	}

	switch vv := v.(type) {
	case map[string]any, map[any]any, map[string]string:
		p := maps.MustToParamsAndPrepare(vv)
		v = p
	}

	key, m := c.getNestedKeyAndMap(k, true)
	if m == nil {
		return
	}

	if existing, found := m[key]; found {
		if p1, ok := existing.(maps.Params); ok {
			if p2, ok := v.(maps.Params); ok {
				p1.Set(p2)
				return
			}
		}
	}

	m[key] = v
}

// SetDefaults will set values from params if not already set.
func (c *defaultConfigProvider) SetDefaults(params maps.Params) {
	maps.PrepareParams(params)
	for k, v := range params {
		if _, found := c.root[k]; !found {
			c.root[k] = v
		}
	}
}

func (c *defaultConfigProvider) Merge(k string, v any) {
	panic("not implemented Merge")
}

func (c *defaultConfigProvider) WalkParams(walkFn func(params ...KeyParams) bool) {
	var walk func(params ...KeyParams)
	walk = func(params ...KeyParams) {
		if walkFn(params...) {
			return
		}
		p1 := params[len(params)-1]
		i := len(params)
		for k, v := range p1.Params {
			if p2, ok := v.(maps.Params); ok {
				paramsplus1 := make([]KeyParams, i+1)
				copy(paramsplus1, params)
				paramsplus1[i] = KeyParams{Key: k, Params: p2}
				walk(paramsplus1...)
			}
		}
	}
	walk(KeyParams{Key: "", Params: c.root})
}

func (c *defaultConfigProvider) getNestedKeyAndMap(key string, create bool) (string, maps.Params) {
	var parts []string
	v, ok := c.keyCache.Load(key)
	if ok {
		parts = v.([]string)
	} else {
		parts = strings.Split(key, ".")
		c.keyCache.Store(key, parts)
	}
	current := c.root
	for i := 0; i < len(parts)-1; i++ {
		next, found := current[parts[i]]
		if !found {
			if create {
				next = make(maps.Params)
				current[parts[i]] = next
			} else {
				return "", nil
			}
		}
		var ok bool
		current, ok = next.(maps.Params)
		if !ok {
			// E.g. a string, not a map that we can store values in.
			return "", nil
		}
	}
	return parts[len(parts)-1], current
}
