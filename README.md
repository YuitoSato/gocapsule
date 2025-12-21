# gocapsule

A Go linter that enforces encapsulation by preventing direct struct creation and field reassignment when `New**` constructors exist.

## Features

- **Prevent direct struct literal creation**: If a package has a `NewXxx` constructor, external packages cannot create the struct directly using struct literals
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

### With golangci-lint

1. Create `.custom-gcl.yml`:

```yaml
version: v2.7.2
plugins:
  - module: 'github.com/YuitoSato/gocapsule'
    import: 'github.com/YuitoSato/gocapsule/gocapsule'
    version: v0.1.2
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
```

3. Build and run:

```bash
golangci-lint custom
./custom-gcl run ./...
```

## Example

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

## Rules

1. **Constructor pattern**: Functions matching `New[A-Z]*` that return `*StructName` or `StructName`
2. **Same package allowed**: Code within the same package can freely create structs and modify fields
3. **No constructor = no restriction**: Structs without `New**` constructors have no restrictions

## License

MIT
