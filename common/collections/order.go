package collections

type Order interface {
	// Ordinal is a zero-based ordinal that represents the order of an object
	// in a collection.
	Ordinal() int
}
