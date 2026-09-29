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

// exportNonNilFacts exports facts for package-level error variables that are
// never nil and functions that return non-nil errors. They may depend on each
// other in any order,
// e.g. `var ErrX = newError("x")` and `func notFound() error { return
// ErrNotFound }`, so this repeats until no new fact is found.
func exportNonNilFacts(pass *analysis.Pass, inspect *inspector.Inspector) {
	vars := nonNilErrorCandidates(pass)
	funcs := nonNilResultCandidates(pass, inspect)
	log := &factLog{pass: pass, order: make(map[types.Object]int)}
	varsChecked := -1
	for {
		n := log.len()
		// Variables depend on facts through their initializers and assignments
		// anywhere in the package, so they are checked again after any new fact
		if varsChecked < log.len() {
			varsChecked = log.len()
			vars = exportNonNilErrorFacts(pass, inspect, log, vars)
		}
		funcs = exportNonNilResultFacts(pass, log, funcs)
		if log.len() == n {
			break
		}
	}
}

// factLog records the order in which facts are exported in the current
// package, so that a candidate is checked again only after a fact it depends
// on is exported.
type factLog struct {
	pass  *analysis.Pass
	order map[types.Object]int
}

// export exports fact for obj.
func (l *factLog) export(obj types.Object, fact analysis.Fact) {
	l.pass.ExportObjectFact(obj, fact)
	l.order[obj] = len(l.order)
}

// len returns the number of exported facts.
func (l *factLog) len() int {
	return len(l.order)
}

// exportedSince reports whether a fact for one of objs was exported after
// the first n facts.
func (l *factLog) exportedSince(objs []types.Object, n int) bool {
	for _, obj := range objs {
		if i, ok := l.order[obj]; ok && i >= n {
			return true
		}
	}
	return false
}

// nonNilErrorCandidates returns the initializers of single package-level
// error variables, in dependency order.
func nonNilErrorCandidates(pass *analysis.Pass) []*types.Initializer {
	var candidates []*types.Initializer
	for _, init := range pass.TypesInfo.InitOrder {
		if len(init.Lhs) == 1 && init.Lhs[0].Name() != "_" && implementsError(init.Lhs[0].Type()) {
			candidates = append(candidates, init)
		}
	}
	return candidates
}

// exportNonNilErrorFacts exports facts for the candidate variables that are
// initialized with a non-nil value and never set to a possibly nil value in
// the package, such as `var ErrNotFound = errors.New("not found")`, and
// returns the remaining candidates.
func exportNonNilErrorFacts(pass *analysis.Pass, inspect *inspector.Inspector, log *factLog, candidates []*types.Initializer) []*types.Initializer {
	if len(candidates) == 0 {
		return nil
	}

	// Initializers are in dependency order, so `var ErrB = ErrA` sees the fact of ErrA
	nilable := nilablePackageVars(pass, inspect)
	var remaining []*types.Initializer
	for _, init := range candidates {
		v := init.Lhs[0]
		if !nilable[v] && isNonNilExpr(pass, init.Rhs, v.Type()) {
			log.export(v, new(NonNilError))
		} else {
			remaining = append(remaining, init)
		}
	}
	return remaining
}

// nilablePackageVars returns the package-level variables that may be set to
// nil anywhere in the current package.
func nilablePackageVars(pass *analysis.Pass, inspect *inspector.Inspector) map[*types.Var]bool {
	nilable := make(map[*types.Var]bool)
	for cur := range inspect.Root().Preorder(writeNodes...) {
		for _, expr := range writtenExprs(cur.Node()) {
			ident, ok := ast.Unparen(expr).(*ast.Ident)
			if !ok {
				continue
			}
			v, ok := pass.TypesInfo.Uses[ident].(*types.Var)
			if ok && v.Parent() == pass.Pkg.Scope() && !assigns(pass, cur.Node(), v, nonNilGuarantee) {
				nilable[v] = true
			}
		}
	}
	return nilable
}

