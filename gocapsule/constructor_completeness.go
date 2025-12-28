package gocapsule

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// checkConstructorCompleteness checks that struct literals in New** constructors
// explicitly specify all fields.
func checkConstructorCompleteness(pass *analysis.Pass, inspect *inspector.Inspector) {
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

		// Get the return type
		returnType := getConstructorReturnType(pass, funcDecl)
		if returnType == nil {
			return
		}

		// Get the named struct type
		namedType := extractNamedStructType(returnType)
		if namedType == nil {
			return
		}

		// Only check constructors in the same package as the struct
		if namedType.Obj().Pkg() != pass.Pkg {
			return
		}

		// Verify the struct name matches the constructor name
		structName := extractStructName(funcName)
		if !strings.EqualFold(namedType.Obj().Name(), structName) {
			return
		}

		// Get the underlying struct type
		structType, ok := namedType.Underlying().(*types.Struct)
		if !ok {
			return
		}

		// Collect all field names
		allFields := collectStructFields(structType)
		if len(allFields) == 0 {
			return // No fields to check
		}

		// Walk the function body to find return statements
		ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
			retStmt, ok := node.(*ast.ReturnStmt)
			if !ok {
				return true
			}

			for _, result := range retStmt.Results {
				checkReturnExpr(pass, result, namedType, allFields, funcName)
			}
			return true
		})
	})
}

// checkReturnExpr checks if a return expression is a struct literal with all fields specified.
func checkReturnExpr(pass *analysis.Pass, expr ast.Expr, expectedType *types.Named, allFields []string, funcName string) {
	// Handle &Struct{} case
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op.String() == "&" {
		expr = unary.X
	}

	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return
	}

	// Get the type of the composite literal
	tv, ok := pass.TypesInfo.Types[lit]
	if !ok {
		return
	}

	// Check if the type matches the expected type
	litNamedType := extractNamedStructType(tv.Type)
	if litNamedType == nil || litNamedType.Obj() != expectedType.Obj() {
		return
	}

	// Get specified fields
	specifiedFields := getSpecifiedFields(lit)

	// Find missing fields
	var missingFields []string
	for _, field := range allFields {
		if !specifiedFields[field] {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		pass.Reportf(lit.Pos(),
			"struct literal in constructor %s is missing fields: %s",
			funcName,
			strings.Join(missingFields, ", "),
		)
	}
}

// collectStructFields collects all field names from a struct type.
func collectStructFields(structType *types.Struct) []string {
	var fields []string
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		fields = append(fields, field.Name())
	}
	return fields
}

// getSpecifiedFields returns a map of field names specified in a composite literal.
func getSpecifiedFields(lit *ast.CompositeLit) map[string]bool {
	specified := make(map[string]bool)
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok {
				specified[ident.Name] = true
			}
		}
	}
	return specified
}
