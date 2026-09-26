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

// allowZeroPackages is the set of package paths whose types are allowed to be
// created as zero values.
var allowZeroPackages packageSet

// Analyzer is the gocapsule analyzer that enforces encapsulation.
var Analyzer = &analysis.Analyzer{
	Name:      "gocapsule",
	Doc:       "enforces encapsulation by preventing direct struct creation, zero value creation, type conversion, and field reassignment when New** constructors exist",
	Run:       run,
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	FactTypes: []analysis.Fact{new(EncapsulatedType), new(NonNilError)},
}

func init() {
	Analyzer.Flags.Var(&ignorePackages, "ignorePackages",
		"comma-separated `list` of package paths to ignore (e.g., net/http,database/sql)")
	Analyzer.Flags.BoolVar(&allowZero, "allowZero", false,
		"allow zero values of encapsulated types (T{}, &T{}, var v T, new(T))")
	Analyzer.Flags.Var(&allowZeroPackages, "allowZeroPackages",
		"comma-separated `list` of package paths whose types are allowed to be created as zero values (e.g., bytes,math/big)")
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Phase 1: Detect and export facts about types with New** constructors
	// and sentinel errors that are never nil
	exportConstructorFacts(pass, inspect)
	exportNonNilErrorFacts(pass, inspect)

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
