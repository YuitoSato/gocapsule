package external

import (
	"errs"
	"target"
)

// OK: Wrap returns a non-nil error when err is non-nil
func ReturnZeroWithWrappedGuardedError() (target.User, error) {
	u, err := findUser()
	if err != nil {
		return target.User{}, errs.Wrap(err, "find user")
	}
	return *u, nil
}

// OK: err is non-nil after the early return
func ReturnZeroWithWrappedErrorAfterEarlyReturn() (*target.User, error) {
	u, err := findUser()
	if err == nil {
		return u, nil
	}
	return new(target.User), errs.Wrapf(err, "find user %d", 1)
}

// OK: nested calls, functions without error parameters, methods, and
// sentinel errors initialized by functions
func ReturnZeroWithVerifiedFunctions(n int, e *errs.CodeError) (target.User, error) {
	_, err := findUser()
	switch n {
	case 0:
		if err != nil {
			return target.User{}, errs.Wrap((errs.WithDetail(err, "detail")), "find user")
		}
	case 1:
		return target.User{}, errs.New("failed")
	case 2:
		if err != nil {
			return target.User{}, errs.Wrapper{}.Wrap(err)
		}
	case 3:
		return target.User{}, errs.ErrWrapped
	case 4:
		return target.User{}, errs.Wrap(errs.New("failed"), "find user")
	case 5:
		if e != nil {
			return target.User{}, errs.WithCode(e)
		}
	case 6:
		// A nil *CodeError stored in an error is non-nil
		return target.User{}, errs.Wrap(e, "code")
	case 7:
		if err != nil {
			return target.User{}, errs.Copy(err)
		}
	case 8:
		if err != nil {
			return target.User{}, errs.WrapGeneric(err, 0)
		}
	case 9:
		if err != nil {
			return target.User{}, errs.Box[int]{}.Wrap(err)
		}
	case 10:
		if err != nil {
			return target.User{}, errs.Wrapper.Wrap(errs.Wrapper{}, err)
		}
	case 11:
		// Only the first argument must be non-nil
		if err != nil {
			return target.User{}, errs.Combine(err, nil)
		}
	}
	return target.User{}, errs.ErrFromHelper
}

// OK: error parameters that do not have to be non-nil
func ReturnZeroWithOptionalErrorArguments(n int) (target.User, error) {
	_, err := findUser()
	switch n {
	case 0:
		return target.User{}, errs.NewCoded("not_found")
	case 1:
		return target.User{}, errs.Translate(err, "not_found")
	}
	return target.User{}, errs.WrapWithFuncLitReturn(nil)
}

// OK: conversions of non-nil values to an interface
func ReturnZeroWithConversion(n int) (target.User, error) {
	_, err := findUser()
	switch n {
	case 0:
		return target.User{}, error(&myError{})
	case 1:
		if err != nil {
			return target.User{}, error(err)
		}
	}
	return target.User{}, errs.Wrap(error(errs.ErrNotFound), "converted")
}

// Violation: err may be nil, so Wrap may return nil
func ReturnZeroWithWrappedUnguardedError() (target.User, error) {
	u, err := findUser()
	if u == nil {
		return target.User{}, errs.Wrap(err, "not found") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return *u, nil
}

// Violation: the arguments are nil
func ReturnZeroWithWrappedNil(n int) (target.User, error) {
	switch n {
	case 0:
		return target.User{}, errs.Wrap(nil, "failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	case 1:
		return target.User{}, errs.WithCode(nil) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	var err error
	return target.User{}, errs.Wrapf(err, "failed") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: functions that are not verified
func ReturnZeroWithUnverifiedFunctions(n int) (target.User, error) {
	_, err := findUser()
	if err != nil {
		switch n {
		case 0:
			return target.User{}, errs.MaybeWrap(err, false) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		case 1:
			return target.User{}, errs.WrapLookup(err) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		case 2:
			return target.User{}, errs.Combine(nil, err) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		case 3:
			return target.User{}, error(nil) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		case 4:
			return target.User{}, errs.Wrapper.Wrap(errs.Wrapper{}, nil) // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		}
	}
	return target.User{}, nil // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: a function value is not the function
func ReturnZeroWithWrapFuncValue() (target.User, error) {
	wrap := errs.Wrap
	_, err := findUser()
	if err != nil {
		return target.User{}, wrap(err, "find user") // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, nil // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: the argument is not known to be non-nil at the assignment, so
// this is not a non-nil assignment
func ReturnZeroWithWrapAssignment() (target.User, error) {
	_, err := findUser()
	if err != nil {
		err = errs.Wrap(err, "find user")
		return target.User{}, err // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, nil // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// OK: the argument is always non-nil, so this is a non-nil assignment
func ReturnZeroWithNonNilWrapAssignment() (target.User, error) {
	err := errs.Wrap(errs.ErrNotFound, "find user")
	return target.User{}, err
}
