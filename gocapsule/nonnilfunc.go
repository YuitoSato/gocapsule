package gocapsule

import (
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/cfg"
	"golang.org/x/tools/go/types/typeutil"
)

// funcCandidate is a function declaration that may have the NonNilResult
// fact.
type funcCandidate struct {
	fn  *types.Func
	cur inspector.Cursor
	cfg *cfg.CFG
	// conds are the conditions that select a branch: those of if and for
	// statements, and the cases of switch statements without a tag, with the
	// indices of the local variables that they refer to
	conds map[ast.Expr][]int
	// locals are the local variables of interface and pointer types of the
	// function, by their indices in a varSet, and vars are the variables by
	// their indices. size is the number of objects declared in the function,
	// which bounds the number of the variables.
	locals map[*types.Var]int
	vars   []*types.Var
	size   int
	// deps are the objects of the package whose facts the last verification
	// looked up. The result depends only on them, since the facts of other
	// packages do not change.
	deps []types.Object
	// checked is the number of facts exported in the package when the
	// function was last verified, or -1
	checked int
}

// nonNilResultCandidates returns the functions whose only result is an error
// interface.
func nonNilResultCandidates(pass *analysis.Pass, inspect *inspector.Inspector) []*funcCandidate {
	var candidates []*funcCandidate
	for cur := range inspect.Root().Preorder((*ast.FuncDecl)(nil)) {
		decl := cur.Node().(*ast.FuncDecl)
		fn, ok := pass.TypesInfo.Defs[decl.Name].(*types.Func)
		if !ok || decl.Body == nil {
			continue
		}
		sig := fn.Signature()
		if sig.Results().Len() != 1 || !isErrorInterface(sig.Results().At(0).Type()) {
			continue
		}
		if c := newFuncCandidate(pass, cur, fn); c != nil {
			candidates = append(candidates, c)
		}
	}
	return candidates
}

// newFuncCandidate returns the candidate for the function fn declared at cur,
// or nil if it cannot be verified: it has a defer statement, or it returns a
// value that cannot be non-nil. A deferred call may recover from a panic, and
// the function then returns without a return statement, e.g. a nil error.
func newFuncCandidate(pass *analysis.Pass, cur inspector.Cursor, fn *types.Func) *funcCandidate {
	var conds []ast.Expr
	var returns []*ast.ReturnStmt
	hasDefer := false
	nodeTypes := []ast.Node{(*ast.FuncLit)(nil), (*ast.DeferStmt)(nil), (*ast.ReturnStmt)(nil), (*ast.IfStmt)(nil), (*ast.ForStmt)(nil), (*ast.SwitchStmt)(nil)}
	cur.Inspect(nodeTypes, func(c inspector.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.FuncLit:
			// The statements of a function literal belong to the literal
			return false
		case *ast.DeferStmt:
			hasDefer = true
		case *ast.ReturnStmt:
			returns = append(returns, n)
		case *ast.IfStmt:
			conds = append(conds, n.Cond)
		case *ast.ForStmt:
			if n.Cond != nil {
				conds = append(conds, n.Cond)
			}
		case *ast.SwitchStmt:
			// A case of a switch with a tag is compared with the tag
			if n.Tag == nil {
				for _, clause := range n.Body.List {
					conds = append(conds, clause.(*ast.CaseClause).List...)
				}
			}
		}
		return !hasDefer
	})
	if hasDefer {
		return nil
	}

	// Only a nil check of a local variable can make a branch unreachable, so
	// without one, every return statement is reached, except in dead code
	// after a return or a panic. Reject the function without building the CFG
	// if one of them cannot return a non-nil value, even if every local
	// variable is non-nil and every function and variable of the package gets
	// a fact.
	if !slices.ContainsFunc(conds, func(cond ast.Expr) bool { return hasLocalNilCheck(pass, cond) }) {
		result := fn.Signature().Results().At(0)
		for _, ret := range returns {
			if len(ret.Results) == 1 && !mayBeNonNil(pass, ret.Results[0], result.Type()) {
				return nil
			}
		}
	}

	mayReturn := func(call *ast.CallExpr) bool { return !isBuiltinCall(pass, call, "panic") }
	c := &funcCandidate{
		fn:      fn,
		cur:     cur,
		cfg:     cfg.New(cur.Node().(*ast.FuncDecl).Body, mayReturn),
		conds:   make(map[ast.Expr][]int, len(conds)),
		locals:  make(map[*types.Var]int),
		size:    countObjects(fn.Scope()),
		checked: -1,
	}
	for _, cond := range conds {
		var vars []int
		for n := range ast.Preorder(cond) {
			if ident, ok := n.(*ast.Ident); ok {
				if i, ok := c.index(pass.TypesInfo.Uses[ident]); ok && !slices.Contains(vars, i) {
					vars = append(vars, i)
				}
			}
		}
		c.conds[cond] = vars
	}
	return c
}

