package gocapsule

import (
	"maps"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ignorePackages is the set of package paths to ignore.
var ignorePackages packageSet

// allowZero allows external packages to create zero values of encapsulated
// types (T{}, &T{}, var v T, new(T)).
var allowZero bool

// allowZeroWithNonNilError allows zero values of encapsulated types returned
// together with an error result that is guaranteed to be non-nil.
var allowZeroWithNonNilError bool

// allowZeroWithFalseOk allows zero values of encapsulated types returned
// together with a last bool result that is guaranteed to be false, which is
// treated as the ok of the comma-ok idiom.
var allowZeroWithFalseOk bool

// Analyzer is the gocapsule analyzer that enforces encapsulation.
var Analyzer = &analysis.Analyzer{
	Name:      "gocapsule",
	Doc:       "enforces encapsulation by preventing direct struct creation, zero value creation, type conversion, and field reassignment when New** constructors exist",
	Run:       run,
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	FactTypes: []analysis.Fact{new(EncapsulatedType), new(NonNilError), new(NonNilResult)},
}

func init() {
	Analyzer.Flags.Var(&ignorePackages, "ignorePackages",
		"comma-separated `list` of package paths to ignore (e.g., net/http,database/sql)")
	Analyzer.Flags.BoolVar(&allowZero, "allowZero", false,
		"allow zero values of encapsulated types (T{}, &T{}, var v T, new(T))")
	Analyzer.Flags.BoolVar(&allowZeroWithNonNilError, "allowZeroWithNonNilError", false,
		"allow zero values of encapsulated types (T{}, &T{}, new(T)) returned with a non-nil error, e.g. return T{}, err inside if err != nil")
	Analyzer.Flags.BoolVar(&allowZeroWithFalseOk, "allowZeroWithFalseOk", false,
		"allow zero values of encapsulated types (T{}, &T{}, new(T)) returned with a false ok as the last result, e.g. return T{}, false")
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Phase 1: Detect and export facts about types with New** constructors,
	// and sentinel errors and functions that never return nil. The latter are
	// only used to allow zero values returned with a non-nil error, so they
	// are skipped otherwise, e.g. with -allowZero, which allows all zero values
	exportConstructorFacts(pass, inspect)
	if allowZeroWithNonNilError && !allowZero {
		exportNonNilFacts(pass, inspect)
	}

	// Phase 2: Detect violations (struct literals, zero values, type conversions, and field assignments)
	detectViolations(pass, inspect)

	return nil, nil
}

// packageSet is a set of package paths. As a flag, it is set from a
// comma-separated list.
type packageSet map[string]bool

// newPackageSet returns the set of the given package paths.
func newPackageSet(paths []string) packageSet {
	set := make(packageSet)
	for _, path := range paths {
		if path = strings.TrimSpace(path); path != "" {
			set[path] = true
		}
	}
	return set
}

// String implements flag.Value.
func (s *packageSet) String() string {
	return strings.Join(slices.Sorted(maps.Keys(*s)), ",")
}

// Set implements flag.Value. Like other flags, it replaces the previous value.
func (s *packageSet) Set(value string) error {
	*s = newPackageSet(strings.Split(value, ","))
	return nil
}
