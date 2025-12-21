package gocapsule

// EncapsulatedStruct is a Fact indicating that a struct type has a
// corresponding New** constructor and should not be directly instantiated
// or have its fields reassigned from external packages.
type EncapsulatedStruct struct {
	ConstructorName string
}

// AFact implements the analysis.Fact interface.
func (*EncapsulatedStruct) AFact() {}
