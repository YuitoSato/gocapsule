package errs

import (
	"errors"
	"fmt"
)

type wrapError struct {
	cause error
	msg   string
}

func (e *wrapError) Error() string { return e.msg + ": " + e.cause.Error() }

type detailError struct {
	cause  error
	detail string
}

func (e *detailError) Error() string { return e.cause.Error() + " (" + e.detail + ")" }

// OK: returns nil only if err is nil, and annotate declared after it returns
// a non-nil error for a non-nil err
func Wrap(err error, msg string) error { // want Wrap:`nonNilResult\[0\]`
	if err == nil {
		return nil
	}
	return annotate(err, msg, "")
}

// OK: e is assigned a non-nil value before the return
func annotate(err error, msg, detail string) error { // want annotate:`nonNilResult\[0\]`
	if err == nil {
		return nil
	}
	e := err
	if detail != "" {
		e = &detailError{cause: e, detail: detail}
	}
	e = &wrapError{cause: e, msg: msg}
	return e
}

// OK: a method of a generic type declared later
func WrapBoxed(err error) error { // want WrapBoxed:`nonNilResult\[0\]`
	return Box[int]{}.Wrap(err)
}

// OK: a generic function declared later
func NewNotFound() error { // want NewNotFound:`nonNilResult\[\]`
	return NewCoded("not_found")
}

// OK: err is non-nil throughout the function
func Wrapf(err error, format string, args ...any) error { // want Wrapf:`nonNilResult\[0\]`
	return Wrap(err, fmt.Sprintf(format, args...))
}

// OK: a function without error parameters always returns a non-nil error
func New(msg string) error { // want New:`nonNilResult\[\]`
	return &wrapError{cause: errors.New(msg), msg: msg}
}

// OK: returns err itself
func Passthrough(err error) error { // want Passthrough:`nonNilResult\[0\]`
	return err
}

// OK: the last return is never reached when err is non-nil
func WithDetail(err error, detail string) error { // want WithDetail:`nonNilResult\[0\]`
	if err != nil {
		return &detailError{cause: err, detail: detail}
	}
	return nil
}

// OK: the else of `if err != nil`, and a negated condition
func WrapInElse(err error, cond bool) error { // want WrapInElse:`nonNilResult\[0\]`
	if !(err == nil) && cond {
		return &wrapError{cause: err}
	} else if err != nil {
		return err
	} else {
		return nil
	}
}

// OK: && and || in the conditions
func WrapWithConditions(err error, cond bool) error { // want WrapWithConditions:`nonNilResult\[0\]`
	if cond && err == nil {
		return nil
	}
	if cond || err != nil {
		return &wrapError{cause: err}
	}
	return nil
}

// OK: a switch without a tag
func WrapInSwitch(err error) error { // want WrapInSwitch:`nonNilResult\[0\]`
	switch {
	case err == nil:
		return nil
	default:
		return &wrapError{cause: err}
	}
}

// OK: err is non-nil when the loop condition is false
func NewAfterLoop() error { // want NewAfterLoop:`nonNilResult\[\]`
	err := lookup()
	for err == nil {
		err = New("loop")
	}
	return err
}

// OK: a type switch
func WrapTypeSwitch(err error) error { // want WrapTypeSwitch:`nonNilResult\[0\]`
	switch e := err.(type) {
	case *CodeError:
		return e
	default:
		return err
	}
}

// OK: a variable declared in a nested block
func WrapInBlock(err error, cond bool) error { // want WrapInBlock:`nonNilResult\[0\]`
	if cond {
		wrapped := Wrap(err, "block")
		return wrapped
	}
	return err
}

// OK: err is reassigned with verified functions of err itself
func WrapTwice(err error, msg string) error { // want WrapTwice:`nonNilResult\[0\]`
	if err == nil {
		return nil
	}
	if msg != "" {
		err = Wrap(err, msg)
	}
	err = WithDetail(err, "twice")
	return err
}

// OK: err is assigned in both branches, and reassigned in a loop with a
// verified function that only requires err to be non-nil
func NewWithCauses(msg string, causes ...error) error { // want NewWithCauses:`nonNilResult\[\]`
	var err error
	if len(causes) > 0 {
		err = &wrapError{cause: causes[0], msg: msg}
	} else {
		err = error(&wrapError{cause: errors.New(msg), msg: msg})
	}
	for _, cause := range causes {
		err = Combine(err, cause)
	}
	err = WithDetail(err, "causes")
	return err
}

// OK: returns err when other is nil, so only err must be non-nil
func Combine(err, other error) error { // want Combine:`nonNilResult\[0\]`
	if err == nil || other == nil {
		return err
	}
	return &wrapError{cause: err, msg: other.Error()}
}

// OK: err is only assigned non-nil values, also in a function literal
func WrapInFuncLit(err error) error { // want WrapInFuncLit:`nonNilResult\[0\]`
	wrap := func() { err = &wrapError{cause: err} }
	wrap()
	return err
}

// OK: the return in the function literal returns from the literal, and err
// does not have to be non-nil
func WrapWithFuncLitReturn(err error) error { // want WrapWithFuncLitReturn:`nonNilResult\[\]`
	f := func() error { return nil }
	_ = f
	return &wrapError{cause: err}
}

// OK: a pointer parameter
func WithCode(e *CodeError) error { // want WithCode:`nonNilResult\[0\]`
	if e == nil {
		return nil
	}
	return &wrapError{cause: e}
}

// CodeError is an error with a code
type CodeError struct{ Code int }

func (e *CodeError) Error() string { return "code error" }

