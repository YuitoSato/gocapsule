package gocapsule

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("gocapsule", New)
}

// New creates a new gocapsule plugin instance for golangci-lint.
func New(settings any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

// BuildAnalyzers returns the analyzers to run.
func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

// GetLoadMode returns the load mode required by the analyzer.
func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
