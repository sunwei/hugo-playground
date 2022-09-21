package maps

import (
	"github.com/spf13/cast"
	"strings"
)

// Params is a map where all keys are lower case.
type Params map[string]any

// ParamsMergeStrategy tells what strategy to use in Params.Merge.
type ParamsMergeStrategy string

// Set overwrites values in p with values in pp for common or new keys.
// This is done recursively.
func (p Params) Set(pp Params) {
	for k, v := range pp {
		vv, found := p[k]
		if !found {
			p[k] = v
		} else {
			switch vvv := vv.(type) {
			case Params:
				if pv, ok := v.(Params); ok {
					vvv.Set(pv)
				} else {
					p[k] = v
				}
			default:
				p[k] = v
			}
		}
	}
}

// GetNestedParam gets the first match of the keyStr in the candidates given.
// It will first try the exact match and then try to find it as a nested map value,
// using the given separator, e.g. "mymap.name".
// It assumes that all the maps given have lower cased keys.
func GetNestedParam(keyStr, separator string, candidates ...Params) (any, error) {
	keyStr = strings.ToLower(keyStr)

	// Try exact match first
	for _, m := range candidates {
		if v, ok := m[keyStr]; ok {
			return v, nil
		}
	}

	keySegments := strings.Split(keyStr, separator)
	for _, m := range candidates {
		if v := m.Get(keySegments...); v != nil {
			return v, nil
		}
	}

	return nil, nil
}

// Get does a lower case and nested search in this map.
// It will return nil if none found.
func (p Params) Get(indices ...string) any {
	v, _, _ := getNested(p, indices)
	return v
}

func getNested(m map[string]any, indices []string) (any, string, map[string]any) {
	if len(indices) == 0 {
		return nil, "", nil
	}

	first := indices[0]
	v, found := m[strings.ToLower(cast.ToString(first))]
	if !found {
		if len(indices) == 1 {
			return nil, first, m
		}
		return nil, "", nil

	}

	if len(indices) == 1 {
		return v, first, m
	}

	switch m2 := v.(type) {
	case Params:
		return getNested(m2, indices[1:])
	case map[string]any:
		return getNested(m2, indices[1:])
	default:
		return nil, "", nil
	}
}

const (
	// ParamsMergeStrategyNone Do not merge.
	ParamsMergeStrategyNone ParamsMergeStrategy = "none"
	// ParamsMergeStrategyShallow Only add new keys.
	ParamsMergeStrategyShallow ParamsMergeStrategy = "shallow"
	// ParamsMergeStrategyDeep Add new keys, merge existing.
	ParamsMergeStrategyDeep ParamsMergeStrategy = "deep"

	mergeStrategyKey = "_merge"
)

func toMergeStrategy(v any) ParamsMergeStrategy {
	s := ParamsMergeStrategy(cast.ToString(v))
	switch s {
	case ParamsMergeStrategyDeep, ParamsMergeStrategyNone, ParamsMergeStrategyShallow:
		return s
	default:
		return ParamsMergeStrategyDeep
	}
}

// PrepareParams
// * makes all the keys in the given map lower cased and will do so
// * This will modify the map given.
// * Any nested map[interface{}]interface{}, map[string]interface{},map[string]string  will be converted to Params.
// * Any _merge value will be converted to proper type and value.
func PrepareParams(m Params) {
	for k, v := range m {
		var retyped bool
		lKey := strings.ToLower(k)
		if lKey == mergeStrategyKey {
			v = toMergeStrategy(v)
			retyped = true
		} else {
			switch vv := v.(type) {
			case map[any]any:
				var p Params = cast.ToStringMap(v)
				v = p
				PrepareParams(p)
				retyped = true
			case map[string]any:
				var p Params = v.(map[string]any)
				v = p
				PrepareParams(p)
				retyped = true
			case map[string]string:
				p := make(Params)
				for k, v := range vv {
					p[k] = v
				}
				v = p
				PrepareParams(p)
				retyped = true
			}
		}

		if retyped || k != lKey {
			delete(m, k)
			m[lKey] = v
		}
	}
}
