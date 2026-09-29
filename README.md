# gocapsule

A Go linter that enforces encapsulation by preventing direct struct creation, zero value creation, type conversion, and field reassignment when `New**` constructors exist.

## Features

- **Prevent direct struct literal creation**: If a package has a `New` or `NewXxx` constructor, external packages cannot create the struct directly using struct literals
- **Prevent zero value creation** (v1.0.0+): External packages cannot create zero values (`T{}`, `&T{}`, `var v T`, `new(T)`) of types with constructors, except when returned together with a non-nil error. Use `-allowZero` to allow them
- **Prevent direct type conversion**: For defined types (e.g., `type Email string`) with constructors, external packages cannot use direct type conversions
- **Prevent field reassignment**: External packages cannot reassign public fields of structs that have constructors
- **Embedded field support**: Detects violations through embedded field access (e.g., `container.User.Name = "x"`)

## Installation

```bash
go install github.com/YuitoSato/gocapsule@latest
```

## Usage

### Standalone

```bash
gocapsule ./...
```

### With Flags

#### Ignore Specific Packages

Use the `-ignorePackages` flag to exclude specific packages from analysis. This is useful for ignoring standard library packages like `net/http` that have constructors but are used in legitimate ways.

```bash
gocapsule -ignorePackages="net/http,database/sql" ./...
```

gocapsule analyzes your dependencies as well, and `New()` is the idiomatic constructor name in many libraries. Types such as `container/list.List`, `container/ring.Ring`, `reflect.Value`, `log.Logger`, `text/template.Template`, `echo.Echo`, `gin.Engine`, and `logrus.Logger` are therefore encapsulated too. If gocapsule reports legitimate usage of such a type (for example `r.Value = 1` on a `ring.Ring`, or `e.Debug = true` on an `echo.Echo`), exclude that package:

```bash
gocapsule -ignorePackages="container/ring,github.com/labstack/echo/v4" ./...
```

Package paths must match exactly; prefixes and globs are not supported.

#### Allow Zero Values (v1.0.0+)

By default, zero values of types with constructors are reported. Use the `-allowZero` flag to allow them:

```bash
gocapsule -allowZero ./...
```

| Code | Default | `-allowZero` |
|------|---------|--------------|
| `users.User{}` | ❌ Reported | ✅ Allowed |
| `&users.User{}` | ❌ Reported | ✅ Allowed |
| `var u users.User` | ❌ Reported | ✅ Allowed |
| `new(users.User)` | ❌ Reported | ✅ Allowed |
| `&users.User{Name: ""}` | ❌ Reported | ❌ Reported |
| `email.Email("")` | ❌ Reported | ❌ Reported |

