package ignored

// IgnoredStruct has a constructor but should be ignored when configured.
type IgnoredStruct struct { // want IgnoredStruct:`&{NewIgnoredStruct}`
	Value string
}

// NewIgnoredStruct creates a new IgnoredStruct.
func NewIgnoredStruct(value string) *IgnoredStruct {
	return &IgnoredStruct{Value: value}
}

// IgnoredType is a defined type with a constructor.
type IgnoredType string // want IgnoredType:`&{NewIgnoredType}`

// NewIgnoredType creates a new IgnoredType.
func NewIgnoredType(value string) IgnoredType {
	return IgnoredType(value)
}
