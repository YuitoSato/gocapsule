//go:build unix

package gocapsule_test

import (
	"fmt"
	"os"
	"runtime"
	"runtime/metrics"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/YuitoSato/gocapsule/gocapsule"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// BenchmarkStdlib measures the analyzer on the standard library. The
// standard library is loaded and type-checked once, outside of the
// measurement, which takes several GB of memory. Like other benchmarks, it
// does not run without -bench, e.g. in CI.
//
// The packages are analyzed sequentially, and garbage is collected before
// each iteration. The following metrics measure the gocapsule analyzer itself,
// excluding the inspect analyzer and the driver:
//
//   - gocapsule-ns/op: the elapsed time
//   - gocapsule-cpu-ns/op: the CPU time of the process, which does not
//     include the time spent waiting for a CPU
//   - gocapsule-B/op and gocapsule-allocs/op: the allocated memory, which is
//     almost deterministic
//
// On an idle machine, the time varies by about 2% between runs, but by 10% or
// more while other programs are busy. To compare two versions, run the
// following on each of them on an idle machine, which takes about 10 seconds,
// and compare the results with benchstat (golang.org/x/perf/cmd/benchstat):
//
//	go test -run='^$' -bench=Stdlib -count=6 -benchtime=0.5s ./gocapsule/ > new.txt
//	benchstat old.txt new.txt
func BenchmarkStdlib(b *testing.B) {
	if testing.Short() {
		b.Skip("loads the whole standard library")
	}
	// cgo is disabled, so that the packages do not depend on a C toolchain
	cfg := &packages.Config{Env: append(os.Environ(), "CGO_ENABLED=0")}
	benchmarkAnalyzer(b, stdlib.load(b, cfg, "std"))
}

// BenchmarkPackages measures the analyzer on the packages of another module,
// including their tests, in the same way as BenchmarkStdlib. It runs only if
// GOCAPSULE_BENCH_DIR is set to the directory of the module, and analyzes the
// space-separated package patterns in GOCAPSULE_BENCH_PATTERNS (./... by
// default):
//
//	GOCAPSULE_BENCH_DIR=/path/to/module go test -run='^$' -bench=Packages -count=6 ./gocapsule/
func BenchmarkPackages(b *testing.B) {
	dir := os.Getenv("GOCAPSULE_BENCH_DIR")
	if dir == "" {
		b.Skip("GOCAPSULE_BENCH_DIR is not set")
	}
	patterns := strings.Fields(os.Getenv("GOCAPSULE_BENCH_PATTERNS"))
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	cfg := &packages.Config{Dir: dir, Tests: true}
	benchmarkAnalyzer(b, module.load(b, cfg, patterns...))
}

// benchmarkAnalyzer measures the analyzer on pkgs.
func benchmarkAnalyzer(b *testing.B, pkgs []*packages.Package) {
	// A copy of the analyzer that measures its own runs
	var elapsed, cpu time.Duration
	var bytes, objects uint64
	allocs := []metrics.Sample{{Name: "/gc/heap/allocs:bytes"}, {Name: "/gc/heap/allocs:objects"}}
	measured := *gocapsule.Analyzer
	measured.Run = func(pass *analysis.Pass) (any, error) {
		metrics.Read(allocs)
		startBytes, startObjects := allocs[0].Value.Uint64(), allocs[1].Value.Uint64()
		start, startCPU := time.Now(), cpuTime(b)
		defer func() {
			elapsed += time.Since(start)
			cpu += cpuTime(b) - startCPU
			metrics.Read(allocs)
			bytes += allocs[0].Value.Uint64() - startBytes
			objects += allocs[1].Value.Uint64() - startObjects
		}()
		return gocapsule.Analyzer.Run(pass)
	}
	analyzers := []*analysis.Analyzer{&measured}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		b.StopTimer()
		runtime.GC()
		b.StartTimer()
		if _, err := checker.Analyze(analyzers, pkgs, &checker.Options{Sequential: true}); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(elapsed.Nanoseconds())/float64(b.N), "gocapsule-ns/op")
	b.ReportMetric(float64(cpu.Nanoseconds())/float64(b.N), "gocapsule-cpu-ns/op")
	b.ReportMetric(float64(bytes)/float64(b.N), "gocapsule-B/op")
	b.ReportMetric(float64(objects)/float64(b.N), "gocapsule-allocs/op")
}

// loaded holds packages loaded once for all runs of a benchmark.
type loaded struct {
	once sync.Once
	pkgs []*packages.Package
	err  error
}

var stdlib, module loaded

// load loads the packages matching patterns with their syntax, once. Like
// the gocapsule command, the analysis skips the packages with errors, e.g.
// type errors, and analyzes the others.
func (l *loaded) load(b *testing.B, cfg *packages.Config, patterns ...string) []*packages.Package {
	l.once.Do(func() {
		cfg.Mode = packages.LoadAllSyntax
		l.pkgs, l.err = packages.Load(cfg, patterns...)
		errors := 0
		packages.Visit(l.pkgs, nil, func(p *packages.Package) { errors += len(p.Errors) })
		if errors > 0 {
			fmt.Fprintf(os.Stderr, "skipping the packages with errors (%d errors)\n", errors)
		}
	})
	if l.err != nil {
		b.Fatal(l.err)
	}
	if !slices.ContainsFunc(l.pkgs, func(p *packages.Package) bool { return len(p.Errors) == 0 }) {
		b.Fatal("no packages to analyze")
	}
	return l.pkgs
}

// cpuTime returns the user and system CPU time of the process, which does not
// include the time spent waiting for a CPU, unlike the elapsed time.
func cpuTime(b *testing.B) time.Duration {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		b.Fatal(err)
	}
	return time.Duration(usage.Utime.Nano() + usage.Stime.Nano())
}
