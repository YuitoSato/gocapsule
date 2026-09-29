package target

import "errors"

// User is a struct with a constructor
type User struct { // want User:`&\{NewUser\}`
	Name  string
	Email string
	age   int // unexported field
}

// NewUser creates a new User
func NewUser(name, email string, age int) *User {
	return &User{
		Name:  name,
		Email: email,
		age:   age,
	}
}

// Config is a struct without a constructor (should be allowed)
type Config struct {
	Host string
	Port int
}

// Client has a constructor
type Client struct { // want Client:`&\{NewClient\}`
	Endpoint string
	Timeout  int
}

// NewClient creates a new Client
func NewClient(endpoint string) *Client {
	return &Client{Endpoint: endpoint, Timeout: 30}
}

// Container embeds User to test embedded field access
type Container struct { // want Container:`&\{NewContainer\}`
	User
	Extra string
}

// NewContainer creates a new Container
func NewContainer(user *User) *Container {
	return &Container{User: *user}
}

// InternalUsage shows that same-package usage is allowed
func InternalUsage() {
	// OK: same package can create structs directly
	_ = &User{Name: "internal"}

	user := NewUser("test", "test@test.com", 30)
	user.Name = "modified" // OK: same package can modify fields

	// OK: same package can use type conversion directly
	_ = Email("internal@test.com")

	// OK: same package can create zero values
	var u User
	_ = u
	_ = User{}
	_ = new(User)
	var e Email
	_ = e
}

// Email is a defined type with a constructor
type Email string // want Email:`&\{NewEmail\}`

// NewEmail creates a validated Email
func NewEmail(s string) (Email, error) {
	// In real code, this would validate the email format
	return Email(s), nil
}

// Token is a defined type without a constructor (should be allowed)
type Token string

// Repository is a struct constructed by a plain New() constructor
type Repository struct { // want Repository:`&\{New\}`
	Name string
}

// New creates a new Repository
func New(name string) *Repository {
	return &Repository{Name: name}
}

// ErrNotFound is a sentinel error. It has no fact without
// -allowZeroWithNonNilError; with it, externalnonnilerror checks its fact
var ErrNotFound = errors.New("not found")

// AppError is an error type with a constructor
type AppError struct { // want AppError:`&\{NewAppError\}`
	Code int
}

// NewAppError creates a new AppError
func NewAppError(code int) *AppError {
	return &AppError{Code: code}
}

func (e *AppError) Error() string { return "app error" }
