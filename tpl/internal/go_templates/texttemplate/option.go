package texttemplate

// missingKeyAction defines how to respond to indexing a map with a key that is not present.
type missingKeyAction int

type option struct {
	missingKey missingKeyAction
}

const (
	mapInvalid   missingKeyAction = iota // Return an invalid reflect.Value.
	mapZeroValue                         // Return the zero value for the map element.
	mapError                             // Error out
)
