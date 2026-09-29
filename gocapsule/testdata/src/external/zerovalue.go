package external

import (
	"errors"
	"fmt"

	"target"
)

// Violation: package-level zero value declaration
var globalUser target.User // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead`

// OK: pointer zero value is nil, not a User
var globalUserPtr *target.User

func TestZeroValue() {
	// Violation: empty struct literal
	_ = target.User{}  // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	_ = &target.User{} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// Violation: elided empty struct literal
	_ = []target.User{{}} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// Violation: var declaration without initializer
	var u target.User // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead`
	_ = u

	// Violation: one diagnostic per declared name
	var a, b target.User // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead` `zero value declaration of User is not allowed; use target.NewUser\(\) instead`
	_, _ = a, b

	// Violation: new
	_ = new(target.User) // want `zero value creation of User with new\(\) is not allowed; use target.NewUser\(\) instead`

	// Violation: zero values of defined types
	var e target.Email // want `zero value declaration of Email is not allowed; use target.NewEmail\(\) instead`
	_ = e
	_ = new(target.Email) // want `zero value creation of Email with new\(\) is not allowed; use target.NewEmail\(\) instead`

	// Violation: plain New() constructor
	var r target.Repository // want `zero value declaration of Repository is not allowed; use target.New\(\) instead`
	_ = r

	// OK: the zero value of a pointer is nil, not a User
	var p *target.User
	_ = p
	_ = new(*target.User)

	// OK: initialized declarations
	var v = target.NewUser("name", "email@test.com", 25)
	var w target.User = *target.NewUser("name", "email@test.com", 25)
	_, _ = v, w

	// OK: Config has no constructor
	var cfg target.Config
	_ = cfg
	_ = new(target.Config)
	_ = target.Config{}

	// OK: interfaces are not encapsulated, even if a constructor returns them
	var store target.Store
	_ = store
	_ = new(target.Store)
	_ = target.Store(fakeStore{})
}

type fakeStore struct{}

func (fakeStore) Get(string) string { return "" }

type notFoundError struct{}

func (*notFoundError) Error() string { return "not found" }

// OK: no facts are exported for sentinel errors and error helpers without
// -allowZeroWithNonNilError
var errNotFound = errors.New("not found")

func wrap(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("wrapped: %w", err)
}

func lookUp() (*target.User, error) {
	return nil, nil
}

// Violation: a non-nil error is not trusted without -allowZeroWithNonNilError
func ReturnZeroWithNonNilError(n int) (target.User, error) {
	_, err := lookUp()
	if err != nil {
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	if n == 0 {
		return target.User{}, errors.New("failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	if n == 1 {
		return target.User{}, errNotFound // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	if n == 2 {
		return target.User{}, wrap(target.ErrNotFound) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// Violation: a non-nil error is not trusted without -allowZeroWithNonNilError
func ReturnPointerZeroWithNonNilError(n int) (*target.User, error) {
	if n == 0 {
		return new(target.User), fmt.Errorf("failed: %d", n) // want `zero value creation of User with new\(\) is not allowed; use target.NewUser\(\) instead`
	}
	return &target.User{}, &notFoundError{} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// OK: the zero value of a pointer is nil, not a User
func ReturnNilWithNonNilError() (*target.User, error) {
	_, err := lookUp()
	if err != nil {
		return nil, err
	}
	return target.NewUser("name", "email@test.com", 25), nil
}

// Violation: a false ok is not trusted without -allowZeroWithFalseOk
func ReturnZeroWithFalseOk() (target.User, bool) {
	return target.User{}, false // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}
