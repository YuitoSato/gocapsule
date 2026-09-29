package externalfalseok

import (
	"errors"

	"target"
)

var users = map[string]target.User{}

func findUser(id string) (target.User, bool) {
	u, ok := users[id]
	return u, ok
}

func setTrue(ok *bool) {
	*ok = true
}

// OK: zero values returned with a constant false
func ReturnZeroWithFalse(n int) (target.User, bool) {
	switch n {
	case 0:
		return target.User{}, false
	case 1:
		return (target.User{}), (false)
	case 2:
		return target.User{}, !true
	}
	const notFound = false
	return target.User{}, notFound
}

// OK: &T{} and new(T)
func ReturnZeroPointerWithFalse(n int) (*target.User, bool) {
	if n == 0 {
		return &target.User{}, false
	}
	return new(target.User), false
}

// OK: defined type
func ReturnZeroEmailWithFalse() (*target.Email, bool) {
	return new(target.Email), false
}

// OK: the last result is the ok, whatever the number of results
func ReturnZeroWithFalseAfterKey() (string, target.User, bool) {
	return "", target.User{}, false
}

// OK: bool types without methods
type found bool

func ReturnZeroWithDefinedBool(id string) (target.User, found) {
	if _, ok := findUser(id); !ok {
		var f found
		return target.User{}, f
	}
	return target.User{}, false
}

// OK: named results
func ReturnZeroWithNamedOk() (u target.User, ok bool) {
	return target.User{}, false
}

// OK: a deferred call that only sets the named result to false
func ReturnZeroWithDeferredFalse() (u target.User, ok bool) {
	defer func() { ok = false }()
	return target.User{}, false
}

// OK: checks guaranteeing that ok is false
func ReturnZeroWithGuardedOk(id string, cond bool) (target.User, bool) {
	u, ok := findUser(id)
	if !ok {
		return target.User{}, ok
	}
	if ok == false {
		return target.User{}, (ok)
	}
	if true != ok {
		return target.User{}, ok
	}
	if cond && !ok {
		return target.User{}, ok
	}
	if !(ok || cond) {
		return target.User{}, ok
	}
	if ok {
		return u, true
	} else {
		return target.User{}, ok
	}
}

// OK: ok declared in the if statement
func ReturnZeroWithOkInIf(id string) (target.User, bool) {
	if u, ok := findUser(id); !ok {
		return target.User{}, ok
	} else {
		return u, true
	}
}

// OK: early exit unless ok is false
func ReturnZeroAfterEarlyExit(id string, cond bool) (target.User, bool) {
	u, ok := findUser(id)
	if ok {
		return u, true
	}
	if cond {
		return target.User{}, ok
	}
	if ok != false || cond {
		panic("unreachable")
	}
	return target.User{}, ok
}

// OK: assignments of false
func ReturnZeroAfterFalseAssignment(id string, n int) (target.User, bool) {
	switch n {
	case 0:
		var ok bool
		return target.User{}, ok
	case 1:
		var a, ok bool
		_ = a
		return target.User{}, ok
	case 2:
		ok := false
		return target.User{}, ok
	}
	_, ok := findUser(id)
	ok = false
	return target.User{}, ok
}

// OK: a nested function literal that only sets ok to false
func ReturnZeroWithNestedFalseAssignment(id string) (target.User, bool) {
	_, ok := findUser(id)
	reset := func() { ok = false }
	reset()
	if !ok {
		return target.User{}, ok
	}
	return target.User{}, false
}

// OK: rule 6 still applies to an error before the ok
func ReturnZeroWithErrorBeforeOk() (target.User, error, bool) {
	return target.User{}, errors.New("failed"), true
}

// OK: returned with a false ok and a non-nil error
func ReturnZeroWithFalseAndError() (target.User, bool, error) {
	return target.User{}, false, errors.New("failed")
}

// OK: declarations with a false value
func ReturnZeroWithFalseDeclaration(n int) (target.User, bool) {
	if n == 0 {
		var ok = false
		return target.User{}, ok
	}
	var ok bool = !true
	return target.User{}, ok
}

// OK: a named ok guaranteed to be false by a check
func ReturnZeroWithGuardedNamedOk(id string) (u target.User, ok bool) {
	u, ok = findUser(id)
	if !ok {
		return target.User{}, ok
	}
	return u, ok
}

