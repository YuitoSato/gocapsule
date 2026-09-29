package gocapsule

import "testing"

func TestNewWithSettings(t *testing.T) {
	// Reset flags after test
	defer func() {
		ignorePackages = nil
		allowZero = false
		allowZeroWithFalseOk = false
	}()

	_, err := New(map[string]any{
		"ignorePackages":       []any{"net/http", " database/sql "},
		"allowZero":            true,
		"allowZeroWithFalseOk": true,
	})
	if err != nil {
		t.Fatalf("New() returned an error: %v", err)
	}

	if got := ignorePackages.String(); got != "database/sql,net/http" {
		t.Errorf("ignorePackages = %q, want %q", got, "database/sql,net/http")
	}
	if !allowZero {
		t.Errorf("allowZero = false, want true")
	}
	if !allowZeroWithFalseOk {
		t.Errorf("allowZeroWithFalseOk = false, want true")
	}
}

func TestNewWithoutSettings(t *testing.T) {
	if _, err := New(nil); err != nil {
		t.Fatalf("New() returned an error: %v", err)
	}

	if len(ignorePackages) != 0 {
		t.Errorf("ignorePackages = %q, want empty", ignorePackages.String())
	}
	if allowZero {
		t.Errorf("allowZero = true, want false")
	}
	if allowZeroWithFalseOk {
		t.Errorf("allowZeroWithFalseOk = true, want false")
	}
}

func TestPluginAnalyzers(t *testing.T) {
	p, err := New(nil)
	if err != nil {
		t.Fatalf("New() returned an error: %v", err)
	}

	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers() returned an error: %v", err)
	}
	if len(analyzers) != 1 || analyzers[0] != Analyzer {
		t.Errorf("BuildAnalyzers() = %v, want [Analyzer]", analyzers)
	}
	if mode := p.GetLoadMode(); mode != "typesinfo" {
		t.Errorf("GetLoadMode() = %q, want %q", mode, "typesinfo")
	}
}

func TestNewWithUnknownSettings(t *testing.T) {
	if _, err := New(map[string]any{"unknown": true}); err == nil {
		t.Fatal("New() returned no error for an unknown setting")
	}
}