// OK: a copy of err
func Copy(err error) error { // want Copy:`nonNilResult\[0\]`
	copied := err
	return copied
}

// OK: a bare return of the named result
func WrapNamed(err error) (wrapped error) { // want WrapNamed:`nonNilResult\[0\]`
	if err != nil {
		wrapped = &wrapError{cause: err}
	}
	return
}

// OK: a generic function
func WrapGeneric[T any](err error, _ T) error { // want WrapGeneric:`nonNilResult\[0\]`
	if err == nil {
		return nil
	}
	return &wrapError{cause: err}
}

type codedError struct {
	cause error
	code  any
}

func (e *codedError) Error() string { return fmt.Sprint(e.code) }

func coded(err error, code any) error { // want coded:`nonNilResult\[\]`
	return &codedError{cause: err, code: code}
}

// OK: passes nil for an error parameter that does not have to be non-nil
func NewCoded[C comparable](code C) error { // want NewCoded:`nonNilResult\[\]`
	return coded(nil, code)
}

// OK: err does not have to be non-nil
func Translate[C comparable](err error, code C) error { // want Translate:`nonNilResult\[\]`
	return coded(err, code)
}

// Wrapper wraps errors
type Wrapper struct{}

// OK: a method
func (Wrapper) Wrap(err error) error { // want Wrap:`nonNilResult\[0\]`
	return Wrap(err, "wrapped")
}

// Box wraps errors
type Box[T any] struct{}

// OK: a method of a generic type
func (Box[T]) Wrap(err error) error { // want Wrap:`nonNilResult\[0\]`
	return Wrap(err, "boxed")
}

// ErrNotFound is a sentinel error
var ErrNotFound = errors.New("not found") // want ErrNotFound:`nonNilError`

// ErrWrapped is a sentinel error initialized by Wrap
var ErrWrapped = Wrap(ErrNotFound, "wrapped") // want ErrWrapped:`nonNilError`

// ErrFromHelper is initialized by a function that returns a sentinel error
var ErrFromHelper = notFound() // want ErrFromHelper:`nonNilError`

// OK: a sentinel error of the same package
func notFound() error { // want notFound:`nonNilResult\[\]`
	return ErrNotFound
}

// OK: a sentinel error that gets its fact after Wrap does
func wrappedNotFound() error { // want wrappedNotFound:`nonNilResult\[\]`
	return ErrWrapped
}

// OK: errors.Join is non-nil if one of its arguments is
func JoinNotFound(err error) error { // want JoinNotFound:`nonNilResult\[\]`
	return errors.Join(err, ErrNotFound)
}

// OK: lookup may return nil, so only err must be non-nil
func JoinLookup(err error) error { // want JoinLookup:`nonNilResult\[0\]`
	return errors.Join(err, lookup())
}

// OK: either err or other is enough, but the fact can only require all of
// its parameters, so it requires the last one
func JoinBoth(err, other error) error { // want JoinBoth:`nonNilResult\[1\]`
	return errors.Join(err, other)
}

// Not verified: the elements of errs may all be nil
func JoinAll(errs ...error) error {
	return errors.Join(errs...)
}

// Not verified: returns nil when cond is true
func MaybeWrap(err error, cond bool) error {
	if cond {
		return nil
	}
	return Wrap(err, "maybe")
}

func lookup() error { return nil }

// Not verified: err is set to a possibly nil value
func WrapLookup(err error) error {
	err = lookup()
	if err == nil {
		return nil
	}
	return err
}

// Not verified: err is assigned in only one branch
func NewInOneBranch(cond bool) error {
	var err error
	if cond {
		err = New("one branch")
	}
	return err
}

// Not verified: err may be set to nil in the loop
func WrapInLoop(err error, n int) error {
	for i := 0; i < n; i++ {
		err = lookup()
	}
	return err
}

// Not verified: err is assigned by range
func WrapRange(err error, errs []error) error {
	for _, err = range errs {
	}
	return err
}

// Not verified: the function literal may set err to nil
func WrapResetInFuncLit(err error) error {
	reset := func() { err = nil }
	reset()
	return err
}

// Not verified: the function literal may set err to nil after the nil check
func WrapResetAfterCheck(err error) error {
	reset := func() { err = nil }
	if err != nil {
		reset()
		if err == nil {
			return nil
		}
	}
	return New("reset")
}

func reset(err *error) { *err = nil }

// Not verified: the address of err is taken
func WrapAddressTaken(err error) error {
	reset(&err)
	return err
}

// Not verified: the case of a switch with a tag is compared with the tag
func WrapTaggedSwitch(cond bool) error {
	err := lookup()
	switch cond {
	case err != nil:
		return err
	}
	return New("tagged")
}

// Not verified: a deferred call may recover from a panic and return nil
func WrapWithRecover(err error) error {
	defer func() { _ = recover() }()
	if err == nil {
		return nil
	}
	return &wrapError{cause: err}
}

type wrapper interface {
	Wrap(err error) error
}

// Not verified: an interface method may return nil
func WrapViaInterface(err error, w wrapper) error {
	if err == nil {
		return nil
	}
	return w.Wrap(err)
}

// Not verified: the return after the label may be reached by a goto
func WrapWithGoto(err error, cond bool) error {
	if cond {
		goto end
	}
	if err != nil {
		return &wrapError{cause: err}
	}
end:
	return nil
}

// Not verified: a recursive call
func WrapRecursive(err error, n int) error {
	if n == 0 {
		return &wrapError{cause: err}
	}
	return WrapRecursive(err, n-1)
}