See [Zero Values](#zero-values-v100) for examples, including zero values that are allowed because they are returned with a non-nil error.

Standard library types whose zero value is ready to use, such as `bytes.Buffer` (`var buf bytes.Buffer`), `math/big.Int` (`new(big.Int)`), and `reflect.Value`, have constructors and are therefore reported by default too. To allow them while keeping zero values of your own types reported, ignore those packages instead of using `-allowZero`. These types have no exported fields, so ignoring their packages loses no other checks:

```bash
gocapsule -ignorePackages="bytes,math/big,reflect" ./...
```

### With golangci-lint

1. Create `.custom-gcl.yml`:

```yaml
version: v2.7.2
plugins:
  - module: 'github.com/YuitoSato/gocapsule'
    import: 'github.com/YuitoSato/gocapsule/gocapsule'
    version: v1.0.0
```

2. Add to `.golangci.yml`:

```yaml
linters:
  enable:
    - gocapsule
  settings:
    custom:
      gocapsule:
        type: "module"
        # Settings are supported since v1.0.0
        settings:
          # Optional: allow zero values (default: false)
          allowZero: false
          # Optional: package paths to ignore
          ignorePackages:
            - net/http
            - database/sql
```

3. Build and run:

```bash
golangci-lint custom
./custom-gcl run ./...
```

## Example

### Structs

Given a package with a constructor:

```go
// package user
type User struct {
    Name  string
    Email string
}

func NewUser(name, email string) *User {
    return &User{Name: name, Email: email}
}
```

The following code in an external package will be flagged:

```go
// package main
import "user"

func main() {
    // NG: direct struct literal creation
    u := &user.User{Name: "test"}
    // -> "direct struct literal creation of User is not allowed; use user.NewUser() instead"

    // OK: using constructor
    u := user.NewUser("test", "test@example.com")

    // NG: field reassignment
    u.Name = "modified"
    // -> "direct field assignment to User.Name is not allowed; User has a constructor NewUser()"
}
```

A constructor named just `New` works the same way. This is convenient when the package is named after the type:

```go
// package repository
type Repository struct {
    Name string
}

func New(name string) *Repository {
    return &Repository{Name: name}
}
```

```go
// package main
import "repository"

func main() {
    // NG: direct struct literal creation
    r := &repository.Repository{Name: "test"}
    // -> "direct struct literal creation of Repository is not allowed; use repository.New() instead"

    // OK: using constructor
    r := repository.New("test")
}
```

### Zero Values (v1.0.0+)

Zero values of types with constructors are not allowed either, since they skip the validation in the constructor:

```go
// package main
import "user"

func main() {
    // NG: empty struct literal
    u := &user.User{}
    // -> "direct struct literal creation of User is not allowed; use user.NewUser() instead"

    // NG: var declaration without initializer
    var u user.User
    // -> "zero value declaration of User is not allowed; use user.NewUser() instead"

    // NG: new
    u := new(user.User)
    // -> "zero value creation of User with new() is not allowed; use user.NewUser() instead"

    // OK: pointer declaration (the zero value is nil, not a User)
    var u *user.User
}
```

A zero value returned together with an error that is guaranteed to be non-nil is allowed, because the caller cannot use it without ignoring the error:

```go
var ErrNotFound = errors.New("not found")

func FindUser(id string) (user.User, error) {
    u, err := repo.Find(id)
    if err != nil {
        // OK: err is non-nil here
        return user.User{}, err
    }
    if u == nil {
        // OK: ErrNotFound is initialized with errors.New and never reassigned
        return user.User{}, ErrNotFound
    }
    return *u, nil
}

func FindUserByEmail(email string) (user.User, error) {
    u, err := repo.FindByEmail(email)
    if err == nil {
        return *u, nil
    }
    // OK: err is non-nil here because of the early return
    return user.User{}, err
}

func Broken() (user.User, error) {
    // NG: the error may be nil
    return user.User{}, nil
}
```

Error helpers are trusted too (v1.1.0+), including your own and those of libraries such as `github.com/pkg/errors`, as long as gocapsule can verify that they return a non-nil error for non-nil error arguments (rule 7). A call with an error argument that may be nil is still reported:

```go
// package errs
func Wrap(err error, msg string) error {
    if err == nil {
        return nil
    }
    return &wrapError{cause: err, msg: msg}
}
```

```go
func FindUserByName(name string) (user.User, error) {
    u, err := repo.FindByName(name)
    if err != nil {
        // OK: Wrap returns a non-nil error because err is non-nil here
        return user.User{}, errs.Wrap(err, "find user")
    }
    if u == nil {
        // NG: err is nil here, so Wrap returns nil
        return user.User{}, errs.Wrap(err, "user not found")
    }
    return *u, nil
}
```

The exact conditions are listed in rules 5 to 7 of [Rules](#rules). See [Limitations](#limitations) for zero values that are not detected, and for safe code that is still reported.

### Defined Types

Defined types with constructors are also protected:

```go
// package email
type Email string

func NewEmail(s string) (Email, error) {
    // validate email format
    return Email(s), nil
}
```

```go
// package main
import "email"

func main() {
    // NG: direct type conversion
    e := email.Email("test@example.com")
    // -> "direct type conversion to Email is not allowed; use email.NewEmail() instead"

    // OK: using constructor
    e, err := email.NewEmail("test@example.com")
}
```

## Rules

1. **Constructor pattern**: Package-level functions named exactly `New`, or `New` followed by the type name (case-insensitive, e.g. `NewUser` for `User`, `NewHTTPClient` for `HTTPClient`), whose first return value is `*TypeName` or `TypeName` of a type declared in the same package. Additional return values such as `error` are ignored, so `NewEmail() (Email, error)` and `New() (*Repository, error)` count as constructors
2. **Same package allowed**: Code within the same package can freely create types and modify fields
3. **No constructor = no restriction**: Types without `New**` constructors have no restrictions
4. **Supported types**: Both structs and defined types (e.g., `type Email string`) are supported. Interfaces are not (v1.0.0+): a constructor returning an interface, e.g. `NewStore() Store`, does not restrict `Store`, since an interface has nothing to encapsulate
5. **Zero values** (v1.0.0+): `T{}`, `&T{}`, `var v T` (without an initializer), and `new(T)` are reported unless `-allowZero` is set. Type aliases of `T` are treated as `T`. Only `T` itself is checked: `var p *T` and `new(*T)` are allowed. A struct literal with any field, even `T{Name: ""}`, is a regular struct literal and is always reported
6. **Zero values returned with an error** (v1.0.0+): `T{}`, `&T{}`, and `new(T)` are allowed when they appear directly in a `return` statement together with an error result that is guaranteed to be non-nil.

   The following are **non-nil expressions**:
   - `errors.New(...)` or `fmt.Errorf(...)`
   - `&x` or `new(E)`, e.g. `&MyError{}`
   - a value of a concrete (non-interface) type returned as an interface, e.g. `NewAppError(...)` returning `*AppError`. As in Go itself, a nil `*AppError` stored in an `error` is non-nil. Any other function returning `error` is trusted only as described in the next item
   - (v1.1.0+) a call to a verified function (rule 7) whose arguments for the error parameters it requires to be non-nil are non-nil expressions, e.g. `errs.New("...")`, `errs.Wrap(ErrNotFound, "...")`, or `github.com/pkg/errors.New("...")`
   - (v1.1.0+) a conversion of a non-nil expression to an interface or pointer type, e.g. `error(&MyError{})`
   - a package-level variable (e.g. a sentinel error such as `ErrNotFound` or `io.EOF`) that is initialized with a non-nil expression, and in its own package is only assigned by non-nil assignments and never has its address taken

   A **non-nil assignment** is a single assignment of a non-nil expression, e.g. `err = fmt.Errorf("...: %w", err)`, `err := errors.New("...")`, or `var err error = &MyError{}`. A multiple assignment such as `n, err = 0, errors.New("...")` is not.

   A local variable `err` is also guaranteed to be non-nil at the `return`:
   - in the body of `if err != nil`, `if nil != err`, or `if ... && err != nil`
   - in the `else` of `if err == nil`
   - after `if err == nil { ... }` (or `if ... || err == nil`) in the same block, when the `if` body ends with `return`, `panic`, `break`, or `continue`. A labeled statement between the check and the `return` disables this, since a `goto` could skip the check
   - after a non-nil assignment in the same block

   `errors.As(err, ...)` and `errors.Is(err, target)`, where `target` is a non-nil expression, can be used in place of `err != nil`, since both are false when `err` is nil. In all of these cases:
   - `err` must be an interface or a pointer
   - `err` must only be assigned by non-nil assignments between the check and the `return`
   - `err` must never have its address taken, and function literals nested in the function that declares `err` must only assign it by non-nil assignments

   If the error result is named, e.g. `func f() (u user.User, err error)`, the last condition also applies to the named result `err`, even when the `return` statement returns `errors.New(...)`, because a deferred call can overwrite a named result after the `return`, e.g. `defer func() { err = nil }()`

   (v1.1.0+) Directly in the `return` statement, the operand of a conversion, or an argument for an error parameter that a verified function requires to be non-nil, can also be a local variable guaranteed to be non-nil as described above, even in a nested call, e.g. `errs.Wrap(err, "...")` or `errs.Wrap(errs.WithCode(err, 404), "...")` inside `if err != nil`.

7. **Verified functions** (v1.1.0+): a function is verified if gocapsule can prove that it returns a non-nil error whenever certain error parameters (interfaces and pointers that implement `error`), possibly none, are non-nil. gocapsule verifies the functions and methods whose only result is an interface that implements `error`, including generic ones and those in your dependencies, and records which error parameters must be non-nil. A function that needs none of them, such as `errs.New(msg string) error`, always returns a non-nil error. Functions that return a concrete type such as `*MyError` are already non-nil expressions by rule 6.

   Unlike rule 6, gocapsule follows every path through the function. It assumes that the error parameters are non-nil at the start, and tracks which local variables of interface and pointer types are non-nil:
   - a variable is non-nil after it is assigned a value that is non-nil at that point, e.g. `err = errs.WithStack(err)` for a non-nil `err`, and in the branch of a nil check, e.g. `if err != nil`, `if err == nil { ... } else`, or `switch { case err != nil: }`
   - a variable is non-nil after a branch only if it is non-nil at the end of every path that reaches it, e.g. when it is assigned a non-nil value in both the `if` and the `else`
   - a branch that requires a non-nil variable to be nil is never taken, e.g. `if err == nil { return nil }`
   - a variable is not tracked if its address is taken or a nested function literal may assign it a possibly nil value, since these may change it at any time, or if it is assigned by `for ... = range`

   Every reachable `return` statement, including a bare `return` of the named result, must return a non-nil value. An error parameter must be non-nil at the call site only if the proof needs it, e.g. `Coded(404, nil)` below is non-nil. A function with a `defer` statement is not verified, because a deferred call may recover from a panic, and the function then returns a nil error. A function that calls itself, directly or through other functions, is not verified either. A verified function may call other verified functions and return non-nil sentinel errors.

   The following were checked with Go 1.26:
   - Verified: the constructors and wrappers of `github.com/pkg/errors`, `github.com/cockroachdb/errors`, `github.com/morikuni/failure/v2`, `github.com/rotisserie/eris`, `github.com/samber/oops`, and `golang.org/x/xerrors`. The functions that initialize standard library sentinel errors such as `os.ErrNotExist` are verified too, so these sentinel errors are non-nil
   - Not verified: `github.com/morikuni/failure` v1 (applies wrappers through an interface method), `errors.Join` and `go.uber.org/multierr` (may return nil), and `google.golang.org/grpc/status.Error` (returns nil for `codes.OK`)

   ```go
   // Verified: err is non-nil after both branches and the reassignment
   func Newf(format string, args ...any) error {
       var err error
       if len(args) > 0 {
           err = &formatError{format: format, args: args}
       } else {
           err = errors.New(format)
       }
       err = WithCode(err, 500)
       return err
   }

   // Verified: the last return is never reached when err is non-nil
   func WithCode(err error, code int) error {
       if err != nil {
           return &codeError{cause: err, code: code}
       }
       return nil
   }

   // Verified: always non-nil, so cause may be nil
   func Coded(code int, cause error) error {
       return &codeError{cause: cause, code: code}
   }

   // Not verified: an interface method may return nil
   func Apply(err error, h Handler) error {
       if err == nil {
           return nil
       }
       return h.Handle(err)
   }
   ```

## Migrating from v0.x

v1.0.0 reports zero values of types with constructors by default:

| Code | v0.x | v1.0.0 |
|------|------|--------|
| `users.User{}` / `&users.User{}` | Reported | Reported (allowed when returned with a non-nil error) |
| `var u users.User` | Allowed | Reported |
| `new(users.User)` | Allowed | Reported |

To keep the v0.x behavior for `var` and `new`, set `-allowZero` (or `allowZero: true` in golangci-lint). Note that `-allowZero` also allows `users.User{}` and `&users.User{}`. To allow zero values only for standard library types such as `bytes.Buffer` and `big.Int`, ignore their packages with `-ignorePackages` instead.

v1.0.0 also changes the following:

- Violations through type aliases, e.g. `type U = users.User; _ = &U{}`, are reported. v0.x did not detect them
- golangci-lint `settings` (`ignorePackages` and `allowZero`) are applied. v0.x ignored them. Settings are decoded strictly: an unknown key or a value of the wrong type, e.g. `ignorePackages: "net/http"` instead of a list, fails at startup
- Interfaces returned by a constructor are no longer restricted, e.g. `store.Store(impl)` is not reported when `store.NewStore()` returns the interface `Store`

## Limitations

gocapsule enforces constructor usage and blocks **field reassignment**, but does **not** detect content mutation of slices, maps, or pointers:

| Pattern | Detected? |
|---------|-----------|
| `u := &user.User{Name: "x"}` | ✅ Yes |
| `u.Name = "modified"` | ✅ Yes |
| `e := email.Email("invalid")` | ✅ Yes |
| `cart.Order.Amount = 0` (embedded) | ✅ Yes |
| `u.Roles[0] = "hacker"` (slice element) | ❌ No |
| `c.Settings["key"] = "value"` (map value) | ❌ No |
| `roles[0] = "x"` after `NewUser(roles)` | ❌ No |
| `dept.Manager.Salary = 0` (pointer field) | ❌ No |

Zero values are detected syntactically, without data flow analysis. The following zero values are **not** detected:

| Pattern | Detected? |
|---------|-----------|
| `func f() (u user.User, err error)` (named result) | ❌ No |
| `make([]user.User, n)`, `var a [3]user.User` | ❌ No |
| `var w Wrapper` where `Wrapper` has a `user.User` field | ❌ No |
| `u := m[key]` (missing map key), `u, _ := x.(user.User)`, `u := <-ch` | ❌ No |
| `var zero T` / `*new(T)` in generic code instantiated with `user.User` | ❌ No |
| `reflect.Zero`, `reflect.New` | ❌ No |

A sentinel error is trusted to be non-nil only if its own package never assigns it a possibly nil value. Another package can still reassign an exported one (e.g. `user.ErrNotFound = nil`); such reassignments are not detected.

The following are **reported** even though the zero value is safe. The data flow analysis of rule 7 only verifies error helpers: the `return` statement that returns a zero value is still checked syntactically by rule 6. Rewrite them in one of the supported forms:

| Pattern | Workaround |
|---------|------------|
| `var u user.User` followed by an assignment before use | Declare at the first assignment |
| `var req Request; json.Unmarshal(b, &req)` | Use `-allowZero` |
| `var ErrX = newError("x")` returned as `return user.User{}, ErrX` (sentinel initialized by a function that is not verified by rule 7 of [Rules](#rules)) | `var ErrX = errors.New("x")`, or return a pointer: `return nil, ErrX` |
| `return user.User{}, newError("x")` where `newError` always returns a non-nil error but is not verified by rule 7 of [Rules](#rules), e.g. it calls an interface method or uses `defer` | Rewrite `newError` in a verified form, return a concrete type from it, e.g. `func newError(msg string) *MyError`, or return a pointer: `return nil, newError("x")` |
| `err = errs.Wrap(err, "...")` followed by `return user.User{}, err` (a guarded `err` counts only as an argument of a call directly in the `return`, not in an assignment) | `return user.User{}, errs.Wrap(err, "...")` |
| `switch { case err != nil: return user.User{}, err }` | `if err != nil { ... }` |
| `if err != nil { ... }` in a loop, followed by `return user.User{}, err` after the loop | `if err != nil { return user.User{}, err }` in the loop |
| `return []user.User{{}}, err` (zero value nested in another literal) | Return `nil` |
| `if cond { err = f() } else { return user.User{}, err }` after an early exit (branches are not distinguished, so a possibly nil value assigned in another branch counts) | `if err != nil { return user.User{}, err }` in the branch |
| A deferred function that may set the named error result to nil, e.g. `defer func() { if cerr := f.Close(); cerr != nil && err == nil { err = cerr } }()` (a variable on the right-hand side is not treated as non-nil, even after a nil check) or `err = errors.Join(err, f.Close())`. This applies to every `return` in the function, including ones before the `defer` | Assign a non-nil expression: `err = fmt.Errorf("close: %w", cerr)`, or return a pointer: `return nil, err` |

Returning a concrete type does not suit a helper that returns nil for a nil error, such as `Wrap`: its nil `*MyError` stored in an `error` is non-nil (rule 6), so `if err != nil` would be true for it.

## License

MIT