// hasLocalNilCheck checks if cond compares a local variable with nil.
func hasLocalNilCheck(pass *analysis.Pass, cond ast.Expr) bool {
	for n := range ast.Preorder(cond) {
		e, ok := n.(*ast.BinaryExpr)
		if !ok || (e.Op != token.EQL && e.Op != token.NEQ) {
			continue
		}
		for _, operands := range [][2]ast.Expr{{e.X, e.Y}, {e.Y, e.X}} {
			ident, ok := ast.Unparen(operands[0]).(*ast.Ident)
			if ok && isNil(pass, operands[1]) && isLocalNilable(pass.TypesInfo.Uses[ident]) {
				return true
			}
		}
	}
	return false
}

// mayBeNonNil checks if expr can be a non-nil value of typ in some state of
// the analysis: when every local variable is non-nil, and every function and
// package-level variable of the package gets a fact, which may happen later.
// The facts of other packages are final.
func mayBeNonNil(pass *analysis.Pass, expr ast.Expr, typ types.Type) bool {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		obj := pass.TypesInfo.Uses[e]
		if isLocalNilable(obj) {
			return true
		}
		if v, ok := obj.(*types.Var); ok && v.Pkg() == pass.Pkg && v.Parent() == pass.Pkg.Scope() {
			return true
		}
	case *ast.CallExpr:
		// An interface method never gets a fact
		if fn, ok := typeutil.Callee(pass.TypesInfo, e).(*types.Func); ok && fn.Pkg() == pass.Pkg {
			if recv := fn.Signature().Recv(); recv == nil || !types.IsInterface(recv.Type()) {
				return true
			}
		}
		if isNonNilIfArgs(pass, e, func(i int, typ types.Type) bool {
			return mayBeNonNil(pass, e.Args[i], typ)
		}) {
			return true
		}
	}
	return isNonNilExpr(pass, expr, typ)
}

// isLocalNilable checks if obj is a local variable of an interface or pointer
// type.
func isLocalNilable(obj types.Object) bool {
	v, ok := obj.(*types.Var)
	return ok && isLocalVar(v) && isNilable(v.Type())
}

// countObjects returns the number of objects declared in scope and its
// children.
func countObjects(scope *types.Scope) int {
	n := scope.Len()
	for child := range scope.Children() {
		n += countObjects(child)
	}
	return n
}

// index returns the index of obj in a varSet if it is a local variable of an
// interface or pointer type, which is assigned when first needed.
func (c *funcCandidate) index(obj types.Object) (int, bool) {
	if !isLocalNilable(obj) {
		return 0, false
	}
	v := obj.(*types.Var)
	if i, ok := c.locals[v]; ok {
		return i, true
	}
	// The variables of the function are declared in its scope
	if len(c.vars) >= c.size {
		return 0, false
	}
	c.locals[v] = len(c.vars)
	c.vars = append(c.vars, v)
	return c.locals[v], true
}

// exportNonNilResultFacts exports facts for the candidate functions that
// return a non-nil error whenever the arguments for some of their error
// parameters are non-nil, and returns the remaining candidates. A function
// that calls a function declared after it gets the fact in a later round.
func exportNonNilResultFacts(pass *analysis.Pass, log *factLog, candidates []*funcCandidate) []*funcCandidate {
	var remaining []*funcCandidate
	for _, c := range candidates {
		// The result can change only after a fact it depends on is exported
		if c.checked >= 0 && !log.exportedSince(c.deps, c.checked) {
			remaining = append(remaining, c)
			continue
		}
		c.checked = log.len()
		if params, ok := c.requiredParams(pass); ok {
			log.export(c.fn, &NonNilResult{Params: params})
		} else {
			remaining = append(remaining, c)
		}
	}
	return remaining
}

// requiredParams returns the indices of the error parameters that must be
// non-nil for the function to return a non-nil error. It returns false if
// the function may return nil even when all of them are non-nil.
func (c *funcCandidate) requiredParams(pass *analysis.Pass) ([]int, bool) {
	deps := make(map[types.Object]bool)
	defer func() { c.deps = slices.Collect(maps.Keys(deps)) }()
	recording := *pass
	recording.ImportObjectFact = func(obj types.Object, fact analysis.Fact) bool {
		if obj.Pkg() == pass.Pkg {
			deps[obj] = true
		}
		return pass.ImportObjectFact(obj, fact)
	}

	a := newNilAnalysis(&recording, c)
	params := c.fn.Signature().Params()
	var required []int
	for i := range params.Len() {
		if _, ok := a.index(params.At(i)); ok && isNilableError(params.At(i).Type()) {
			required = append(required, i)
		}
	}
	if !a.returnsNonNil(required) {
		return nil, false
	}

	// Drop the parameters that are not needed, e.g. an error that is only
	// stored in the result
	for i := 0; i < len(required); {
		fewer := slices.Delete(slices.Clone(required), i, i+1)
		if a.returnsNonNil(fewer) {
			required = fewer
		} else {
			i++
		}
	}
	return required, true
}

