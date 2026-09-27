package gocapsule

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// detectViolations checks for struct literal creation, zero value creation,
// type conversion, and field assignment violations from external packages.
func detectViolations(pass *analysis.Pass, inspect *inspector.Inspector) {
	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.CallExpr)(nil),
		(*ast.ValueSpec)(nil),
	}

	for cur := range inspect.Root().Preorder(nodeFilter...) {
		switch node := cur.Node().(type) {
		case *ast.CompositeLit:
			checkCompositeLit(pass, cur, node)
		case *ast.AssignStmt:
			checkAssignment(pass, node)
		case *ast.CallExpr:
			checkTypeConversion(pass, node)
			checkNewCall(pass, cur, node)
		case *ast.ValueSpec:
			checkVarDecl(pass, cur, node)
		}
	}
}

// checkCompositeLit checks if a composite literal creates an encapsulated struct
// from an external package.
func checkCompositeLit(pass *analysis.Pass, cur inspector.Cursor, lit *ast.CompositeLit) {
	// Get the type of the composite literal
	tv, ok := pass.TypesInfo.Types[lit]
	if !ok {
		return
	}

	typ := tv.Type
	if typ == nil {
		return
	}

	// Extract the named type (handle pointers)
	namedType := extractNamedType(typ)
	if namedType == nil {
		return
	}

	fact, ok := lookupEncapsulatedType(pass, namedType)
	if !ok {
		return
	}

	// An empty literal (T{} or &T{}) is a zero value
	if len(lit.Elts) == 0 && (allowZero || isReturnedWithNonNilError(pass, cur)) {
		return
	}

	// Report violation
	pass.Reportf(lit.Pos(),
		"direct struct literal creation of %s is not allowed; use %s instead",
		namedType.Obj().Name(),
		constructorCall(namedType, fact),
	)
}

// checkTypeConversion checks if a type conversion creates an encapsulated
// defined type from an external package.
func checkTypeConversion(pass *analysis.Pass, call *ast.CallExpr) {
	// Type conversions look like function calls but the "function" is a type
	// e.g., Email("test") where Email is a defined type

	// Check if this is a type conversion (not a function call)
	tv, ok := pass.TypesInfo.Types[call.Fun]
	if !ok {
		return
	}

	// If it's not a type (IsType), it's a function call, not a type conversion
	if !tv.IsType() {
		return
	}

	// Get the named type
	namedType := extractNamedType(tv.Type)
	if namedType == nil {
		return
	}

	// Skip structs - they are handled by checkCompositeLit
	if _, ok := namedType.Underlying().(*types.Struct); ok {
		return
	}

	fact, ok := lookupEncapsulatedType(pass, namedType)
	if !ok {
		return
	}

	// Report violation
	pass.Reportf(call.Pos(),
		"direct type conversion to %s is not allowed; use %s instead",
		namedType.Obj().Name(),
		constructorCall(namedType, fact),
	)
}

// checkAssignment checks if an assignment modifies a field of an encapsulated
// struct from an external package.
func checkAssignment(pass *analysis.Pass, assign *ast.AssignStmt) {
	for _, lhs := range assign.Lhs {
		checkFieldAssignment(pass, lhs)
	}
}

// checkFieldAssignment recursively checks field assignments including chained access.
func checkFieldAssignment(pass *analysis.Pass, expr ast.Expr) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Get the selection information
	selection, ok := pass.TypesInfo.Selections[sel]
	if !ok {
		return
	}

	// We only care about field values, not method values
	if selection.Kind() != types.FieldVal {
		return
	}

	// Get the receiver type
	recvType := selection.Recv()
	if recvType == nil {
		return
	}

	// Check if this is a direct field access or chained through embedded fields
	namedType := findEncapsulatedType(pass, recvType, selection)
	if namedType == nil {
		return
	}

	fact, ok := lookupEncapsulatedType(pass, namedType)
	if !ok {
		return
	}

	// Report violation
	pass.Reportf(sel.Pos(),
		"direct field assignment to %s.%s is not allowed; %s has a constructor %s()",
		namedType.Obj().Name(),
		sel.Sel.Name,
		namedType.Obj().Name(),
		fact.ConstructorName,
	)
}

// findEncapsulatedType finds the encapsulated struct type from a selection.
// It handles both direct field access and access through embedded fields.
func findEncapsulatedType(pass *analysis.Pass, recvType types.Type, selection *types.Selection) *types.Named {
	// For direct field access (index length is 1)
	if len(selection.Index()) == 1 {
		return extractNamedStructTypeFromReceiver(recvType)
	}

	// For embedded field access, we need to find which struct the field belongs to
	// Walk through the selection index to find the actual struct containing the field
	currentType := recvType

	// The last index is the actual field, so we iterate up to len-1
	for i := 0; i < len(selection.Index())-1; i++ {
		named := extractNamedType(currentType)
		if named == nil {
			return nil
		}

		underlying, ok := named.Underlying().(*types.Struct)
		if !ok {
			return nil
		}

		// Get the next embedded field
		field := underlying.Field(selection.Index()[i])
		currentType = field.Type()
	}

	// Now currentType should be the struct containing the final field
	return extractNamedStructTypeFromReceiver(currentType)
}

// extractNamedStructTypeFromReceiver extracts the named type from a receiver type.
func extractNamedStructTypeFromReceiver(typ types.Type) *types.Named {
	named := extractNamedType(typ)
	if named == nil {
		return nil
	}

	if _, ok := named.Underlying().(*types.Struct); !ok {
		return nil
	}

	return named
}

// dereferencePointer removes pointer indirection from a type, looking
// through type aliases.
func dereferencePointer(typ types.Type) types.Type {
	if ptr, ok := types.Unalias(typ).(*types.Pointer); ok {
		return ptr.Elem()
	}
	return typ
}

// namedOf returns the named type of typ, looking through type aliases, or nil
// if typ is not a named type.
func namedOf(typ types.Type) *types.Named {
	named, _ := types.Unalias(typ).(*types.Named)
	return named
}

// lookupEncapsulatedType returns the EncapsulatedType fact of a named type
// that must be protected in the current package. It returns false for types
// defined in the current package, types in ignored packages, and types
// without a New** constructor.
func lookupEncapsulatedType(pass *analysis.Pass, named *types.Named) (EncapsulatedType, bool) {
	var fact EncapsulatedType

	// Skip if the type is defined in the current package
	if isLocalType(pass, named) {
		return fact, false
	}

	// Skip ignored packages
	if named.Obj().Pkg() != nil && ignorePackages[named.Obj().Pkg().Path()] {
		return fact, false
	}

	// Check if the type has an EncapsulatedType fact
	if !pass.ImportObjectFact(named.Obj(), &fact) {
		return fact, false // No constructor exists for this type
	}

	return fact, true
}

// isLocalType checks if a type is defined in the current package.
func isLocalType(pass *analysis.Pass, named *types.Named) bool {
	typePkg := named.Obj().Pkg()
	if typePkg == nil {
		return false // Universe scope types
	}

	// Handle test packages (e.g., "pkg" vs "pkg_test")
	currentPath := strings.TrimSuffix(pass.Pkg.Path(), "_test")
	typePath := strings.TrimSuffix(typePkg.Path(), "_test")

	return currentPath == typePath
}

// constructorCall returns the call of the constructor of a named type, e.g.
// "users.NewUser()".
func constructorCall(named *types.Named, fact EncapsulatedType) string {
	return named.Obj().Pkg().Name() + "." + fact.ConstructorName + "()"
}
