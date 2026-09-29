package gocapsule

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("gocapsule", New)
}

// Settings is the golangci-lint configuration for gocapsule.
type Settings struct {
	// IgnorePackages is a list of package paths to ignore.
	IgnorePackages []string `json:"ignorePackages"`
	// AllowZero allows zero values of encapsulated types.
	AllowZero bool `json:"allowZero"`
	// AllowZeroWithNonNilError allows zero values of encapsulated types
	// returned with a non-nil error.
	AllowZeroWithNonNilError bool `json:"allowZeroWithNonNilError"`
	// AllowZeroWithFalseOk allows zero values of encapsulated types returned
	// with a false ok as the last result.
	AllowZeroWithFalseOk bool `json:"allowZeroWithFalseOk"`
}

// New creates a new gocapsule plugin instance for golangci-lint.
func New(settings any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	ignorePackages = newPackageSet(s.IgnorePackages)
	allowZero = s.AllowZero
	allowZeroWithNonNilError = s.AllowZeroWithNonNilError
	allowZeroWithFalseOk = s.AllowZeroWithFalseOk

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
