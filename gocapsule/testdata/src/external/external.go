package external

import "target"

func CreateUser() {
	// Violation: direct struct literal creation
	_ = &target.User{Name: "test"} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// Violation: unkeyed struct literal
	_ = target.User{} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// OK: using constructor
	user := target.NewUser("name", "email@test.com", 25)

	// Violation: field reassignment
	user.Name = "new name"   // want `direct field assignment to User.Name is not allowed; User has a constructor NewUser\(\)`
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
	_ = &target.Container{} // want `direct struct literal creation of Container is not allowed; use target.NewContainer\(\) instead`
}

func TestDefinedType() {
	// Violation: direct type conversion
	_ = target.Email("test@example.com") // want `direct type conversion to Email is not allowed; use target.NewEmail\(\) instead`

	// OK: using constructor
	_, _ = target.NewEmail("test@example.com")

	// OK: Token has no constructor, so direct type conversion is allowed
	_ = target.Token("abc123")
}
