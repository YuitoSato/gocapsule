package gocapsule

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// checkVarDecl checks if a var declaration without an initializer creates a
// zero value of an encapsulated type from an external package.
func checkVarDecl(pass *analysis.Pass, cur inspector.Cursor, spec *ast.ValueSpec) {
	if allowZero || spec.Type == nil || len(spec.Values) > 0 {
		return
	}

	if decl, ok := cur.Parent().Node().(*ast.GenDecl); !ok || decl.Tok != token.VAR {
		return
	}

	// Only the type itself is checked; the zero value of *T is nil, not a T
	namedType := namedOf(pass.TypesInfo.TypeOf(spec.Type))
	if namedType == nil {
		return
	}

	fact, ok := lookupEncapsulatedType(pass, namedType)
	if !ok {
		return
	}

	// Report violation for each declared name
	for _, name := range spec.Names {
		pass.Reportf(name.Pos(),
			"zero value declaration of %s is not allowed; use %s instead",
			namedType.Obj().Name(),
			constructorCall(namedType, fact),
		)
	}
}

// checkNewCall checks if new(T) creates a zero value of an encapsulated type
// from an external package.
func checkNewCall(pass *analysis.Pass, cur inspector.Cursor, call *ast.CallExpr) {
	if allowZero || len(call.Args) != 1 || !isBuiltinCall(pass, call, "new") {
		return
	}

	// Skip new(expr), which copies a value instead of creating a zero value
	tv, ok := pass.TypesInfo.Types[call.Args[0]]
	if !ok || !tv.IsType() {
		return
	}

	// Only the type itself is checked; new(*T) creates a nil *T, not a T
	namedType := namedOf(tv.Type)
	if namedType == nil {
		return
	}

	fact, ok := lookupEncapsulatedType(pass, namedType)
	if !ok || isReturnedWithNonNilError(pass, cur) {
		return
	}

	// Report violation
	pass.Reportf(call.Pos(),
		"zero value creation of %s with new() is not allowed; use %s instead",
		namedType.Obj().Name(),
		constructorCall(namedType, fact),
	)
}