// OK: function literals
func ReturnZeroInFuncLit() {
	_ = func() (target.User, bool) {
		return target.User{}, false
	}
	_ = func(id string) (*target.User, bool) {
		if _, ok := findUser(id); !ok {
			return &target.User{}, ok
		}
		return nil, false
	}
}

// Violation: the ok is true
func ReturnZeroWithTrue(id string) (target.User, bool) {
	_, ok := findUser(id)
	if ok {
		return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, true // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: the ok may be true
func ReturnZeroWithUncheckedOk(id string, cond bool) (target.User, bool) {
	_, ok := findUser(id)
	if !ok || cond {
		return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: ok may be set to true after the check
func ReturnZeroWithReassignedOk(id string, cond bool) (target.User, bool) {
	_, ok := findUser(id)
	if !ok {
		if cond {
			ok = true
		}
		return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, false
}

// Violation: the address of ok is taken
func ReturnZeroWithAddressTaken(id string) (target.User, bool) {
	_, ok := findUser(id)
	setTrue(&ok)
	if !ok {
		return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, false
}

// Violation: a nested function literal may set ok to true
func ReturnZeroWithNestedAssignment(id string) (target.User, bool) {
	_, ok := findUser(id)
	set := func() { ok = true }
	if !ok {
		set()
		return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, false
}

// Violation: a deferred call may set the named result to true after the return
func ReturnZeroWithDeferredTrue() (u target.User, ok bool) {
	defer func() { ok = true }()
	return target.User{}, false // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: bool types with methods are not tracked, since a method call
// can take the address of the variable
type flag bool

func (f *flag) set() { *f = true }

func ReturnZeroWithBoolMethods(id string) (target.User, flag) {
	var f flag
	f.set()
	if !f {
		return target.User{}, f // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return target.User{}, false
}

// Violation: the bool is not the last result
func ReturnZeroWithFalseBeforeError() (target.User, bool, error) {
	return target.User{}, false, nil // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

func ReturnZeroWithFalseFirst() (bool, target.User) {
	return false, target.User{} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: the last result is not a bool
func ReturnZeroWithZeroInt() (target.User, int) {
	return target.User{}, 0 // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: zero values that are not returned
func CreateZeroWithFalse() (target.User, bool) {
	var u target.User             // want `zero value declaration of User is not allowed; use target.NewUser\(\) instead`
	users["zero"] = target.User{} // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	return u, false
}

// Violation: unsupported checks, as in rule 6
func ReturnZeroWithUnsupportedChecks(id string, ids []string) (target.User, bool) {
	_, ok := findUser(id)
	switch {
	case !ok:
		return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	for _, id := range ids {
		if _, ok = findUser(id); ok {
			break
		}
	}
	return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: the named result is not tracked from its zero value
func ReturnZeroWithUnassignedNamedOk() (u target.User, ok bool) {
	return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: conditions that do not imply ok == false
func ReturnZeroWithUnimpliedCondition(id string, cond bool) (target.User, bool) {
	u, ok := findUser(id)
	switch {
	case cond:
		if ok == true {
			return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		}
	case !cond:
		if ok != false {
			return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		}
	default:
		if ok == cond {
			return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
		}
	}
	if ok || cond {
		return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return u, ok
}

// Violation: an early exit that does not imply ok == false after it
func ReturnZeroAfterUnimpliedEarlyExit(id string, cond bool) (target.User, bool) {
	u, ok := findUser(id)
	if !ok && cond {
		return u, true
	}
	return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: shadowed false is not the constant
func ReturnZeroWithShadowedFalse() (target.User, bool) {
	false := true
	return target.User{}, false // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: package-level variables are not tracked, even if never assigned
var notFoundVar = false

func ReturnZeroWithPackageLevelOk() (target.User, bool) {
	return target.User{}, notFoundVar // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: a deferred call may set the named result to true after the check
func ReturnZeroWithGuardedNamedOkAndDefer(id string) (u target.User, ok bool) {
	defer func() { ok = true }()
	u, ok = findUser(id)
	if !ok {
		return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
	}
	return u, ok
}

// Violation: goto may skip the early exit
func ReturnZeroAfterGoto(id string, cond bool) (target.User, bool) {
	u, ok := findUser(id)
	if cond {
		goto L
	}
	if ok {
		return u, true
	}
L:
	return target.User{}, ok // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}

// Violation: a type parameter is not a bool type, even if constrained to one
func ReturnZeroWithTypeParamOk[B ~bool]() (target.User, B) {
	return target.User{}, false // want `direct struct literal creation of User is not allowed; use target.NewUser\(\) instead`
}
