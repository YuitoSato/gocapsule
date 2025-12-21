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
