package gocapsule

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// detectViolations checks for struct literal creation and field assignment
// violations from external packages.
func detectViolations(pass *analysis.Pass, inspect *inspector.Inspector) {
	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.CompositeLit:
			checkCompositeLit(pass, node)
		case *ast.AssignStmt:
			checkAssignment(pass, node)
		case *ast.CallExpr:
			checkTypeConversion(pass, node)
		}
	})
}

// checkCompositeLit checks if a composite literal creates an encapsulated struct
// from an external package.
func checkCompositeLit(pass *analysis.Pass, lit *ast.CompositeLit) {
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

	// Skip if the struct is defined in the current package
	if isLocalType(pass, namedType) {
		return
	}

	// Skip ignored packages
	if namedType.Obj().Pkg() != nil && shouldIgnorePackage(namedType.Obj().Pkg().Path()) {
		return
	}

	// Check if the struct has an EncapsulatedType fact
	var fact EncapsulatedType
	if !pass.ImportObjectFact(namedType.Obj(), &fact) {
		return // No constructor exists for this struct
	}

	// Report violation
	pass.Reportf(lit.Pos(),
		"direct struct literal creation of %s is not allowed; use %s.%s() instead",
		namedType.Obj().Name(),
		namedType.Obj().Pkg().Name(),
		fact.ConstructorName,
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

	// Skip if the type is defined in the current package
	if isLocalType(pass, namedType) {
		return
	}

	// Skip ignored packages
	if namedType.Obj().Pkg() != nil && shouldIgnorePackage(namedType.Obj().Pkg().Path()) {
		return
	}

	// Check if the type has an EncapsulatedType fact
	var fact EncapsulatedType
	if !pass.ImportObjectFact(namedType.Obj(), &fact) {
		return // No constructor exists for this type
	}

	// Report violation
	pass.Reportf(call.Pos(),
		"direct type conversion to %s is not allowed; use %s.%s() instead",
		namedType.Obj().Name(),
		namedType.Obj().Pkg().Name(),
		fact.ConstructorName,
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

	// Skip if the struct is defined in the current package
	if isLocalType(pass, namedType) {
		return
	}

	// Skip ignored packages
	if namedType.Obj().Pkg() != nil && shouldIgnorePackage(namedType.Obj().Pkg().Path()) {
		return
	}

	// Check if the struct has an EncapsulatedType fact
	var fact EncapsulatedType
	if !pass.ImportObjectFact(namedType.Obj(), &fact) {
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
		currentType = dereferencePointer(currentType)

		named, ok := currentType.(*types.Named)
		if !ok {
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
	typ = dereferencePointer(typ)

	named, ok := typ.(*types.Named)
	if !ok {
		return nil
	}

	if _, ok := named.Underlying().(*types.Struct); !ok {
		return nil
	}

	return named
}

// dereferencePointer removes pointer indirection from a type.
func dereferencePointer(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return typ
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

// shouldIgnorePackage checks if a package path is in the ignore list.
func shouldIgnorePackage(pkgPath string) bool {
	if ignorePackages == "" {
		return false
	}
	for _, ignored := range strings.Split(ignorePackages, ",") {
		if strings.TrimSpace(ignored) == pkgPath {
			return true
		}
	}
	return false
}
