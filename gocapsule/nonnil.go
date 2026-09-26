package gocapsule

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

var (
	// errorType is the predeclared error type.
	errorType = types.Universe.Lookup("error").Type()
	// errorInterface is the interface of the error type.
	errorInterface = errorType.Underlying().(*types.Interface)
)

// exportNonNilErrorFacts exports facts for package-level error variables that
// are initialized with a non-nil value and never modified in the package,
// such as `var ErrNotFound = errors.New("not found")`.
func exportNonNilErrorFacts(pass *analysis.Pass, inspect *inspector.Inspector) {
	// Initializers are in dependency order, so `var ErrB = ErrA` sees the fact of ErrA
	var candidates []*types.Initializer
	for _, init := range pass.TypesInfo.InitOrder {
		if len(init.Lhs) == 1 && init.Lhs[0].Name() != "_" && implementsError(init.Lhs[0].Type()) {
			candidates = append(candidates, init)
		}
	}
	if len(candidates) == 0 {
		return
	}

	modified := modifiedPackageVars(pass, inspect)
	for _, init := range candidates {
		v := init.Lhs[0]
		if !modified[v] && isNonNilExpr(pass, init.Rhs, v.Type()) {
			pass.ExportObjectFact(v, new(NonNilError))
		}
	}
}

// modifiedPackageVars returns the package-level variables that are written
// anywhere in the current package.
func modifiedPackageVars(pass *analysis.Pass, inspect *inspector.Inspector) map[*types.Var]bool {
	modified := make(map[*types.Var]bool)
	for cur := range inspect.Root().Preorder(writeNodes...) {
		for _, expr := range writtenExprs(cur.Node()) {
			ident, ok := ast.Unparen(expr).(*ast.Ident)
			if !ok {
				continue
			}
			if v, ok := pass.TypesInfo.Uses[ident].(*types.Var); ok && v.Parent() == pass.Pkg.Scope() {
				modified[v] = true
			}
		}
	}
	return modified
}

// isReturnedWithNonNilError checks if the zero value expression at cur is
// returned together with an error that is guaranteed to be non-nil, e.g.
// `return &T{}, err` inside `if err != nil { ... }`. The caller cannot use the
// zero value without ignoring the error.
func isReturnedWithNonNilError(pass *analysis.Pass, cur inspector.Cursor) bool {
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

	for i := range ret.Results {
		if i == zeroIndex {
			continue
		}
		resultType := sig.Results().At(i).Type()
		if !implementsError(resultType) {
			continue
		}
		if isNonNilOperand(pass, retCur.ChildAt(edge.ReturnStmt_Results, i), resultType) {
			return true
		}
	}

	return false
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

// isNonNilOperand checks if the return operand at cur is guaranteed to be a
// non-nil value of resultType, taking nil checks before it into account.
func isNonNilOperand(pass *analysis.Pass, cur inspector.Cursor, resultType types.Type) bool {
	expr := ast.Unparen(cur.Node().(ast.Expr))
	if ident, ok := expr.(*ast.Ident); ok && isGuardedNonNil(pass, cur, ident) {
		return true
	}
	return isNonNilExpr(pass, expr, resultType)
}

// isNonNilExpr checks if expr is always a non-nil value of typ, regardless of
// control flow.
func isNonNilExpr(pass *analysis.Pass, expr ast.Expr, typ types.Type) bool {
	expr = ast.Unparen(expr)

	switch e := expr.(type) {
	case *ast.UnaryExpr:
		// &x is never nil
		if e.Op == token.AND {
			return true
		}
	case *ast.CallExpr:
		if isNonNilCall(pass, e) {
			return true
		}
	case *ast.Ident:
		if isNonNilVar(pass, e) {
			return true
		}
	case *ast.SelectorExpr:
		if isNonNilVar(pass, e.Sel) {
			return true
		}
	}

	// A value of a concrete type stored in an interface is never nil,
	// even if the value itself is a nil pointer
	if types.IsInterface(typ) {
		tv, ok := pass.TypesInfo.Types[expr]
		if ok && tv.Type != nil && !tv.IsNil() && !types.IsInterface(tv.Type) {
			return true
		}
	}

	return false
}

// isNonNilCall checks if a call always returns a non-nil value: new,
// errors.New, or fmt.Errorf.
func isNonNilCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	if isBuiltinCall(pass, call, "new") {
		return true
	}
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	return ok && (fn.FullName() == "errors.New" || fn.FullName() == "fmt.Errorf")
}

