package external

import (
	"errors"
	"fmt"

	"target"
)

// Violation: package-level zero value declaration
var globalUser target.User // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead`

// OK: pointer zero value is nil, not a User
var globalUserPtr *target.User

func TestZeroValue() {
	// Violation: empty struct literal
	_ = target.User{}  // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	_ = &target.User{} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// Violation: elided empty struct literal
	_ = []target.User{{}} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`

	// Violation: var declaration without initializer
	var u target.User // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead`
	_ = u

	// Violation: one diagnostic per declared name
	var a, b target.User // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead` `zero value declaration of User is not allowed; use target.NewUser\(\) instead`
	_, _ = a, b

	// Violation: new
	_ = new(target.User) // want `zero value creation of User with new\(\) is not allowed; use target.NewUser\(\) instead`

	// Violation: zero values of defined types
	var e target.Email // want `zero value declaration of Email is not allowed; use target.NewEmail\(\) instead`
	_ = e
	_ = new(target.Email) // want `zero value creation of Email with new\(\) is not allowed; use target.NewEmail\(\) instead`

	// Violation: plain New() constructor
	var r target.Repository // want `zero value declaration of Repository is not allowed; use target.New\(\) instead`
	_ = r

	// OK: the zero value of a pointer is nil, not a User
	var p *target.User
	_ = p
	_ = new(*target.User)

	// OK: initialized declarations
	var v = target.NewUser("name", "email@test.com", 25)
	var w target.User = *target.NewUser("name", "email@test.com", 25)
	_, _ = v, w

	// OK: Config has no constructor
	var cfg target.Config
	_ = cfg
	_ = new(target.Config)
	_ = target.Config{}

	// OK: interfaces are not encapsulated, even if a constructor returns them
	var store target.Store
	_ = store
	_ = new(target.Store)
	_ = target.Store(fakeStore{})
}

type fakeStore struct{}

func (fakeStore) Get(string) string { return "" }

type myError struct{}

func (*myError) Error() string { return "my error" }

func findUser() (*target.User, error) {
	return nil, nil
}

func resetError(err *error) {
	*err = nil
}

// OK: guarded by if err != nil
func ReturnZeroWithGuardedError() (*target.User, error) {
	u, err := findUser()
	if err != nil {
		return &target.User{}, err
	}
	return u, nil
}

