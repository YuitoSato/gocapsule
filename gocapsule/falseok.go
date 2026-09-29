package gocapsule

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// falseGuarantee guarantees that a bool variable is false. Only variables of
// bool types without methods are tracked.
var falseGuarantee = guarantee{
	tracked: isMethodlessBool,
	implied: impliesFalse,
	holds:   isFalseConst,
	zero:    true,
}

// isReturnedWithFalseOk checks if the last result of the return statement at
// retCur is a bool that is guaranteed to be false, e.g. `return T{}, false`.
// With -allowZeroWithFalseOk, it is treated as the ok of the comma-ok idiom,
// so the caller cannot use the zero value without ignoring the ok.
func isReturnedWithFalseOk(pass *analysis.Pass, retCur inspector.Cursor, sig *types.Signature) bool {
	last := sig.Results().Len() - 1
	result := sig.Results().At(last)
	if !isBool(result.Type()) {
		return false
	}
	// A deferred call may set a named result to true after the return,
	// e.g. `defer func() { ok = true }()`
	if result.Name() != "" && mayBreakIndirectly(pass, retCur, result, falseGuarantee) {
		return false
	}
	return isFalseOperand(pass, retCur.ChildAt(edge.ReturnStmt_Results, last), result.Type())
}

// isFalseOperand checks if the operand at cur is guaranteed to be false,
// taking checks before it into account.
func isFalseOperand(pass *analysis.Pass, cur inspector.Cursor, typ types.Type) bool {
	cur = unparenCursor(cur)
	if ident, ok := cur.Node().(*ast.Ident); ok && isGuarded(pass, cur, ident, falseGuarantee) {
		return true
	}
	return isFalseConst(pass, cur.Node().(ast.Expr), typ)
}

// isFalseConst checks if expr is a constant false, e.g. `false`, `!true`, or
// a constant declared as `const notFound = false`.
func isFalseConst(pass *analysis.Pass, expr ast.Expr, _ types.Type) bool {
	value, ok := boolConst(pass, expr)
	return ok && !value
}

// boolConst returns the value of expr if it is a bool constant.
func boolConst(pass *analysis.Pass, expr ast.Expr) (value, ok bool) {
	tv, found := pass.TypesInfo.Types[expr]
	if !found || tv.Value == nil || tv.Value.Kind() != constant.Bool {
		return false, false
	}
	return constant.BoolVal(tv.Value), true
}

// impliesFalse checks if cond evaluating to condValue guarantees that the
// bool variable obj is false.
func impliesFalse(pass *analysis.Pass, cond ast.Expr, obj types.Object, condValue bool) bool {
	switch e := ast.Unparen(cond).(type) {
	case *ast.Ident:
		return !condValue && refersTo(pass, e, obj)
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return impliesFalse(pass, e.X, obj, !condValue)
		}
	case *ast.BinaryExpr:
		switch e.Op {
		case token.EQL, token.NEQ:
			// obj is c if `obj == c` is true or `obj != c` is false, and !c
			// otherwise
			c, ok := comparedBoolConst(pass, e, obj)
			isC := condValue == (e.Op == token.EQL)
			return ok && c != isC
		case token.LAND:
			return condValue && (impliesFalse(pass, e.X, obj, true) || impliesFalse(pass, e.Y, obj, true))
		case token.LOR:
			return !condValue && (impliesFalse(pass, e.X, obj, false) || impliesFalse(pass, e.Y, obj, false))
		}
	}
	return false
}

// comparedBoolConst returns the bool constant that e compares obj with, e.g.
// false in `ok == false`.
func comparedBoolConst(pass *analysis.Pass, e *ast.BinaryExpr, obj types.Object) (value, ok bool) {
	switch {
	case refersTo(pass, e.X, obj):
		return boolConst(pass, e.Y)
	case refersTo(pass, e.Y, obj):
		return boolConst(pass, e.X)
	}
	return false, false
}

// isBool checks if the underlying type of typ is bool.
func isBool(typ types.Type) bool {
	basic, ok := typ.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsBoolean != 0
}

// isMethodlessBool checks if typ is a bool type without methods. A method
// call with a pointer receiver implicitly takes the address of a variable.
func isMethodlessBool(typ types.Type) bool {
	return isBool(typ) && types.NewMethodSet(types.NewPointer(typ)).Len() == 0
}
