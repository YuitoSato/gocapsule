package externalwithignore

import "ignored"

func TestIgnoredPackage() {
	// When "ignored" package is in ignorePackages, these should NOT be violations
	// No "want" comment means no violation is expected

	// Struct literal creation - should be ignored
	_ = &ignored.IgnoredStruct{Value: "test"}

	// Using constructor is always OK
	s := ignored.NewIgnoredStruct("test")

	// Field assignment - should be ignored
	s.Value = "new"

	// Type conversion - should be ignored
	_ = ignored.IgnoredType("test")

	// Using constructor is always OK
	_ = ignored.NewIgnoredType("test")

	// Zero values - should be ignored
	_ = ignored.IgnoredStruct{}
	var z ignored.IgnoredStruct
	_ = z
	_ = new(ignored.IgnoredStruct)
	var t ignored.IgnoredType
	_ = t
}
