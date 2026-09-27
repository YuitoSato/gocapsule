# gocapsule

A Go linter that enforces encapsulation by preventing direct struct creation, zero value creation, type conversion, and field reassignment when `New**` constructors exist.

## Features

- **Prevent direct struct literal creation**: If a package has a `New` or `NewXxx` constructor, external packages cannot create the struct directly using struct literals
- **Prevent zero value creation** (v1.0.0+): External packages cannot create zero values (`T{}`, `&T{}`, `var v T`, `new(T)`) of types with constructors, except when returned together with a non-nil error. Use `-allowZero` or `-allowZeroPackages` to allow them
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

#### Allow Zero Values of Specific Packages (v1.0.0+)

Standard library types whose zero value is ready to use, such as `bytes.Buffer` (`var buf bytes.Buffer`), `math/big.Int` (`new(big.Int)`), and `reflect.Value`, have constructors and are therefore reported by default too. Use the `-allowZeroPackages` flag to allow zero values of types in those packages only; unlike `-ignorePackages`, struct literals with fields, type conversions, and field assignments are still reported:

```bash
gocapsule -allowZeroPackages="bytes,math/big,reflect" ./...
```

Package paths must match exactly, as with `-ignorePackages`.

### With golangci-lint

1. Create `.custom-gcl.yml`:

```yaml
version: v2.7.2
plugins:
  - module: 'github.com/YuitoSato/gocapsule'
    import: 'github.com/YuitoSato/gocapsule/gocapsule'
    version: v0.4.0
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
          # Optional: package paths whose types are allowed to be created as zero values
          allowZeroPackages:
            - bytes
            - math/big
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
4. **Supported types**: Both structs and defined types (e.g., `type Email string`) are supported
5. **Zero values** (v1.0.0+): `T{}`, `&T{}`, `var v T` (without an initializer), and `new(T)` are reported unless `-allowZero` is set or `T` is in a package listed in `-allowZeroPackages`. Type aliases of `T` are treated as `T`. Only `T` itself is checked: `var p *T` and `new(*T)` are allowed. A struct literal with any field, even `T{Name: ""}`, is a regular struct literal and is always reported
6. **Zero values returned with an error** (v1.0.0+): `T{}`, `&T{}`, and `new(T)` are allowed when they appear directly in a `return` statement together with an error result that is guaranteed to be non-nil. An error is guaranteed to be non-nil when it is:
   - `errors.New(...)` or `fmt.Errorf(...)`
   - `&x` or `new(E)`
   - a value of a concrete (non-interface) type returned as an interface, e.g. `&MyError{}`. As in Go itself, a nil `*MyError` stored in an `error` is non-nil
   - a package-level variable (e.g. a sentinel error such as `ErrNotFound` or `io.EOF`) that is initialized with one of the above or with another such variable, and is never assigned and never has its address taken in its own package
   - a local variable after a nil check, with no assignment to it between the check and the `return`:
     - in the body of `if err != nil`, `if nil != err`, or `if ... && err != nil`
     - in the `else` of `if err == nil`
     - after `if err == nil { ... }` (or `if ... || err == nil`) in the same block, when the `if` body ends with `return`, `panic`, `break`, or `continue`. A labeled statement between the check and the `return` disables this, since a `goto` could skip the check
     - after an assignment of one of the above in the same block, e.g. `err = fmt.Errorf("...: %w", err)`, `err := errors.New("...")`, or `var err error = &MyError{}`. Only single assignments are supported

     `errors.As(err, ...)` and `errors.Is(err, target)`, where `target` is one of the above, can be used in place of `err != nil`, since both are false when `err` is nil. The variable must be an interface or a pointer, and must not have its address taken or be assigned by any function literal in the enclosing function

## Migrating from v0.x

v1.0.0 reports zero values of types with constructors by default:

| Code | v0.x | v1.0.0 |
|------|------|--------|
| `users.User{}` / `&users.User{}` | Reported | Reported (allowed when returned with a non-nil error) |
| `var u users.User` | Allowed | Reported |
| `new(users.User)` | Allowed | Reported |

To keep the v0.x behavior for `var` and `new`, set `-allowZero` (or `allowZero: true` in golangci-lint). Note that `-allowZero` also allows `users.User{}` and `&users.User{}`. To allow zero values only for specific packages, such as `bytes` and `math/big`, use `-allowZeroPackages` instead.

v1.0.0 also changes the following:

- Violations through type aliases, e.g. `type U = users.User; _ = &U{}`, are reported. v0.x did not detect them
- golangci-lint `settings` (`ignorePackages`, `allowZero`, and `allowZeroPackages`) are applied. v0.x ignored them

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

A sentinel error is trusted to be non-nil only if its own package never reassigns it. Another package can still reassign an exported one (e.g. `user.ErrNotFound = nil`); such reassignments are not detected.

The following are **reported** even though the zero value is safe, because proving that requires data flow analysis. Rewrite them in one of the supported forms:

| Pattern | Workaround |
|---------|------------|
| `var u user.User` followed by an assignment before use | Declare at the first assignment |
| `var req Request; json.Unmarshal(b, &req)` | Use `-allowZero` |
| `var ErrX = newError("x")` returned as `return user.User{}, ErrX` (sentinel initialized by a function other than `errors.New` or `fmt.Errorf`, including `os.ErrNotExist` and `github.com/pkg/errors.New`) | `var ErrX = errors.New("x")`, or return a pointer: `return nil, ErrX` |
| `switch { case err != nil: return user.User{}, err }` | `if err != nil { ... }` |
| `if err != nil { ... }` in a loop, followed by `return user.User{}, err` after the loop | `if err != nil { return user.User{}, err }` in the loop |
| `return []user.User{{}}, err` (zero value nested in another literal) | Return `nil` |

## License

MIT
