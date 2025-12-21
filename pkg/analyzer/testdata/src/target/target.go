package target

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
}