// isNonNilVar checks if ident refers to a package-level variable that is
// known to be non-nil, such as a sentinel error. Only package-level variables
// have the fact.
func isNonNilVar(pass *analysis.Pass, ident *ast.Ident) bool {
	v, ok := pass.TypesInfo.Uses[ident].(*types.Var)
	return ok && pass.ImportObjectFact(v, new(NonNilError))
}

// isGuardedNonNil checks if the identifier at cur refers to a local variable
// that is guaranteed to be non-nil by a nil check.
func isGuardedNonNil(pass *analysis.Pass, cur inspector.Cursor, ident *ast.Ident) bool {
	obj, ok := pass.TypesInfo.Uses[ident].(*types.Var)
	if !ok || !isLocalVar(obj) {
		return false
	}

	// Only interfaces and pointers: a method call on other types (e.g. a
	// slice type with a pointer receiver method) can implicitly take &obj
	switch obj.Type().Underlying().(type) {
	case *types.Interface, *types.Pointer:
	default:
		return false
	}

	return hasNilCheck(pass, cur, obj) && !isModifiableIndirectly(pass, cur, obj)
}

// hasNilCheck walks up from cur to the innermost function, looking for a nil
// check that guarantees obj != nil at cur, with no write to obj in between.
// The supported nil checks are:
//
//	if err != nil { <cur> }             // or the else of `if err == nil`
//	if err == nil { return ... }; <cur> // early exit
//	err = fmt.Errorf("...: %w", err); <cur>
//
// errors.Is(err, target) with a non-nil target and errors.As(err, ...) can
// be used in place of err != nil.
func hasNilCheck(pass *analysis.Pass, cur inspector.Cursor, obj types.Object) bool {
	for c := cur; ; c = c.Parent() {
		// c contains cur, so a write in c may run after the nil check
		if isWrittenIn(pass, c.Node(), obj) {
			return false
		}

		kind, index := c.ParentEdge()
		switch n := c.Parent().Node().(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return false
		case *ast.IfStmt:
			// The body runs when the condition is true, the else when it is false
			if (kind == edge.IfStmt_Body || kind == edge.IfStmt_Else) &&
				impliesNonNil(pass, n.Cond, obj, kind == edge.IfStmt_Body) {
				return true
			}
		case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause:
			if stmts := stmtList(n, kind); stmts != nil && hasPrecedingNilCheck(pass, stmts, index, obj) {
				return true
			}
		}
	}
}

// stmtList returns the statement list of n that kind refers to, or nil.
func stmtList(n ast.Node, kind edge.Kind) []ast.Stmt {
	switch kind {
	case edge.BlockStmt_List:
		return n.(*ast.BlockStmt).List
	case edge.CaseClause_Body:
		return n.(*ast.CaseClause).Body
	case edge.CommClause_Body:
		return n.(*ast.CommClause).Body
	}
	return nil
}