// isReturnedWithNonNilError checks if an error result of the return statement
// at retCur, other than the zero value at zeroIndex, is guaranteed to be
// non-nil, e.g. `return &T{}, err` inside `if err != nil { ... }`. With
// -allowZeroWithNonNilError, the zero value is then allowed, since the caller
// cannot use it without ignoring the error.
func isReturnedWithNonNilError(pass *analysis.Pass, retCur inspector.Cursor, sig *types.Signature, zeroIndex int) bool {
	for i := range sig.Results().Len() {
		if i == zeroIndex {
			continue
		}
		result := sig.Results().At(i)
		if !implementsError(result.Type()) {
			continue
		}
		// A deferred call may set a named result to nil after the return,
		// e.g. `defer func() { err = nil }()`
		if result.Name() != "" && mayBreakIndirectly(pass, retCur, result, nonNilGuarantee) {
			continue
		}
		if isNonNilOperand(pass, retCur.ChildAt(edge.ReturnStmt_Results, i), result.Type()) {
			return true
		}
	}

	return false
}

// isNonNilOperand checks if the operand at cur is guaranteed to be a non-nil
// value of typ, taking nil checks before it into account.
func isNonNilOperand(pass *analysis.Pass, cur inspector.Cursor, typ types.Type) bool {
	cur = unparenCursor(cur)
	switch e := cur.Node().(type) {
	case *ast.Ident:
		if isGuarded(pass, cur, e, nonNilGuarantee) {
			return true
		}
	case *ast.CallExpr:
		// The arguments may be guaranteed to be non-nil by nil checks, e.g.
		// `if err != nil { return T{}, Wrap(err) }`
		if isNonNilIfArgs(pass, e, func(i int, typ types.Type) bool {
			return isNonNilOperand(pass, cur.ChildAt(edge.CallExpr_Args, i), typ)
		}) {
			return true
		}
	}
	return isNonNilExpr(pass, cur.Node().(ast.Expr), typ)
}

