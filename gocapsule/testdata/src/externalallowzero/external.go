package externalallowzero

import "target"

// OK: package-level zero value declaration
var globalUser target.User

func TestAllowZero() {
	// OK: zero values are allowed
	_ = target.User{}
	_ = &target.User{}
	_ = []target.User{{}}
	var u target.User
	_ = new(target.User)
	var e target.Email
	_ = e
	_ = new(target.Email)

	// OK: returned with a nil error
	_ = func() (target.User, error) {
		return target.User{}, nil
	}

	// Violation: non-empty struct literal, even with zero field values
	_ = &target.User{Name: "test"} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	_ = target.User{Name: ""}      // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// Violation: type conversion, even to the zero value
	_ = target.Email("") // want `direct type conversion to Email is not allowed; use target.NewEmail\(\) instead`

	// Violation: field assignment on a zero value
	u.Name = "test" // want `direct field assignment to User.Name is not allowed; User has a constructor NewUser\(\)`
}
