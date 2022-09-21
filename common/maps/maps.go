package maps

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/sunwei/hugo-playground/types"
)

// ToParamsAndPrepare converts in to Params and prepares it for use.
// If in is nil, an empty map is returned.
// See PrepareParams.
func ToParamsAndPrepare(in any) (Params, bool) {
	if types.IsNil(in) {
		return Params{}, true
	}
	m, err := ToStringMapE(in)
	if err != nil {
		return nil, false
	}
	PrepareParams(m)
	return m, true
}

// ToStringMapE converts in to map[string]interface{}.
func ToStringMapE(in any) (map[string]any, error) {
	switch vv := in.(type) {
	case Params:
		return vv, nil
	case map[string]string:
		var m = map[string]any{}
		for k, v := range vv {
			m[k] = v
		}
		return m, nil

	default:
		return cast.ToStringMapE(in)
	}
}

// MustToParamsAndPrepare calls ToParamsAndPrepare and panics if it fails.
func MustToParamsAndPrepare(in any) Params {
	if p, ok := ToParamsAndPrepare(in); ok {
		return p
	} else {
		panic(fmt.Sprintf("cannot convert %T to maps.Params", in))
	}
}

// ToStringMap converts in to map[string]interface{}.
func ToStringMap(in any) map[string]any {
	m, _ := ToStringMapE(in)
	return m
}

// ToStringMapBool converts in to bool.
func ToStringMapBool(in any) map[string]bool {
	m, _ := ToStringMapE(in)
	return cast.ToStringMapBool(m)
}

// ToStringMapString converts in to map[string]string.
func ToStringMapString(in any) map[string]string {
	m, _ := ToStringMapStringE(in)
	return m
}

// ToStringMapStringE converts in to map[string]string.
func ToStringMapStringE(in any) (map[string]string, error) {
	m, err := ToStringMapE(in)
	if err != nil {
		return nil, err
	}
	return cast.ToStringMapStringE(m)
}

// ToSliceStringMap converts in to []map[string]interface{}.
func ToSliceStringMap(in any) ([]map[string]any, error) {
	switch v := in.(type) {
	case []map[string]any:
		return v, nil
	case []any:
		var s []map[string]any
		for _, entry := range v {
			if vv, ok := entry.(map[string]any); ok {
				s = append(s, vv)
			}
		}
		return s, nil
	default:
		return nil, fmt.Errorf("unable to cast %#v of type %T to []map[string]interface{}", in, in)
	}
}
