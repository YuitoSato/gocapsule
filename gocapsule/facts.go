package gocapsule

// EncapsulatedType is a Fact indicating that a type (struct or defined type) has a
// corresponding New** constructor and should not be directly instantiated
// or have its fields reassigned from external packages.
type EncapsulatedType struct {
	ConstructorName string
}

// AFact implements the analysis.Fact interface.
func (*EncapsulatedType) AFact() {}

// NonNilError is a Fact indicating that a package-level variable is
// initialized with a non-nil value and never modified in its package, such
// as a sentinel error `var ErrNotFound = errors.New("not found")`.
type NonNilError struct{}

// AFact implements the analysis.Fact interface.
func (*NonNilError) AFact() {}

func (*NonNilError) String() string { return "nonNilError" }