// OK: value type, err declared in the if statement
func ReturnValueZeroWithGuardedError() (target.User, error) {
	if _, err := findUser(); err != nil {
		return target.User{}, err
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// OK: new(T) and parenthesized operands
func ReturnNewWithGuardedError() (*target.User, error) {
	_, err := findUser()
	if nil != err {
		return new(target.User), (err)
	}
	if err != nil {
		return (&target.User{}), err
	}
	return nil, nil
}

// OK: defined type
func ReturnZeroEmailWithGuardedError() (*target.Email, error) {
	_, err := findUser()
	if err != nil {
		return new(target.Email), err
	}
	return nil, nil
}

// OK: else branch of err == nil
func ReturnZeroInElseOfNilCheck(cond bool) (*target.User, error) {
	u, err := findUser()
	if err == nil {
		return u, nil
	} else {
		return &target.User{}, err
	}
}

// OK: else-if chain of err == nil
func ReturnZeroInElseIfOfNilCheck(cond bool) (*target.User, error) {
	u, err := findUser()
	if err == nil {
		return u, nil
	} else if cond {
		return &target.User{}, err
	}
	return nil, nil
}

// OK: conditions implying err != nil
func ReturnZeroWithImpliedCondition(cond bool) (*target.User, error) {
	_, err := findUser()
	if cond && err != nil {
		return &target.User{}, err
	}
	if !(err == nil) {
		return &target.User{}, err
	}
	if !(cond || err == nil) {
		return &target.User{}, err
	}
	if err != nil {
		if cond {
			return &target.User{}, err
		}
	}
	return nil, nil
}

// OK: errors that are never nil
func ReturnZeroWithNewError(err error) (target.User, error) {
	switch {
	case err == nil:
		return target.User{}, errors.New("failed")
	case err != nil:
		return target.User{}, fmt.Errorf("failed: %w", err)
	}
	return target.User{}, &myError{}
}

// OK: a concrete type stored in an error interface is never nil
func ReturnZeroWithConcreteError() (target.User, error) {
	var me *myError
	return target.User{}, me
}

// OK: a non-nil concrete error type
func ReturnZeroWithConcreteErrorResult() (target.User, *myError) {
	return target.User{}, &myError{}
}

// OK: function literal with its own guard
func ReturnZeroInFuncLit() {
	_ = func() (*target.User, error) {
		_, err := findUser()
		if err != nil {
			return &target.User{}, err
		}
		return nil, nil
	}
}

// OK: err is declared in the function literal, so assignments in it are not
// indirect
func ReturnZeroInFuncLitWithReassignedError() {
	_ = func() (*target.User, error) {
		_, err := findUser()
		if err != nil {
			return &target.User{}, err
		}
		_, err = findUser()
		if err != nil {
			return &target.User{}, err
		}
		return nil, nil
	}
}

// Violation: a function literal nested in the function literal assigns err
func ReturnZeroInFuncLitWithErrorAssignedInNestedFuncLit() {
	_ = func() (*target.User, error) {
		_, err := findUser()
		reset := func() { err = nil }
		if err != nil {
			reset()
			return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		}
		return nil, nil
	}
}

// Violation: nil error
func ReturnZeroWithNilError() (target.User, error) {
	return target.User{}, nil // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: no error result
func ReturnZeroWithoutError() *target.User {
	return new(target.User) // want `zero value creation of User with new\(\) is not allowed; use target.NewUser\(\) instead`
}

// Violation: unchecked error
func ReturnZeroWithUncheckedError() (*target.User, error) {
	_, err := findUser()
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: returned in the else branch of err != nil
func ReturnZeroInElseOfNonNilCheck() (*target.User, error) {
	u, err := findUser()
	if err != nil {
		return u, err
	} else {
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
}

// Violation: conditions not implying err != nil
func ReturnZeroWithUnimpliedCondition(cond bool) (*target.User, error) {
	_, err := findUser()
	if cond || err != nil {
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	if err == nil {
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: switch cases are not supported
func ReturnZeroInSwitchCase() (*target.User, error) {
	_, err := findUser()
	switch {
	case err != nil:
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: err is reassigned in the guarded branch
func ReturnZeroWithReassignedError() (*target.User, error) {
	_, err := findUser()
	if err != nil {
		_, err = findUser()
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: err is reassigned after the return in a loop
func ReturnZeroWithReassignedErrorInLoop() (*target.User, error) {
	_, err := findUser()
	if err != nil {
		for i := 0; ; i++ {
			if i > 0 {
				return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
			}
			_, err = findUser()
		}
	}
	return nil, nil
}

// Violation: the address of err is taken
func ReturnZeroWithAddressTakenError() (*target.User, error) {
	_, err := findUser()
	if err != nil {
		resetError(&err)
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: a function literal assigns err
func ReturnZeroWithErrorAssignedInFuncLit() (*target.User, error) {
	_, err := findUser()
	reset := func() { err = nil }
	if err != nil {
		reset()
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: a deferred function literal assigns the named result
func ReturnZeroWithErrorAssignedInDefer() (u *target.User, err error) {
	defer func() { err = nil }()
	_, err = findUser()
	if err != nil {
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: a deferred function may set the named error result to nil after
// the return, whatever the returned error is
func ReturnZeroWithNamedErrorResetInDefer(n int) (u target.User, err error) {
	defer func() {
		if errors.Is(err, target.ErrNotFound) {
			err = nil
		}
	}()
	if n == 0 {
		return target.User{}, target.ErrNotFound // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, errors.New("failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: a deferred call may set the named error result through its address
func ReturnZeroWithAddressTakenNamedError() (u *target.User, err error) {
	defer resetError(&err)
	return &target.User{}, errors.New("failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: errors.Join in a deferred function may return nil
func ReturnZeroWithNamedErrorJoinedInDefer() (u target.User, err error) {
	defer func() { err = errors.Join(err, lookupError()) }()
	return target.User{}, errors.New("failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

type closer struct{}

func (closer) Close() error { return nil }

// Violation: guards are not used for assignments in deferred functions, so
// the close error idiom disables the exemption, even before the defer
func ReturnZeroWithCloseErrorInDefer(c closer, n int) (u target.User, err error) {
	if n == 0 {
		return target.User{}, errors.New("failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	defer func() {
		if cerr := c.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	return target.User{}, errors.New("failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// OK: the close error is wrapped with fmt.Errorf, which is non-nil
func ReturnZeroWithWrappedCloseErrorInDefer(c closer) (u target.User, err error) {
	defer func() {
		if cerr := c.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close: %w", cerr)
		}
	}()
	return target.User{}, errors.New("failed")
}

// Violation: the named error result of a function literal is reset in defer
func ReturnZeroInFuncLitWithNamedErrorResetInDefer() {
	_ = func() (u target.User, err error) {
		defer func() { err = nil }()
		return target.User{}, errors.New("failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
}

// OK: deferred functions only set the named error result to non-nil values
func ReturnZeroWithNamedErrorWrappedInDefer() (u target.User, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	defer func() {
		if err != nil {
			err = fmt.Errorf("wrapped: %w", err)
		}
	}()
	_, err = findUser()
	if err != nil {
		return target.User{}, err
	}
	return target.User{}, errors.New("failed")
}

// OK: the deferred function sets the named result of the enclosing function,
// not of the function literal
func ReturnZeroInFuncLitWithOuterNamedErrorResetInDefer() (err error) {
	defer func() { err = nil }()
	f := func() (target.User, error) {
		return target.User{}, errors.New("failed")
	}
	_, err = f()
	return err
}

var errGlobal error

// Violation: package-level variables can be modified by any function
func ReturnZeroWithPackageLevelError() (*target.User, error) {
	if errGlobal != nil {
		return &target.User{}, errGlobal // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: errors.Join may return nil
func ReturnZeroWithJoinedError(err error) (target.User, error) {
	return target.User{}, errors.Join(err) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: a function literal in the guarded branch may run later
func ReturnZeroInFuncLitInGuardedBranch() {
	_, err := findUser()
	if err != nil {
		_ = func() (*target.User, error) {
			return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		}
	}
}

// Violation: a non-empty literal is never allowed
func ReturnNonZeroWithGuardedError() (*target.User, error) {
	_, err := findUser()
	if err != nil {
		return &target.User{Name: "x"}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: a zero value declared with var is reported at its declaration
func ReturnZeroVarWithGuardedError() (target.User, error) {
	var zero target.User // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead`
	_, err := findUser()
	if err != nil {
		return zero, err
	}
	return zero, nil
}

// Violation: a type parameter constrained by error may be a nil interface
func ReturnZeroWithTypeParamError[E error](e E) (target.User, error) {
	return target.User{}, e // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// OK: early exit unless err != nil
func ReturnZeroAfterEarlyReturn(cond bool) (*target.User, error) {
	u, err := findUser()
	if err == nil {
		return u, nil
	}
	fmt.Println(err)
	if cond {
		return &target.User{}, err
	}
	return new(target.User), err
}

// OK: early exit with panic, an else branch, and a condition implying err != nil
func ReturnZeroAfterEarlyExit(cond bool) (target.User, error) {
	_, err := findUser()
	if cond || err == nil {
		panic("unreachable")
	} else {
		fmt.Println(err)
	}
	return target.User{}, err
}

// OK: early exit with continue and break
func ReturnZeroAfterContinue() (target.User, error) {
	for {
		_, err := findUser()
		if err == nil {
			continue
		}
		switch {
		case true:
			if err == nil {
				break
			}
			return target.User{}, err
		}
		return target.User{}, err
	}
}

// OK: err is assigned before the early exit
func ReturnZeroAfterAssignmentAndEarlyReturn() (*target.User, error) {
	_, err := findUser()
	_, err = findUser()
	if err == nil {
		return nil, nil
	}
	return &target.User{}, err
}

// Violation: the if statement does not exit early
func ReturnZeroAfterNonExitingNilCheck() (*target.User, error) {
	_, err := findUser()
	if err == nil {
		fmt.Println("no error")
	}
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: early exit when err != nil
func ReturnZeroAfterWrongEarlyReturn() (*target.User, error) {
	_, err := findUser()
	if err != nil {
		return nil, err
	}
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: the false branch of && does not imply err != nil
func ReturnZeroAfterConjunctionEarlyReturn(cond bool) (*target.User, error) {
	_, err := findUser()
	if cond && err == nil {
		return nil, nil
	}
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: err is reassigned after the early exit
func ReturnZeroWithReassignmentAfterEarlyReturn(cond bool) (*target.User, error) {
	_, err := findUser()
	if err == nil {
		return nil, nil
	}
	_, err = findUser()
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: err is reassigned in the statement containing the return
func ReturnZeroWithNestedReassignmentAfterEarlyReturn(cond bool) (*target.User, error) {
	_, err := findUser()
	if err == nil {
		return nil, nil
	}
	if cond {
		_, err = findUser()
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	if _, err = findUser(); cond {
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

// Violation: err is reassigned in the else branch of the early exit
func ReturnZeroWithReassignmentInElseOfEarlyReturn() (*target.User, error) {
	_, err := findUser()
	if err == nil {
		return nil, nil
	} else {
		_, err = findUser()
	}
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: goto may skip the early exit
func ReturnZeroAfterGoto(cond bool) (*target.User, error) {
	_, err := findUser()
	if cond {
		goto L
	}
	if err == nil {
		return nil, nil
	}
L:
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: goto in the early exit may jump forward
func ReturnZeroAfterGotoExit() (*target.User, error) {
	_, err := findUser()
	if err == nil {
		goto L
	}
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
L:
	return nil, nil
}

// Violation: the address of err is taken
func ReturnZeroWithAddressTakenErrorAfterEarlyReturn() (*target.User, error) {
	_, err := findUser()
	if err == nil {
		return nil, nil
	}
	resetError(&err)
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

var errLocalSentinel = errors.New("local") // want errLocalSentinel:`nonNilError`

var errWrappedSentinel = fmt.Errorf("wrapped: %w", target.ErrNotFound) // want errWrappedSentinel:`nonNilError`

var errAliasSentinel = target.ErrNotFound // want errAliasSentinel:`nonNilError`

var errConcreteSentinel error = &myError{} // want errConcreteSentinel:`nonNilError`

var errUninitialized error

var errNilSentinel error = nil

var errJoinedSentinel = errors.Join(target.ErrNotFound)

var errReassignedSentinel = errors.New("reassigned")

var errAddressTakenSentinel = errors.New("address taken")

var errRewrappedSentinel = errors.New("rewrapped") // want errRewrappedSentinel:`nonNilError`

func init() {
	errReassignedSentinel = nil
	resetError(&errAddressTakenSentinel)
	errRewrappedSentinel = fmt.Errorf("rewrapped: %w", errRewrappedSentinel)
}

// OK: sentinel errors initialized with a non-nil value and never modified
func ReturnZeroWithSentinelError(n int) (target.User, error) {
	switch n {
	case 0:
		return target.User{}, target.ErrNotFound
	case 1:
		return target.User{}, errLocalSentinel
	case 2:
		return target.User{}, errWrappedSentinel
	case 3:
		return target.User{}, errAliasSentinel
	case 4:
		return target.User{}, errRewrappedSentinel
	}
	return target.User{}, errConcreteSentinel
}

// Violation: package-level variables that may be nil
func ReturnZeroWithNilableSentinelError(n int) (target.User, error) {
	switch n {
	case 0:
		return target.User{}, errUninitialized // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	case 1:
		return target.User{}, errNilSentinel // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	case 2:
		return target.User{}, errJoinedSentinel // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	case 3:
		return target.User{}, errReassignedSentinel // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, errAddressTakenSentinel // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: goto may jump to a labeled statement between the early exit and the return
func ReturnZeroAfterGotoToPrecedingStatement(cond bool) (*target.User, error) {
	_, err := findUser()
	if cond {
		goto L
	}
	if err == nil {
		return nil, nil
	}
L:
	fmt.Println()
	return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: the else branch of && and the body of || do not imply err != nil
func ReturnZeroWithUnimpliedLogicalCondition(cond bool) (*target.User, error) {
	_, err := findUser()
	if cond && err != nil {
		return nil, err
	} else {
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
}

func ReturnZeroWithUnimpliedDisjunction(cond bool) (*target.User, error) {
	_, err := findUser()
	if cond || err == nil {
		return &target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

type errList []error

func (l errList) Error() string { return "errors" }

func (l *errList) Reset() { *l = nil }

// Violation: a method with a pointer receiver can set a non-pointer error to nil
func ReturnZeroWithSliceError(errs errList) (target.User, errList) {
	if errs != nil {
		errs.Reset()
		return target.User{}, errs // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// Violation: a zero value of an error type is not its own non-nil error
func ReturnZeroAppError() error {
	return &target.AppError{} // want `direct struct literal creation of AppError is not allowed; use target.NewAppError\(\) instead`
}

// OK: err is assigned a non-nil value
func ReturnZeroWithWrappedError() (target.User, error) {
	_, err := findUser()
	if err != nil {
		err = fmt.Errorf("find user: %w", err)
		fmt.Println(err)
		return target.User{}, err
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// OK: err is declared with a non-nil value
func ReturnZeroWithDeclaredError(n int) (target.User, error) {
	if n == 0 {
		err := errors.New("failed")
		return target.User{}, err
	}
	if n == 1 {
		var err error = &myError{}
		return target.User{}, err
	}
	var err = target.ErrNotFound
	return target.User{}, err
}

// Violation: err is assigned a possibly nil value after a non-nil one
func ReturnZeroWithReassignmentAfterNonNilAssignment() (target.User, error) {
	err := errors.New("failed")
	_, err = findUser()
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// OK: only non-nil values are assigned after the nil check
func ReturnZeroWithConditionalWrapInGuardedBranch(cond bool) (*target.User, error) {
	_, err := findUser()
	if err != nil {
		if cond {
			err = fmt.Errorf("wrapped: %w", err)
		}
		return &target.User{}, err
	}
	return nil, nil
}

// OK: only non-nil values are assigned in the other branches
func ReturnZeroWithNonNilAssignmentInOtherBranch(n int) (target.User, error) {
	_, err := findUser()
	if err == nil {
		return *target.NewUser("name", "email@test.com", 25), nil
	}
	switch n {
	case 0:
		err = target.ErrNotFound
	default:
		return target.User{}, err
	}
	if n == 1 {
		err = fmt.Errorf("wrapped: %w", err)
	} else {
		return target.User{}, err
	}
	return target.User{}, err
}

// Violation: branches are not distinguished, so a possibly nil value assigned
// in the other branch disables the early exit
func ReturnZeroWithPossiblyNilAssignmentInOtherBranch(cond bool) (target.User, error) {
	_, err := findUser()
	if err == nil {
		return *target.NewUser("name", "email@test.com", 25), nil
	}
	if cond {
		err = lookupError()
	} else {
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// Violation: the non-nil assignment may not run
func ReturnZeroWithConditionalNonNilAssignment(cond bool) (target.User, error) {
	var err error
	if cond {
		err = errors.New("failed")
	}
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: a function literal may set err to nil
func ReturnZeroWithNonNilAssignmentAndFuncLit() (target.User, error) {
	err := errors.New("failed")
	reset := func() { err = nil }
	reset()
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: multiple assignments are not supported
func ReturnZeroWithMultipleNonNilAssignment() (target.User, error) {
	var n int
	n, err := 1, errors.New("failed")
	_ = n
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// OK: errors.Is with a non-nil target and errors.As are false for a nil error
func ReturnZeroWithErrorsIsAndAs() (target.User, error) {
	_, err := findUser()
	if errors.Is(err, target.ErrNotFound) {
		return target.User{}, err
	}
	var me *myError
	if errors.As(err, &me) && me != nil {
		return target.User{}, err
	}
	if !errors.Is(err, errLocalSentinel) {
		return *target.NewUser("name", "email@test.com", 25), nil
	}
	return target.User{}, err
}

// Violation: errors.Is with a possibly nil target is true for a nil error
func ReturnZeroWithErrorsIsPossiblyNilTarget() (target.User, error) {
	_, err := findUser()
	if errors.Is(err, errUninitialized) {
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	if errors.Is(err, nil) {
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	if errors.Is(target.ErrNotFound, err) {
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// Violation: errors.As being false does not imply err == nil or err != nil
func ReturnZeroAfterErrorsAsEarlyReturn() (target.User, error) {
	_, err := findUser()
	var me *myError
	if errors.As(err, &me) {
		return *target.NewUser("name", "email@test.com", 25), nil
	}
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

func lookupError() error {
	return nil
}

// Violation: err is assigned a possibly nil value
func ReturnZeroWithPossiblyNilAssignment() (target.User, error) {
	err := errors.New("failed")
	err = lookupError()
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: a non-nil value is assigned to another variable
func ReturnZeroWithNonNilAssignmentToOtherVariable() (target.User, error) {
	err := lookupError()
	other := errors.New("failed")
	_ = other
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: errors.Is and errors.As on another variable
func ReturnZeroWithErrorsIsOnOtherVariable() (target.User, error) {
	err := lookupError()
	other := lookupError()
	if errors.Is(other, target.ErrNotFound) {
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	var me *myError
	if errors.As(other, &me) {
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// OK: new(E) is never nil, even for a concrete error result
func ReturnZeroWithNewConcreteError() (target.User, *myError) {
	return target.User{}, new(myError)
}

// Violation: a non-nil value for a result that is not an error
func ReturnZeroWithNonErrorResult() (target.User, *int) {
	n := 1
	return target.User{}, &n // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// OK: early exit in a select case
func ReturnZeroInSelectCase(ch chan error) (target.User, error) {
	select {
	case err := <-ch:
		if err == nil {
			return *target.NewUser("name", "email@test.com", 25), nil
		}
		return target.User{}, err
	}
}

// OK: declarations that do not assign err between the nil check and the return
func ReturnZeroAfterDeclarations() (target.User, error) {
	_, err := findUser()
	if err == nil {
		return *target.NewUser("name", "email@test.com", 25), nil
	}
	const msg = "failed"
	type wrapper struct{ error }
	var (
		a = msg
		b = wrapper{err}
	)
	_, _ = a, b
	return target.User{}, err
}

// OK: the early exit ends with an if statement whose branches all exit
func ReturnZeroAfterNestedEarlyExit(cond bool) (target.User, error) {
	_, err := findUser()
	if err == nil {
		if cond {
			return *target.NewUser("name", "email@test.com", 25), nil
		} else {
			panic("unreachable")
		}
	}
	return target.User{}, err
}

// Violation: an if statement without else may not exit
func ReturnZeroAfterPartialNestedEarlyExit(cond bool) (target.User, error) {
	_, err := findUser()
	if err == nil {
		if cond {
			return *target.NewUser("name", "email@test.com", 25), nil
		}
	}
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: an empty early exit does not exit
func ReturnZeroAfterEmptyNilCheck() (target.User, error) {
	_, err := findUser()
	if err == nil {
	}
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

func hasCode(err error, code int) bool {
	return err == nil && code == 0
}

// Violation: other functions may return true for a nil error
func ReturnZeroWithCustomErrorCheck() (target.User, error) {
	_, err := findUser()
	if hasCode(err, 0) {
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// Violation: err is reassigned by a range statement in the guarded branch
func ReturnZeroWithRangeReassignedError() (target.User, error) {
	_, err := findUser()
	if err != nil {
		for _, err = range []error{nil} {
		}
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// Violation: the nil check in a loop does not cover the return after the loop
func ReturnZeroAfterLoop() (target.User, error) {
	var err error
	for i := 0; i < 3; i++ {
		_, err = findUser()
		if err == nil {
			return *target.NewUser("name", "email@test.com", 25), nil
		}
	}
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: a zero value nested in another literal
func ReturnNestedZeroWithGuardedError() ([]target.User, error) {
	_, err := findUser()
	if err != nil {
		return []target.User{{}}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return nil, nil
}

func newError(msg string) error {
	return errors.New(msg)
}

func twoErrors() (error, error) {
	return nil, nil
}

// Sentinel facts are only for single error variables with a non-nil initializer
var errForwardSentinel = errLaterSentinel // want errForwardSentinel:`nonNilError`

var errLaterSentinel = errors.New("later") // want errLaterSentinel:`nonNilError`

var errFuncSentinel = newError("func")

var errRangeAssignedSentinel = errors.New("range assigned")

var errTupleA, errTupleB error = twoErrors()

var _ = errors.New("blank")

var nonErrorPointer = new(int)

func init() {
	for _, errRangeAssignedSentinel = range []error{nil} {
	}
	_, _, _ = errTupleA, errTupleB, nonErrorPointer
}

// OK: a sentinel error initialized with a sentinel declared later
func ReturnZeroWithForwardSentinelError() (target.User, error) {
	return target.User{}, errForwardSentinel
}

// Violation: sentinel errors that are not known to be non-nil
func ReturnZeroWithUnknownSentinelError(n int) (target.User, error) {
	switch n {
	case 0:
		return target.User{}, errFuncSentinel // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	case 1:
		return target.User{}, errRangeAssignedSentinel // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, errTupleA // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// OK: early exit in a switch case
func ReturnZeroAfterEarlyExitInCase(n int) (target.User, error) {
	_, err := findUser()
	switch n {
	case 0:
		if err == nil {
			return *target.NewUser("name", "email@test.com", 25), nil
		}
		return target.User{}, err
	}
	return *target.NewUser("name", "email@test.com", 25), nil
}

// Violation: shadowed builtins are not the builtins
func ReturnZeroWithShadowedBuiltins(n int) (target.User, error) {
	if n == 0 {
		new := func(err error) error { return err }
		return target.User{}, new(nil) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	panic := func(string) {}
	_, err := findUser()
	if err == nil {
		panic("does not exit")
	}
	return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}
