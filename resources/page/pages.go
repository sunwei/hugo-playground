package page

// Pages is a slice of Page objects. This is the most common list type in Hugo.
type Pages []Page

// Len returns the number of pages in the list.
func (p Pages) Len() int {
	return len(p)
}
