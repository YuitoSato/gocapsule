package gocapsule

import "golang.org/x/tools/go/analysis"

// ignorePackages is a comma-separated list of package paths to ignore.
var ignorePackages string

// Analyzer is the gocapsule analyzer that enforces encapsulation.
var Analyzer = &analysis.Analyzer{
	Name: "gocapsule",
	Doc:  "enforces encapsulation by preventing direct struct creation, type conversion, and field reassignment when New** constructors exist",
	Run:  run,
	// No Requires: a fact-using analyzer and its Requires run on every
	// transitive dependency, and the x/tools checker (singlechecker,
	// checker.Analyze) retains their results until the whole run ends, so
	// requiring inspect.Analyzer would keep an inspector per dependency
	// resident. The syntax is walked with ast.Inspect instead.
	FactTypes: []analysis.Fact{new(EncapsulatedType)},
}

func init() {
	Analyzer.Flags.StringVar(&ignorePackages, "ignorePackages", "",
		"comma-separated list of package paths to ignore (e.g., net/http,database/sql)")
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Phase 1: Detect and export facts about types with New** constructors
	exportConstructorFacts(pass)

	// Phase 2: Detect violations (struct literals, type conversions, and field assignments)
	detectViolations(pass)

	return nil, nil
}
