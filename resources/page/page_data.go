package page

import "fmt"

// Data represents the .Data element in a Page in Hugo. We make this
// a type so we can do lazy loading of .Data.Pages
type Data map[string]any

// Pages returns the pages stored with key "pages". If this is a func,
// it will be invoked.
func (d Data) Pages() Pages {
	v, found := d["pages"]
	if !found {
		return nil
	}

	switch vv := v.(type) {
	case Pages:
		return vv
	case func() Pages:
		return vv()
	default:
		panic(fmt.Sprintf("%T is not Pages", v))
	}
}