// varSet is a set of the local variables of a funcCandidate, by their
// indices.
type varSet []uint64

// newVarSet returns an empty set of the local variables of c. It is never
// nil, so that a nil varSet can stand for an unreachable block.
func newVarSet(c *funcCandidate) varSet {
	return make(varSet, max(1, (c.size+63)/64))
}

func (s varSet) has(i int) bool { return s[i/64]&(1<<(i%64)) != 0 }

func (s varSet) set(i int, in bool) {
	if in {
		s[i/64] |= 1 << (i % 64)
	} else {
		s[i/64] &^= 1 << (i % 64)
	}
}

// intersect removes the variables not in t from s, and reports whether s
// changed.
func (s varSet) intersect(t varSet) bool {
	changed := false
	for i := range s {
		if s[i]&^t[i] != 0 {
			s[i] &= t[i]
			changed = true
		}
	}
	return changed
}

// nilAnalysis tracks which local variables of a function are non-nil along
// its control flow.
type nilAnalysis struct {
	pass *analysis.Pass
	c    *funcCandidate
	// untracked are the variables that may be set without a CFG node, or to a
	// possibly nil value at any time
	untracked map[*types.Var]bool
}

// newNilAnalysis returns a nilAnalysis for the candidate c.
func newNilAnalysis(pass *analysis.Pass, c *funcCandidate) *nilAnalysis {
	a := &nilAnalysis{pass: pass, c: c, untracked: make(map[*types.Var]bool)}
	for cur := range c.cur.Preorder(writeNodes...) {
		for _, expr := range writtenExprs(cur.Node()) {
			ident, ok := ast.Unparen(expr).(*ast.Ident)
			if !ok {
				continue
			}
			v, ok := pass.TypesInfo.ObjectOf(ident).(*types.Var)
			if !ok {
				continue
			}
			switch n := cur.Node().(type) {
			case *ast.UnaryExpr, *ast.RangeStmt:
				// The address may be written through at any time, and the CFG
				// has no node for the assignment of `for k, v = range`
				a.untracked[v] = true
			case *ast.AssignStmt:
				// A function literal may run at any time
				if isInNestedFunc(cur, v) && !assigns(pass, n, v, nonNilGuarantee) {
					a.untracked[v] = true
				}
			}
		}
	}
	return a
}

// index returns the index of obj if it is a tracked local variable.
func (a *nilAnalysis) index(obj types.Object) (int, bool) {
	if v, ok := obj.(*types.Var); !ok || a.untracked[v] {
		return 0, false
	}
	return a.c.index(obj)
}

// returnsNonNil checks if every reachable return statement of the function
// returns a non-nil error when the parameters at params are non-nil at the
// start.
func (a *nilAnalysis) returnsNonNil(params []int) bool {
	sig := a.c.fn.Signature()
	entry := newVarSet(a.c)
	for _, i := range params {
		if j, ok := a.index(sig.Params().At(i)); ok {
			entry.set(j, true)
		}
	}

	result := sig.Results().At(0)
	in := a.solve(entry)
	for _, b := range a.c.cfg.Blocks {
		s := in[b.Index]
		if s == nil {
			continue
		}
		for _, n := range b.Nodes {
			if ret, ok := n.(*ast.ReturnStmt); ok && !a.returnsNonNilResult(ret, result, s) {
				return false
			}
			a.transfer(n, s)
		}
	}
	return true
}

// returnsNonNilResult checks if ret returns a non-nil value of result when the
// variables in s are non-nil.
func (a *nilAnalysis) returnsNonNilResult(ret *ast.ReturnStmt, result *types.Var, s varSet) bool {
	// A bare return returns the named result
	if len(ret.Results) == 0 {
		i, ok := a.index(result)
		return ok && s.has(i)
	}
	return len(ret.Results) == 1 && a.isNonNil(ret.Results[0], result.Type(), s)
}

