package gocapsule_test

import (
	"testing"

	"github.com/YuitoSato/gocapsule/gocapsule"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	// Run tests on all test packages
	// The order matters: target and errs must be analyzed before external
	analysistest.Run(t, testdata, gocapsule.Analyzer,
		"target",
		"errs",
		"external",
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

	// Set the allowZero flag
	if err := gocapsule.Analyzer.Flags.Set("allowZero", "true"); err != nil {
		t.Fatalf("failed to set allowZero flag: %v", err)
	}

	// Reset flag after test
	defer func() {
		_ = gocapsule.Analyzer.Flags.Set("allowZero", "false")
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
