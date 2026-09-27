package target_test

import (
	"testing"

	"target"
)

// OK: the external test package of target is treated as the same package
func TestTargetInternals(t *testing.T) {
	_ = &target.User{Name: "test"}
	_ = target.User{}
	var u target.User
	u.Name = "modified"
	_ = new(target.User)
	_ = target.Email("test@example.com")
}