// solve returns the variables that are non-nil at the start of each block,
// by the block index, given those at the entry, or nil for an unreachable
// block. A branch that requires a non-nil variable to be nil, e.g.
// `if err == nil` for a non-nil err, is not taken.
func (a *nilAnalysis) solve(entry varSet) []varSet {
	blocks := a.c.cfg.Blocks
	in := make([]varSet, len(blocks))
	in[0] = entry
	out, refined := newVarSet(a.c), newVarSet(a.c)
	work := []*cfg.Block{blocks[0]}
	for len(work) > 0 {
		b := work[len(work)-1]
		work = work[:len(work)-1]

		copy(out, in[b.Index])
		for _, n := range b.Nodes {
			a.transfer(n, out)
		}

		cond := a.condition(b)
		for i, succ := range b.Succs {
			s := out
			if cond != nil {
				// The first successor runs when the condition is true
				copy(refined, out)
				if !a.refine(refined, cond, i == 0) {
					continue
				}
				s = refined
			}
			if merge(in, succ, s) {
				work = append(work, succ)
			}
		}
	}
	return in
}

// merge merges s into the variables at the start of b, which are non-nil only
// if they are non-nil on every path to b, and reports whether they changed.
func merge(in []varSet, b *cfg.Block, s varSet) bool {
	if in[b.Index] == nil {
		in[b.Index] = slices.Clone(s)
		return true
	}
	return in[b.Index].intersect(s)
}

// condition returns the condition that selects between the two successors of
// b, or nil.
func (a *nilAnalysis) condition(b *cfg.Block) ast.Expr {
	if len(b.Succs) != 2 || len(b.Nodes) == 0 {
		return nil
	}
	cond, ok := b.Nodes[len(b.Nodes)-1].(ast.Expr)
	if !ok {
		return nil
	}
	if _, ok := a.c.conds[cond]; !ok {
		return nil
	}
	return cond
}

// refine adds to s the variables that are non-nil when cond evaluates to
// condValue. It returns false if cond cannot evaluate to condValue because it
// requires a variable in s to be nil.
func (a *nilAnalysis) refine(s varSet, cond ast.Expr, condValue bool) bool {
	vars := a.c.conds[cond]
	for _, i := range vars {
		if s.has(i) && impliesNil(a.pass, cond, a.c.vars[i], condValue) {
			return false
		}
	}
	for _, i := range vars {
		v := a.c.vars[i]
		if !s.has(i) && !a.untracked[v] && impliesNonNil(a.pass, cond, v, condValue) {
			s.set(i, true)
		}
	}
	return true
}

// transfer updates s with the assignments in n.
func (a *nilAnalysis) transfer(n ast.Node, s varSet) {
	switch n := n.(type) {
	case *ast.AssignStmt:
		// All right-hand sides are evaluated before the assignments
		simple := (n.Tok == token.ASSIGN || n.Tok == token.DEFINE) && len(n.Lhs) == len(n.Rhs)
		var buf [8]bool
		nonNil := buf[:0]
		for i, lhs := range n.Lhs {
			v, _, ok := a.assigned(lhs)
			nonNil = append(nonNil, ok && simple && a.isNonNil(n.Rhs[i], v.Type(), s))
		}
		for i, lhs := range n.Lhs {
			if _, j, ok := a.assigned(lhs); ok {
				s.set(j, nonNil[i])
			}
		}
	case *ast.ValueSpec:
		for i, name := range n.Names {
			if v, j, ok := a.assigned(name); ok {
				s.set(j, len(n.Values) == len(n.Names) && a.isNonNil(n.Values[i], v.Type(), s))
			}
		}
	}
}

// assigned returns the tracked variable that expr on the left-hand side of an
// assignment denotes, and its index.
func (a *nilAnalysis) assigned(expr ast.Expr) (*types.Var, int, bool) {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	if !ok {
		return nil, 0, false
	}
	v, ok := a.pass.TypesInfo.ObjectOf(ident).(*types.Var)
	if !ok {
		return nil, 0, false
	}
	i, ok := a.index(v)
	return v, i, ok
}

// isNonNil checks if expr is a non-nil value of typ when the variables in s
// are non-nil.
func (a *nilAnalysis) isNonNil(expr ast.Expr, typ types.Type, s varSet) bool {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		if i, ok := a.index(a.pass.TypesInfo.Uses[e]); ok && s.has(i) {
			return true
		}
	case *ast.CallExpr:
		if isNonNilIfArgs(a.pass, e, func(i int, typ types.Type) bool {
			return a.isNonNil(e.Args[i], typ, s)
		}) {
			return true
		}
	}
	return isNonNilExpr(a.pass, expr, typ)
}

// isNilable checks if typ is an interface or a pointer.
func isNilable(typ types.Type) bool {
	switch typ.Underlying().(type) {
	case *types.Interface, *types.Pointer:
		return true
	}
	return false
}
