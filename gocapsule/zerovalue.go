package gocapsule

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/edge"
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
	if !ok || isReturnedWithFailure(pass, cur) {
		return
	}

	// Report violation
	pass.Reportf(call.Pos(),
		"zero value creation of %s with new() is not allowed; use %s instead",
		namedType.Obj().Name(),
		constructorCall(namedType, fact),
	)
}

// isReturnedWithFailure checks if the zero value expression at cur is returned
// together with a result that tells the caller not to use it: an error that is
// guaranteed to be non-nil, e.g. `return &T{}, err` inside
// `if err != nil { ... }`, or, with -allowZeroWithFalseOk, a last bool result
// that is guaranteed to be false, e.g. `return T{}, false`.
func isReturnedWithFailure(pass *analysis.Pass, cur inspector.Cursor) bool {
	// Walk up through parentheses and & to the return operand
	for isParenOrAddr(cur.Parent().Node()) {
		cur = cur.Parent()
	}

	kind, zeroIndex := cur.ParentEdge()
	if kind != edge.ReturnStmt_Results {
		return false
	}
	retCur := cur.Parent()
	ret := retCur.Node().(*ast.ReturnStmt)

	sig := enclosingSignature(pass, retCur)
	if sig == nil || sig.Results().Len() != len(ret.Results) {
		return false
	}

	return isReturnedWithNonNilError(pass, retCur, sig, zeroIndex) ||
		allowZeroWithFalseOk && isReturnedWithFalseOk(pass, retCur, sig)
}

// isParenOrAddr checks if n is a parenthesized expression or &x.
func isParenOrAddr(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.ParenExpr:
		return true
	case *ast.UnaryExpr:
		return n.Op == token.AND
	}
	return false
}

// enclosingSignature returns the signature of the innermost function
// (declaration or literal) enclosing cur.
func enclosingSignature(pass *analysis.Pass, cur inspector.Cursor) *types.Signature {
	for c := range cur.Enclosing((*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)) {
		switch fn := c.Node().(type) {
		case *ast.FuncDecl:
			if obj, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func); ok {
				return obj.Signature()
			}
		case *ast.FuncLit:
			if sig, ok := pass.TypesInfo.TypeOf(fn).(*types.Signature); ok {
				return sig
			}
		}
		return nil
	}
	return nil
}
