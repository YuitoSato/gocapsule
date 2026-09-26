package externalallowzeropackages

import (
	"ignored"
	"target"
)

func TestAllowZeroPackages() {
	// OK: zero values of types in allowed packages
	_ = target.User{}
	_ = &target.User{}
	var u target.User
	_ = new(target.User)

	// Violation: other checks still apply to allowed packages
	_ = &target.User{Name: "test"} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	u.Name = "test"                // want `direct field assignment to User.Name is not allowed; User has a constructor NewUser\(\)`

	// Violation: zero values of types in other packages
	_ = ignored.IgnoredStruct{} // want `direct struct literal creation of IgnoredStruct is not allowed; use ignored.NewIgnoredStruct\(\) instead`
	var s ignored.IgnoredStruct // want `zero value declaration of IgnoredStruct is not allowed; use ignored.NewIgnoredStruct\(\) instead`
	_ = s
	_ = new(ignored.IgnoredStruct) // want `zero value creation of IgnoredStruct with new\(\) is not allowed; use ignored.NewIgnoredStruct\(\) instead`
}