// hasPrecedingNilCheck checks if a statement before stmts[index], which
// contains cur, guarantees obj != nil with no write to obj after it: an if
// statement that exits early unless obj != nil, e.g.
// `if err == nil { return u, nil }`, or an assignment of a non-nil value.
func hasPrecedingNilCheck(pass *analysis.Pass, stmts []ast.Stmt, index int, obj types.Object) bool {
	// A goto may jump to a labeled statement, skipping the nil check
	if _, ok := stmts[index].(*ast.LabeledStmt); ok {
		return false
	}

	for _, stmt := range slices.Backward(stmts[:index]) {
		switch s := stmt.(type) {
		case *ast.IfStmt:
			if exitsEarly(pass, s.Body) && impliesNonNil(pass, s.Cond, obj, false) {
				return s.Else == nil || !isWrittenIn(pass, s.Else, obj)
			}
		case *ast.LabeledStmt:
			return false
		}
		if assignsNonNil(pass, stmt, obj) {
			return true
		}
		if isWrittenIn(pass, stmt, obj) {
			return false
		}
	}
	return false
}

// assignsNonNil checks if stmt assigns a non-nil value to obj, e.g.
// `err = fmt.Errorf("...: %w", err)`, `err := errors.New("...")`, or
// `var err error = &MyError{}`.
func assignsNonNil(pass *analysis.Pass, stmt ast.Stmt, obj types.Object) bool {
	var lhs, rhs []ast.Expr
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if s.Tok != token.ASSIGN && s.Tok != token.DEFINE {
			return false
		}
		lhs, rhs = s.Lhs, s.Rhs
	case *ast.DeclStmt:
		decl, ok := s.Decl.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR || len(decl.Specs) != 1 {
			return false
		}
		spec := decl.Specs[0].(*ast.ValueSpec)
		for _, name := range spec.Names {
			lhs = append(lhs, name)
		}
		rhs = spec.Values
	}

	// Only a single assignment: `a, err = f(), g()` is left unsupported
	if len(lhs) != 1 || len(rhs) != 1 {
		return false
	}
	ident, ok := ast.Unparen(lhs[0]).(*ast.Ident)
	return ok && pass.TypesInfo.ObjectOf(ident) == obj && isNonNilExpr(pass, rhs[0], obj.Type())
}

// exitsEarly checks if a statement never continues to the next statement: it
// ends with return, panic, break, or continue. goto is excluded, since it may
// jump forward to the next statement.
func exitsEarly(pass *analysis.Pass, stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return len(s.List) > 0 && exitsEarly(pass, s.List[len(s.List)-1])
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		return s.Tok == token.BREAK || s.Tok == token.CONTINUE
	case *ast.ExprStmt:
		call, ok := s.X.(*ast.CallExpr)
		return ok && isBuiltinCall(pass, call, "panic")
	case *ast.IfStmt:
		return s.Else != nil && exitsEarly(pass, s.Body) && exitsEarly(pass, s.Else)
	}
	return false
}

// impliesNonNil checks if cond evaluating to condValue guarantees obj != nil.
func impliesNonNil(pass *analysis.Pass, cond ast.Expr, obj types.Object, condValue bool) bool {
	switch e := ast.Unparen(cond).(type) {
	case *ast.CallExpr:
		return condValue && isErrorMatch(pass, e, obj)
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return impliesNonNil(pass, e.X, obj, !condValue)
		}
	case *ast.BinaryExpr:
		switch e.Op {
		case token.NEQ:
			return condValue && isNilComparison(pass, e, obj)
		case token.EQL:
			return !condValue && isNilComparison(pass, e, obj)
		case token.LAND:
			return condValue && (impliesNonNil(pass, e.X, obj, true) || impliesNonNil(pass, e.Y, obj, true))
		case token.LOR:
			return !condValue && (impliesNonNil(pass, e.X, obj, false) || impliesNonNil(pass, e.Y, obj, false))
		}
	}
	return false
}

// isErrorMatch checks if call is errors.As(obj, ...) or errors.Is(obj, target)
// with a non-nil target. Both return false when obj is nil.
func isErrorMatch(pass *analysis.Pass, call *ast.CallExpr, obj types.Object) bool {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok || len(call.Args) != 2 || !refersTo(pass, call.Args[0], obj) {
		return false
	}

	switch fn.FullName() {
	case "errors.As":
		return true
	case "errors.Is":
		// errors.Is(nil, nil) is true
		return isNonNilExpr(pass, call.Args[1], errorType)
	}
	return false
}

