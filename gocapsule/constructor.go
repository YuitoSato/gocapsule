package gocapsule

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// exportConstructorFacts scans the current package for New** functions
// and exports facts for their corresponding struct types.
func exportConstructorFacts(pass *analysis.Pass, inspect *inspector.Inspector) {
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)

		// Skip methods (only look for package-level functions)
		if funcDecl.Recv != nil {
			return
		}

		funcName := funcDecl.Name.Name

		// Check if function name matches New** pattern
		if !isConstructorName(funcName) {
			return
		}

		// Extract the type name from the constructor name
		typeName := extractTypeName(funcName)
		if typeName == "" {
			return
		}

		// Find the return type
		returnType := getConstructorReturnType(pass, funcDecl)
		if returnType == nil {
			return
		}

		// Get the named type and verify it matches the expected type name
		namedType := extractNamedType(returnType)
		if namedType == nil {
			return
		}

		// Verify the type name matches
		if !strings.EqualFold(namedType.Obj().Name(), typeName) {
			return
		}

		// Verify the type is defined in the current package
		if namedType.Obj().Pkg() != pass.Pkg {
			return
		}

		// Export the fact for this type
		fact := &EncapsulatedType{
			ConstructorName: funcName,
		}
		pass.ExportObjectFact(namedType.Obj(), fact)
	})
}

// isConstructorName checks if a function name matches the New** pattern.
func isConstructorName(name string) bool {
	if len(name) <= 3 {
		return false
	}
	if !strings.HasPrefix(name, "New") {
		return false
	}
	// The character after "New" must be uppercase
	return name[3] >= 'A' && name[3] <= 'Z'
}

// extractTypeName extracts the type name from a constructor name.
// "NewUser" -> "User", "NewHTTPClient" -> "HTTPClient", "NewEmail" -> "Email"
func extractTypeName(constructorName string) string {
	if len(constructorName) <= 3 {
		return ""
	}
	return constructorName[3:]
}

// getConstructorReturnType extracts the return type from a function declaration.
func getConstructorReturnType(pass *analysis.Pass, funcDecl *ast.FuncDecl) types.Type {
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) == 0 {
		return nil
	}

	// Get the function's type information
	funcObj := pass.TypesInfo.Defs[funcDecl.Name]
	if funcObj == nil {
		return nil
	}

	funcType, ok := funcObj.Type().(*types.Signature)
	if !ok {
		return nil
	}

	results := funcType.Results()
	if results.Len() == 0 {
		return nil
	}

	// Return the first return value (ignore error returns)
	return results.At(0).Type()
}

// extractNamedType extracts the named type from a type.
// Handles both *T and T where T is a named type (struct or defined type).
func extractNamedType(typ types.Type) *types.Named {
	// Dereference pointer if necessary
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	// Get the named type
	named, ok := typ.(*types.Named)
	if !ok {
		return nil
	}

	return named
}
