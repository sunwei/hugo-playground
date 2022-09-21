package config

import "github.com/sunwei/hugo-playground/common/maps"

// NewCompositeConfig creates a new composite Provider with a read-only base
// and a writeable layer.
func NewCompositeConfig(base, layer Provider) Provider {
	return &compositeConfig{
		base:  base,
		layer: layer,
	}
}

// compositeConfig contains a read only config base with
// a possibly writeable config layer on top.
type compositeConfig struct {
	base  Provider
	layer Provider
}

func (c *compositeConfig) GetBool(key string) bool {
	if c.layer.IsSet(key) {
		return c.layer.GetBool(key)
	}
	return c.base.GetBool(key)
}

func (c *compositeConfig) GetInt(key string) int {
	if c.layer.IsSet(key) {
		return c.layer.GetInt(key)
	}
	return c.base.GetInt(key)
}

func (c *compositeConfig) Merge(key string, value any) {
	c.layer.Merge(key, value)
}

func (c *compositeConfig) GetParams(key string) maps.Params {
	if c.layer.IsSet(key) {
		return c.layer.GetParams(key)
	}
	return c.base.GetParams(key)
}

func (c *compositeConfig) GetStringMap(key string) map[string]any {
	if c.layer.IsSet(key) {
		return c.layer.GetStringMap(key)
	}
	return c.base.GetStringMap(key)
}

func (c *compositeConfig) GetStringMapString(key string) map[string]string {
	if c.layer.IsSet(key) {
		return c.layer.GetStringMapString(key)
	}
	return c.base.GetStringMapString(key)
}

func (c *compositeConfig) GetStringSlice(key string) []string {
	if c.layer.IsSet(key) {
		return c.layer.GetStringSlice(key)
	}
	return c.base.GetStringSlice(key)
}

func (c *compositeConfig) Get(key string) any {
	if c.layer.IsSet(key) {
		return c.layer.Get(key)
	}
	return c.base.Get(key)
}

func (c *compositeConfig) IsSet(key string) bool {
	if c.layer.IsSet(key) {
		return true
	}
	return c.base.IsSet(key)
}

func (c *compositeConfig) GetString(key string) string {
	if c.layer.IsSet(key) {
		return c.layer.GetString(key)
	}
	return c.base.GetString(key)
}

func (c *compositeConfig) Set(key string, value any) {
	c.layer.Set(key, value)
}

func (c *compositeConfig) SetDefaults(params maps.Params) {
	c.layer.SetDefaults(params)
}

func (c *compositeConfig) WalkParams(walkFn func(params ...KeyParams) bool) {
	panic("not supported")
}

func (c *compositeConfig) SetDefaultMergeStrategy() {
	panic("not supported")
}