// isNilComparison checks if a binary expression compares obj with nil.
func isNilComparison(pass *analysis.Pass, e *ast.BinaryExpr, obj types.Object) bool {
	return (refersTo(pass, e.X, obj) && isNil(pass, e.Y)) ||
		(isNil(pass, e.X) && refersTo(pass, e.Y, obj))
}

// isModifiableIndirectly checks if obj can be written other than by a direct
// assignment in its own function: its address is taken, or a function
// literal writes to it.
func isModifiableIndirectly(pass *analysis.Pass, cur inspector.Cursor, obj types.Object) bool {
	// Local variables can only be captured within their top-level declaration
	var decl inspector.Cursor
	for c := range cur.Enclosing() {
		if _, ok := c.Parent().Node().(*ast.File); ok {
			decl = c
			break
		}
	}

	for c := range decl.Preorder(writeNodes...) {
		for _, expr := range writtenExprs(c.Node()) {
			if !refersTo(pass, expr, obj) {
				continue
			}
			if _, isAddr := c.Node().(*ast.UnaryExpr); isAddr || isInFuncLit(c) {
				return true
			}
		}
	}
	return false
}

// isInFuncLit checks if cur is inside a function literal.
func isInFuncLit(cur inspector.Cursor) bool {
	for range cur.Enclosing((*ast.FuncLit)(nil)) {
		return true
	}
	return false
}

// writeNodes are the node types that can write to a variable.
var writeNodes = []ast.Node{
	(*ast.AssignStmt)(nil),
	(*ast.RangeStmt)(nil),
	(*ast.UnaryExpr)(nil),
}

// writtenExprs returns the expressions that n may write to: the left-hand
// sides of an assignment, the key and value of `for k, v = range`, and the
// operand of &, which may be written through the pointer.
func writtenExprs(n ast.Node) []ast.Expr {
	switch n := n.(type) {
	case *ast.AssignStmt:
		return n.Lhs
	case *ast.RangeStmt:
		if n.Tok == token.ASSIGN {
			return []ast.Expr{n.Key, n.Value}
		}
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			return []ast.Expr{n.X}
		}
	}
	return nil
}

// isWrittenIn checks if obj is written anywhere within node.
func isWrittenIn(pass *analysis.Pass, node ast.Node, obj types.Object) bool {
	for n := range ast.Preorder(node) {
		for _, expr := range writtenExprs(n) {
			if refersTo(pass, expr, obj) {
				return true
			}
		}
	}
	return false
}

// refersTo checks if expr is an identifier that uses obj. The declaration of
// obj is not a use, but a redeclaration in a short variable declaration is.
func refersTo(pass *analysis.Pass, expr ast.Expr, obj types.Object) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && pass.TypesInfo.Uses[ident] == obj
}

// isNil checks if expr is the predeclared nil.
func isNil(pass *analysis.Pass, expr ast.Expr) bool {
	tv, ok := pass.TypesInfo.Types[expr]
	return ok && tv.IsNil()
}

// isBuiltinCall checks if call is a call to the named builtin function.
func isBuiltinCall(pass *analysis.Pass, call *ast.CallExpr, name string) bool {
	ident, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok || ident.Name != name {
		return false
	}
	_, ok = pass.TypesInfo.Uses[ident].(*types.Builtin)
	return ok
}

// isLocalVar checks if a variable is declared inside a function.
func isLocalVar(v *types.Var) bool {
	return !v.IsField() && v.Pkg() != nil && v.Parent() != nil && v.Parent() != v.Pkg().Scope()
}

// implementsError checks if a type implements the error interface.
func implementsError(typ types.Type) bool {
	return types.Implements(typ, errorInterface)
}
