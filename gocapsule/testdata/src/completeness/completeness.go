package completeness

// --- Basic test: missing fields ---

type User struct { // want User:`&\{NewUser\}`
	Name    string
	email   string // private field
	Profile        // embedded field
}

type Profile struct {
	Age int
}

// NG: email and Profile are not specified
func NewUser(name string) *User {
	return &User{Name: name} // want `struct literal in constructor NewUser is missing fields: email, Profile`
}

// --- OK: all fields specified ---

type CompleteUser struct { // want CompleteUser:`&\{NewCompleteUser\}`
	Name    string
	email   string
	Profile Profile
}

func NewCompleteUser(name string, email string, profile Profile) *CompleteUser {
	return &CompleteUser{
		Name:    name,
		email:   email,
		Profile: profile,
	}
}

// --- Empty struct (OK) ---

type Empty struct{} // want Empty:`&\{NewEmpty\}`

func NewEmpty() *Empty {
	return &Empty{}
}

// --- Single field missing ---

type Config struct { // want Config:`&\{NewConfig\}`
	Host    string
	Port    int
	Timeout int
}

func NewConfig(host string, port int) *Config {
	return &Config{Host: host, Port: port} // want `struct literal in constructor NewConfig is missing fields: Timeout`
}

// --- Multiple return statements ---

type Result struct { // want Result:`&\{NewResult\}`
	Value   int
	IsValid bool
}

func NewResult(value int) *Result {
	if value < 0 {
		return &Result{Value: 0} // want `struct literal in constructor NewResult is missing fields: IsValid`
	}
	return &Result{Value: value, IsValid: true}
}

// --- Private field only missing ---

type Secret struct { // want Secret:`&\{NewSecret\}`
	ID     string
	secret string
}

func NewSecret(id string) *Secret {
	return &Secret{ID: id} // want `struct literal in constructor NewSecret is missing fields: secret`
}

// --- Embedded field only missing ---

type Embedded struct{}

type Wrapper struct { // want Wrapper:`&\{NewWrapper\}`
	Value string
	Embedded
}

func NewWrapper(value string) *Wrapper {
	return &Wrapper{Value: value} // want `struct literal in constructor NewWrapper is missing fields: Embedded`
}

// --- Non-pointer return (value type) ---

type ValueType struct { // want ValueType:`&\{NewValueType\}`
	X int
	Y int
}

func NewValueType(x int) ValueType {
	return ValueType{X: x} // want `struct literal in constructor NewValueType is missing fields: Y`
}

// --- All fields specified (OK) ---

type FullStruct struct { // want FullStruct:`&\{NewFullStruct\}`
	A string
	b int
	C
}

type C struct {
	D string
}

func NewFullStruct(a string, b int, c C) *FullStruct {
	return &FullStruct{
		A: a,
		b: b,
		C: c,
	}
}
