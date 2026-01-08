package gocapsule

// EncapsulatedType is a Fact indicating that a type (struct or defined type) has a
// corresponding New** constructor and should not be directly instantiated
// or have its fields reassigned from external packages.
type EncapsulatedType struct {
	ConstructorName string
}

// AFact implements the analysis.Fact interface.
func (*EncapsulatedType) AFact() {}
