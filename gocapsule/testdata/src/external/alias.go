package external

import "target"

type aliasUser = target.User

type aliasEmail = target.Email

type aliasUserPtr = *target.User

func TestAlias() {
	// Violation: struct literal through an alias
	_ = &aliasUser{Name: "test"} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	_ = aliasUser{}              // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// Violation: zero values through an alias
	var u aliasUser // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead`
	_ = u
	_ = new(aliasUser) // want `zero value creation of User with new\(\) is not allowed; use target.NewUser\(\) instead`

	// Violation: type conversion through an alias
	_ = aliasEmail("test@example.com") // want `direct type conversion to Email is not allowed; use target.NewEmail\(\) instead`

	// Violation: field assignment through an alias of a pointer
	var p aliasUserPtr = target.NewUser("name", "email@test.com", 25)
	p.Name = "modified" // want `direct field assignment to User.Name is not allowed; User has a constructor NewUser\(\)`

	// Violation: field assignment through a pointer to an alias
	var pa *aliasUser = target.NewUser("name", "email@test.com", 25)
	pa.Name = "modified" // want `direct field assignment to User.Name is not allowed; User has a constructor NewUser\(\)`

	// OK: the zero value of a pointer is nil, not a User
	var q *aliasUser
	var r aliasUserPtr
	_, _ = q, r
}
