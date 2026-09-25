package gocapsule

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// exportConstructorFacts scans the current package for constructor functions
// (New or New*) and exports a fact for each type they return.
func exportConstructorFacts(pass *analysis.Pass, inspect *inspector.Inspector) {
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	// A type may have several constructors (e.g. New, NewUser, NewUserFromDTO).
	// Collect them all first so that exactly one fact is exported per type,
	// with a deterministic constructor name in diagnostics.
	constructors := map[*types.TypeName][]string{}
	var order []*types.TypeName

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)

		// Skip methods (only look for package-level functions)
		if funcDecl.Recv != nil {
			return
		}

		funcName := funcDecl.Name.Name

		// Check if function name matches the New / New* pattern
		if !isConstructorName(funcName) {
			return
		}

		// Find the return type
		returnType := getConstructorReturnType(pass, funcDecl)
		if returnType == nil {
			return
		}

		// Get the named type
		namedType := extractNamedType(returnType)
		if namedType == nil {
			return
		}

		// Verify the type is defined in the current package
		if namedType.Obj().Pkg() != pass.Pkg {
			return
		}

		obj := namedType.Obj()
		if _, seen := constructors[obj]; !seen {
			order = append(order, obj)
		}
		constructors[obj] = append(constructors[obj], funcName)
	})

	for _, obj := range order {
		pass.ExportObjectFact(obj, &EncapsulatedType{
			ConstructorName: preferredConstructorName(obj.Name(), constructors[obj]),
		})
	}
}

// isConstructorName reports whether a function name is a constructor name:
// exactly "New", or "New" followed by an uppercase letter ("NewUser", "NewRouter").
// The suffix does not need to match the returned type name.
func isConstructorName(name string) bool {
	if name == "New" {
		return true
	}
	if len(name) <= 3 {
		return false
	}
	if !strings.HasPrefix(name, "New") {
		return false
	}
	// The character after "New" must be uppercase
	return name[3] >= 'A' && name[3] <= 'Z'
}

// preferredConstructorName picks the constructor name shown in diagnostics when
// a type has several constructors: New<TypeName> first, then New, then the first
// one declared in the package.
func preferredConstructorName(typeName string, names []string) string {
	for _, n := range names {
		if strings.EqualFold(n, "New"+typeName) {
			return n
		}
	}
	for _, n := range names {
		if n == "New" {
			return n
		}
	}
	return names[0]
}

// getConstructorReturnType extracts the constructed type from a function declaration.
// Only the first result is considered, so (T, error) is treated as T.
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

	// Return the first result (additional results such as error are ignored)
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