// unparenCursor returns the cursor of the expression at cur without
// parentheses.
func unparenCursor(cur inspector.Cursor) inspector.Cursor {
	for {
		if _, ok := cur.Node().(*ast.ParenExpr); !ok {
			return cur
		}
		cur = cur.ChildAt(edge.ParenExpr_X, -1)
	}
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
		if isNonNilCall(pass, e) || isNonNilIfArgs(pass, e, func(i int, typ types.Type) bool {
			return isNonNilExpr(pass, e.Args[i], typ)
		}) {
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

// isNonNilIfArgs checks if call returns a non-nil value provided that some of
// its arguments are non-nil, which isNonNilArg checks given the index and the
// type of each such argument: a conversion to an interface or pointer type,
// e.g. `error(&MyError{})`, or a call to a function with the NonNilResult
// fact, whose error arguments at the Params of the fact must be non-nil.
func isNonNilIfArgs(pass *analysis.Pass, call *ast.CallExpr, isNonNilArg func(i int, typ types.Type) bool) bool {
	if tv, ok := pass.TypesInfo.Types[call.Fun]; ok && tv.IsType() {
		if _, ok := tv.Type.(*types.TypeParam); ok || len(call.Args) != 1 {
			return false
		}
		switch tv.Type.Underlying().(type) {
		case *types.Interface, *types.Pointer:
			return isNonNilArg(0, tv.Type)
		}
		return false
	}

	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	fact := new(NonNilResult)
	if !ok || !pass.ImportObjectFact(fn.Origin(), fact) {
		return false
	}
	sig, ok := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
	if !ok {
		return false
	}
	// f(g()) passes the results of g as the arguments
	if len(call.Args) == 1 {
		if _, ok := pass.TypesInfo.TypeOf(call.Args[0]).(*types.Tuple); ok {
			return false
		}
	}

	// A method expression such as (*T).Wrap takes the receiver as the first
	// argument. The parameters of the fact are error parameters, so none of
	// them is variadic, and each has its own argument.
	offset := sig.Params().Len() - fn.Signature().Params().Len()
	for _, i := range fact.Params {
		if !isNonNilArg(i+offset, sig.Params().At(i+offset).Type()) {
			return false
		}
	}
	return true
}

// isNonNilVar checks if ident refers to a package-level variable that is
// known to be non-nil, such as a sentinel error. Only package-level variables
// have the fact.
func isNonNilVar(pass *analysis.Pass, ident *ast.Ident) bool {
	v, ok := pass.TypesInfo.Uses[ident].(*types.Var)
	return ok && pass.ImportObjectFact(v, new(NonNilError))
}

// guarantee is a property of a local variable that checks and assignments
// before a point can guarantee at that point, such as err != nil inside
// `if err != nil { ... }`.
type guarantee struct {
	// tracked checks if a variable of typ can be tracked. A method call can
	// implicitly take the address of a variable of some types, e.g. a slice
	// type with a pointer receiver method.
	tracked func(typ types.Type) bool
	// implied checks if cond evaluating to condValue guarantees the property
	// for obj.
	implied func(pass *analysis.Pass, cond ast.Expr, obj types.Object, condValue bool) bool
	// holds checks if expr is always a value of typ with the property.
	holds func(pass *analysis.Pass, expr ast.Expr, typ types.Type) bool
	// zero is whether the zero value has the property.
	zero bool
}

// nonNilGuarantee guarantees that a variable is non-nil. Only interfaces and
// pointers are tracked.
var nonNilGuarantee = guarantee{
	tracked: isNilable,
	implied: impliesNonNil,
	holds:   isNonNilExpr,
}

// isGuarded checks if the identifier at cur refers to a local variable that
// is guaranteed to have the property of g by a check, e.g. a nil check.
func isGuarded(pass *analysis.Pass, cur inspector.Cursor, ident *ast.Ident, g guarantee) bool {
	obj, ok := pass.TypesInfo.Uses[ident].(*types.Var)
	if !ok || !isLocalVar(obj) || !g.tracked(obj.Type()) {
		return false
	}
	return hasCheck(pass, cur, obj, g) && !mayBreakIndirectly(pass, cur, obj, g)
}

// hasCheck walks up from cur to the innermost function, looking for a check
// that guarantees the property of g for obj at cur, with no write of a value
// that may not have it in between.
// For nonNilGuarantee, the supported checks are:
//
//	if err != nil { <cur> }             // or the else of `if err == nil`
//	if err == nil { return ... }; <cur> // early exit
//	err = fmt.Errorf("...: %w", err); <cur>
//
// errors.Is(err, target) with a non-nil target and errors.As(err, ...) can
// be used in place of err != nil. The checks for other properties have the
// same forms, e.g. `if !ok { <cur> }` for falseGuarantee.
func hasCheck(pass *analysis.Pass, cur inspector.Cursor, obj types.Object, g guarantee) bool {
	for c := cur; ; c = c.Parent() {
		// c contains cur, so a write in c may run after the check
		if mayBreakIn(pass, c.Node(), obj, g) {
			return false
		}

		kind, index := c.ParentEdge()
		switch n := c.Parent().Node().(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return false
		case *ast.IfStmt:
			// The body runs when the condition is true, the else when it is false
			if (kind == edge.IfStmt_Body || kind == edge.IfStmt_Else) &&
				g.implied(pass, n.Cond, obj, kind == edge.IfStmt_Body) {
				return true
			}
		case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause:
			if stmts := stmtList(n, kind); stmts != nil && hasPrecedingCheck(pass, stmts, index, obj, g) {
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

// hasPrecedingCheck checks if a statement before stmts[index], which contains
// cur, guarantees the property of g for obj with no write of a value that may
// not have it after it: an if statement that exits early unless obj has it,
// e.g. `if err == nil { return u, nil }`, or an assignment of a value with it.
func hasPrecedingCheck(pass *analysis.Pass, stmts []ast.Stmt, index int, obj types.Object, g guarantee) bool {
	// A goto may jump to a labeled statement, skipping the check
	if _, ok := stmts[index].(*ast.LabeledStmt); ok {
		return false
	}

	for _, stmt := range slices.Backward(stmts[:index]) {
		switch s := stmt.(type) {
		case *ast.IfStmt:
			if exitsEarly(pass, s.Body) && g.implied(pass, s.Cond, obj, false) {
				return s.Else == nil || !mayBreakIn(pass, s.Else, obj, g)
			}
		case *ast.LabeledStmt:
			return false
		}
		if assigns(pass, stmt, obj, g) {
			return true
		}
		if mayBreakIn(pass, stmt, obj, g) {
			return false
		}
	}
	return false
}

// assigns checks if n assigns a value with the property of g to obj, e.g.
// `err = fmt.Errorf("...: %w", err)`, `err := errors.New("...")`, or
// `var err error = &MyError{}` for nonNilGuarantee, and `var ok bool` for
// falseGuarantee.
func assigns(pass *analysis.Pass, n ast.Node, obj types.Object, g guarantee) bool {
	var lhs, rhs []ast.Expr
	switch s := n.(type) {
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
		// Without values, the variables are set to the zero value
		if len(spec.Values) == 0 {
			return g.zero && slices.ContainsFunc(spec.Names, func(name *ast.Ident) bool {
				return pass.TypesInfo.Defs[name] == obj
			})
		}
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
	return ok && pass.TypesInfo.ObjectOf(ident) == obj && g.holds(pass, rhs[0], obj.Type())
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

// impliesNil checks if cond evaluating to condValue guarantees obj == nil.
func impliesNil(pass *analysis.Pass, cond ast.Expr, obj types.Object, condValue bool) bool {
	switch e := ast.Unparen(cond).(type) {
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return impliesNil(pass, e.X, obj, !condValue)
		}
	case *ast.BinaryExpr:
		switch e.Op {
		case token.EQL:
			return condValue && isNilComparison(pass, e, obj)
		case token.NEQ:
			return !condValue && isNilComparison(pass, e, obj)
		case token.LAND:
			return condValue && (impliesNil(pass, e.X, obj, true) || impliesNil(pass, e.Y, obj, true))
		case token.LOR:
			return !condValue && (impliesNil(pass, e.X, obj, false) || impliesNil(pass, e.Y, obj, false))
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

// mayBreakIndirectly checks if obj may be set to a value without the property
// of g other than by a direct assignment in the function that declares it:
// its address is taken, or a function literal nested in that function writes
// a value that may not have it to obj.
func mayBreakIndirectly(pass *analysis.Pass, cur inspector.Cursor, obj types.Object, g guarantee) bool {
	// Local variables can only be captured within their top-level declaration
	var decl inspector.Cursor
	for c := range cur.Enclosing() {
		if _, ok := c.Parent().Node().(*ast.File); ok {
			decl = c
			break
		}
	}

	for c := range decl.Preorder(writeNodes...) {
		if !mayBreak(pass, c.Node(), obj, g) {
			continue
		}
		if _, isAddr := c.Node().(*ast.UnaryExpr); isAddr || isInNestedFunc(c, obj) {
			return true
		}
	}
	return false
}

// isInNestedFunc checks if cur is inside a function literal nested in the
// function that declares obj, which may run at any time after it is created.
func isInNestedFunc(cur inspector.Cursor, obj types.Object) bool {
	for c := range cur.Enclosing((*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)) {
		fn := c.Node()
		return obj.Pos() < fn.Pos() || obj.Pos() >= fn.End()
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

// mayBreakIn checks if obj may be set to a value without the property of g
// anywhere within node.
func mayBreakIn(pass *analysis.Pass, node ast.Node, obj types.Object, g guarantee) bool {
	for n := range ast.Preorder(node) {
		if mayBreak(pass, n, obj, g) {
			return true
		}
	}
	return false
}

// mayBreak checks if n writes to obj, other than by a single assignment of a
// value with the property of g such as `err = fmt.Errorf("...: %w", err)` for
// nonNilGuarantee, which cannot make obj nil.
func mayBreak(pass *analysis.Pass, n ast.Node, obj types.Object, g guarantee) bool {
	for _, expr := range writtenExprs(n) {
		if refersTo(pass, expr, obj) {
			return !assigns(pass, n, obj, g)
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

// isErrorInterface checks if typ is an interface that implements error.
func isErrorInterface(typ types.Type) bool {
	return types.IsInterface(typ) && implementsError(typ)
}

// isNilableError checks if typ is an interface or a pointer that implements
// error.
func isNilableError(typ types.Type) bool {
	switch typ.Underlying().(type) {
	case *types.Interface, *types.Pointer:
		return implementsError(typ)
	}
	return false
}
