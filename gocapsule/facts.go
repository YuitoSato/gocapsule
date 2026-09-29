package gocapsule

import "fmt"

// EncapsulatedType is a Fact indicating that a type (struct or defined type,
// but not an interface) has a corresponding New** constructor and should not
// be directly instantiated or have its fields reassigned from external
// packages.
type EncapsulatedType struct {
	ConstructorName string
}

// AFact implements the analysis.Fact interface.
func (*EncapsulatedType) AFact() {}

// NonNilError is a Fact indicating that a package-level variable is
// initialized with a non-nil value and never set to a possibly nil value in
// its package, such as a sentinel error `var ErrNotFound = errors.New("not found")`.
type NonNilError struct{}

// AFact implements the analysis.Fact interface.
func (*NonNilError) AFact() {}

func (*NonNilError) String() string { return "nonNilError" }

// NonNilResult is a Fact indicating that a function returns a non-nil error
// whenever the arguments for the error parameters at Params are non-nil, such
// as `func Wrap(err error) error` that returns nil only if err is nil. A
// function with no such parameters always returns a non-nil error.
type NonNilResult struct {
	Params []int
}

// AFact implements the analysis.Fact interface.
func (*NonNilResult) AFact() {}

func (f *NonNilResult) String() string { return fmt.Sprintf("nonNilResult%v", f.Params) }
