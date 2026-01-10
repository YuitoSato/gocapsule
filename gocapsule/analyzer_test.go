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

func TestAnalyzerWithIgnorePackages(t *testing.T) {
	testdata := analysistest.TestData()

	// Set the ignorePackages flag
	if err := gocapsule.Analyzer.Flags.Set("ignorePackages", "ignored"); err != nil {
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
