package analyzer_test

import (
	"testing"

	"gocapsule/pkg/analyzer"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	// Run tests on all test packages
	// The order matters: target must be analyzed before external
	analysistest.Run(t, testdata, analyzer.Analyzer,
		"target",
		"external",
	)
}
