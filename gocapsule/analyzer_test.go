package gocapsule_test

import (
	"testing"

	"github.com/YuitoSato/gocapsule/gocapsule"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	// Run tests on all test packages
	// The order matters: target must be analyzed before external
	analysistest.Run(t, testdata, gocapsule.Analyzer,
		"target",
		"external",
	)
}

func TestAnalyzerWithAllowZeroWithNonNilError(t *testing.T) {
	testdata := analysistest.TestData()

	// Set the allowZeroWithNonNilError flag
	if err := gocapsule.Analyzer.Flags.Set("allowZeroWithNonNilError", "true"); err != nil {
		t.Fatalf("failed to set allowZeroWithNonNilError flag: %v", err)
	}

	// Reset flag after test
	defer func() {
		_ = gocapsule.Analyzer.Flags.Set("allowZeroWithNonNilError", "false")
	}()

	// Run tests - zero values returned with a non-nil error should be allowed
	// The order matters: errs must be analyzed before externalnonnilerror
	analysistest.Run(t, testdata, gocapsule.Analyzer,
		"errs",
		"externalnonnilerror",
	)
}

func TestAnalyzerWithIgnorePackages(t *testing.T) {
	testdata := analysistest.TestData()

	// Set the ignorePackages flag. Package paths are trimmed and must match
	// exactly: "targ" does not match "target"
	if err := gocapsule.Analyzer.Flags.Set("ignorePackages", "targ, ignored"); err != nil {
		t.Fatalf("failed to set ignorePackages flag: %v", err)
	}

	// Reset flag after test
	defer func() {
		_ = gocapsule.Analyzer.Flags.Set("ignorePackages", "")
	}()

	// Run tests - violations in "ignored" package should be skipped
	analysistest.Run(t, testdata, gocapsule.Analyzer,
		"ignored",
		"externalwithignore",
	)
}

func TestAnalyzerWithAllowZero(t *testing.T) {
	testdata := analysistest.TestData()

	// Set the allowZero flag. allowZeroWithNonNilError is set too, to check
	// that allowZero skips the facts that it needs
	for _, name := range []string{"allowZero", "allowZeroWithNonNilError"} {
		if err := gocapsule.Analyzer.Flags.Set(name, "true"); err != nil {
			t.Fatalf("failed to set %s flag: %v", name, err)
		}
	}

	// Reset flags after test
	defer func() {
		_ = gocapsule.Analyzer.Flags.Set("allowZero", "false")
		_ = gocapsule.Analyzer.Flags.Set("allowZeroWithNonNilError", "false")
	}()

	// Run tests - zero values of encapsulated types should be allowed
	analysistest.Run(t, testdata, gocapsule.Analyzer,
		"externalallowzero",
	)
}

func TestAnalyzerWithAllowZeroWithFalseOk(t *testing.T) {
	testdata := analysistest.TestData()

	// Set the allowZeroWithFalseOk flag
	if err := gocapsule.Analyzer.Flags.Set("allowZeroWithFalseOk", "true"); err != nil {
		t.Fatalf("failed to set allowZeroWithFalseOk flag: %v", err)
	}

	// Reset flag after test
	defer func() {
		_ = gocapsule.Analyzer.Flags.Set("allowZeroWithFalseOk", "false")
	}()

	// Run tests - zero values returned with a false ok should be allowed
	analysistest.Run(t, testdata, gocapsule.Analyzer,
		"externalfalseok",
	)
}
