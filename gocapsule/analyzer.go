package gocapsule

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the gocapsule analyzer that enforces encapsulation.
var Analyzer = &analysis.Analyzer{
	Name:      "gocapsule",
	Doc:       "enforces encapsulation by preventing direct struct creation and field reassignment when New** constructors exist",
	Run:       run,
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	FactTypes: []analysis.Fact{new(EncapsulatedStruct)},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Phase 1: Detect and export facts about structs with New** constructors
	exportConstructorFacts(pass, inspect)

	// Phase 2: Detect violations (struct literals and field assignments)
	detectViolations(pass, inspect)

	// Phase 3: Check that constructors specify all struct fields
	checkConstructorCompleteness(pass, inspect)

	return nil, nil
}
