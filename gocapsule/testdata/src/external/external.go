package external

import (
	"errors"

	"target"
)

func CreateUser() {
	// Violation: direct struct literal creation
	_ = &target.User{Name: "test"} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// OK: an empty literal is the zero value (see TestZeroValueLiteral)
	_ = target.User{}

	// OK: using constructor
	user := target.NewUser("name", "email@test.com", 25)

	// Violation: field reassignment
	user.Name = "new name"      // want `direct field assignment to User.Name is not allowed; User has a constructor NewUser\(\)`
	user.Email = "new@test.com" // want `direct field assignment to User.Email is not allowed; User has a constructor NewUser\(\)`

	// OK: Config has no constructor, so direct creation is allowed
	_ = &target.Config{Host: "localhost", Port: 8080}
	cfg := target.Config{}
	cfg.Host = "new" // OK: Config has no constructor

	// Violation: Client has a constructor
	_ = &target.Client{Endpoint: "http://api.example.com"} // want `direct struct literal creation of Client is not allowed; use target.NewClient\(\) instead`

	client := target.NewClient("http://api.example.com")
	client.Endpoint = "new endpoint" // want `direct field assignment to Client.Endpoint is not allowed; Client has a constructor NewClient\(\)`
}

func TestEmbeddedAccess() {
	container := target.NewContainer(target.NewUser("test", "test@test.com", 25))

	// Violation: accessing embedded field from external package
	container.Name = "modified" // want `direct field assignment to User.Name is not allowed; User has a constructor NewUser\(\)`

	// Violation: direct Container creation
	_ = &target.Container{Extra: "x"} // want `direct struct literal creation of Container is not allowed; use target.NewContainer\(\) instead`
}

func TestDefinedType() {
	// Violation: direct type conversion
	_ = target.Email("test@example.com") // want `direct type conversion to Email is not allowed; use target.NewEmail\(\) instead`

	// OK: using constructor
	_, _ = target.NewEmail("test@example.com")

	// OK: Token has no constructor, so direct type conversion is allowed
	_ = target.Token("abc123")
}

func TestPlainNewConstructor() {
	// Violation: direct struct literal creation
	_ = &target.Repository{Name: "test"} // want `direct struct literal creation of Repository is not allowed; use target.New\(\) instead`

	// OK: using constructor
	repo := target.New("test")

	// Violation: field reassignment
	repo.Name = "modified" // want `direct field assignment to Repository.Name is not allowed; Repository has a constructor New\(\)`
}

func TestZeroValueLiteral() {
	// OK: an empty literal is the zero value of the struct, exactly like
	// `var u target.User` or new(target.User), and cannot carry any data
	_ = target.User{}
	_ = &target.User{}
	_ = &target.Container{}
	_ = []target.User{{}}
	var zero target.User
	_ = zero
	_ = new(target.User)

	// OK: comparing against the zero value
	_ = zero == target.User{}

	// Violation: a literal that sets a field is a real construction
	_ = target.User{Name: "x"}     // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	_ = []target.User{{Name: "x"}} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// Violation: field assignment after a zero-value literal is still reported
	u := &target.User{}
	u.Name = "x" // want `direct field assignment to User.Name is not allowed; User has a constructor NewUser\(\)`

	// OK: an empty literal of an array type is its zero value too
	_ = target.UserID{}

	// Violation: an array literal with elements is a real construction
	_ = target.UserID{1} // want `direct struct literal creation of UserID is not allowed; use target.NewUserID\(\) instead`

	// Violation: an empty literal of a slice type is not its zero value (nil), so it is still reported
	_ = target.Roles{} // want `direct struct literal creation of Roles is not allowed; use target.NewRoles\(\) instead`
}

// saveUser mirrors the pattern reported in issue #7: a function returning
// (T, error) needs a placeholder T on the error path.
func saveUser(fail bool) (target.User, error) {
	if fail {
		// OK: the zero value is a placeholder; the error is the real result
		return target.User{}, errors.New("save failed")
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

func buildUser() (target.User, error) {
	// Violation: setting a field is a construction, even when returned alongside an error
	return target.User{Name: "x"}, errors.New("build failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}
