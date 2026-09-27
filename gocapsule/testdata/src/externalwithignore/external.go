package externalwithignore

import (
	"ignored"
	"target"
)

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

func TestNotIgnoredPackage() {
	// Violation: "target" is not ignored, since package paths must match exactly
	_ = &target.User{Name: "test"} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}
