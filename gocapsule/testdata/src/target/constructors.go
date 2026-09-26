package target

import "bytes"

// HTTPClient is constructed by a constructor whose name matches case-insensitively
type HTTPClient struct { // want HTTPClient:`&\{NewHttpClient\}`
	URL string
}

// NewHttpClient creates a new HTTPClient
func NewHttpClient(url string) *HTTPClient {
	return &HTTPClient{URL: url}
}

// Admin has no constructor: the function name does not match the type name
type Admin struct {
	Name string
}

// NewSuperUser is not a constructor of Admin
func NewSuperUser(name string) *Admin {
	return &Admin{Name: name}
}

// Line has no constructor: "New" must be followed by an uppercase letter
type Line struct {
	Text string
}

// Newline is not a constructor of Line
func Newline() *Line {
	return &Line{Text: "\n"}
}

// Session has no constructor: methods are not constructors
type Session struct {
	ID string
}

// NewSession is a method, not a constructor
func (s *Session) NewSession() *Session {
	return &Session{ID: s.ID}
}

// Functions named New** that do not return a type of this package are not constructors
func NewNothing() {}

func NewCount() int {
	return 0
}

func NewBuffer() *bytes.Buffer {
	return bytes.NewBuffer(nil)
}
