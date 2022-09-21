package types

import (
	"fmt"
	"github.com/spf13/cast"
	"reflect"
)

// ToStringSlicePreserveString is the same as ToStringSlicePreserveStringE,
// but it never fails.
func ToStringSlicePreserveString(v any) []string {
	vv, _ := ToStringSlicePreserveStringE(v)
	return vv
}

// ToStringSlicePreserveStringE converts v to a string slice.
// If v is a string, it will be wrapped in a string slice.
func ToStringSlicePreserveStringE(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	if sds, ok := v.(string); ok {
		return []string{sds}, nil
	}
	result, err := cast.ToStringSliceE(v)
	if err == nil {
		return result, nil
	}

	// Probably []int or similar. Fall back to reflect.
	vv := reflect.ValueOf(v)

	switch vv.Kind() {
	case reflect.Slice, reflect.Array:
		result = make([]string, vv.Len())
		for i := 0; i < vv.Len(); i++ {
			s, err := cast.ToStringE(vv.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			result[i] = s
		}
		return result, nil
	default:
		return nil, fmt.Errorf("failed to convert %T to a string slice", v)
	}

}
