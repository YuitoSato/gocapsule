#!/usr/bin/env bash
#
# Compares gocapsule at a git ref (v1.0.0 by default) with the working tree,
# including uncommitted changes, on another Go module:
#
#   1. the reports of the gocapsule command
#   2. the elapsed and CPU time of the gocapsule command, which includes
#      loading and type-checking the packages
#   3. the time and memory spent in the analyzer itself (BenchmarkPackages)
#   4. optionally, the elapsed time of golangci-lint running only gocapsule
#
# Usage:
#
#   scripts/compare-repo.sh <module-dir> [base-ref]
#
# Environment variables:
#
#   PATTERNS  space-separated package patterns to analyze (default: ./...)
#   RUNS      runs of each version for the elapsed time (default: 5)
#   COUNT     -count of the benchmark for each version (default: 6)
#   GOLANGCI  a golangci-lint version, e.g. v2.12.2, to also compare custom
#             golangci-lint builds. Building them downloads golangci-lint.
#
# Run it on an idle machine: the time varies by 10% or more while other
# programs are busy. The results are kept in a temporary directory.

set -euo pipefail

if [ $# -lt 1 ]; then
	sed -n '2,/^$/s/^# \{0,1\}//p' "$0"
	exit 2
fi

target=$(cd "$1" && pwd)
base=${2:-v1.0.0}
read -r -a patterns <<<"${PATTERNS:-./...}"
runs=${RUNS:-5}
count=${COUNT:-6}
repo=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/gocapsule-compare.XXXXXX")

# The analyzer type-checks the packages with the go/types it is built with, so
# it must be built with the Go version of the module, e.g. go1.27 for a module
# that uses Go 1.27 features
goversion=$(cd "$target" && go env GOVERSION)
export GOTOOLCHAIN=$goversion

echo "Comparing gocapsule $base with the working tree on $target"
echo "Building with $goversion, the Go version of the module"
echo "Results are kept in $work"
echo

# The benchmark of the working tree also measures the base, which may not have it
mkdir -p "$work/base"
git -C "$repo" archive "$base" | tar -x -C "$work/base"
cp "$repo/gocapsule/bench_test.go" "$work/base/gocapsule/"
(cd "$work/base" && go build -o "$work/gocapsule-base" .)
(cd "$repo" && go build -o "$work/gocapsule-head" .)

# median prints the median of the numbers on stdin.
median() {
	sort -n | awk '{ v[NR] = $1 } END { if (NR % 2) print v[(NR + 1) / 2]; else print (v[NR / 2] + v[NR / 2 + 1]) / 2 }'
}

# change prints the relative change from $1 to $2.
change() {
	awk -v a="$1" -v b="$2" 'BEGIN { if (a > 0) printf "%+.1f%%", (b / a - 1) * 100; else print "-" }'
}

echo "== 1. Reports"
for v in base head; do
	# The command reports to stderr, like the errors of the packages that it
	# cannot analyze, and exits with 3 if it reports anything
	(cd "$target" && "$work/gocapsule-$v" "${patterns[@]}" 2>&1 >/dev/null || true) |
		sed "s|^$target/||" | sort >"$work/output-$v.txt"
	grep 'is not allowed' "$work/output-$v.txt" >"$work/reports-$v.txt" || true
done
echo "$base: $(wc -l <"$work/reports-base.txt" | tr -d ' ') reports, working tree: $(wc -l <"$work/reports-head.txt" | tr -d ' ') reports"
others=$(grep -vc 'is not allowed' "$work/output-head.txt" || true)
if [ "$others" -gt 0 ]; then
	echo "Warning: $others other lines, e.g. errors of packages that are not analyzed, in $work/output-head.txt:"
	grep -v 'is not allowed' "$work/output-head.txt" | head -n 5 | sed 's/^/  /'
fi
echo "-- Only in $base (e.g. false positives that are gone):"
comm -23 "$work/reports-base.txt" "$work/reports-head.txt" | head -n 30
echo "-- Only in the working tree:"
comm -13 "$work/reports-base.txt" "$work/reports-head.txt" | head -n 30
echo

# timed runs the command in "${@:2}" in the module, and appends its elapsed and
# CPU time in seconds to the file $1.
timed() {
	local out=$1
	shift
	(cd "$target" && /usr/bin/time -p sh -c '"$@" >/dev/null 2>&1' sh "$@") 2>"$work/time.txt" || true
	awk '/^real/ { r = $2 } /^user/ { u = $2 } /^sys/ { s = $2 } END { print r, u + s }' "$work/time.txt" >>"$out"
}

# compare_times prints the median elapsed and CPU time of the runs in
# $work/$1-base.txt and $work/$1-head.txt.
compare_times() {
	local b h
	for i in 1 2; do
		b=$(cut -d' ' -f$i "$work/$1-base.txt" | median)
		h=$(cut -d' ' -f$i "$work/$1-head.txt" | median)
		printf "  %-7s %s: %6.2fs   working tree: %6.2fs   %s\n" "$([ $i = 1 ] && echo elapsed || echo cpu)" "$base" "$b" "$h" "$(change "$b" "$h")"
	done
}

echo "== 2. gocapsule command ($runs runs each, alternately, after a warm-up)"
for v in base head; do
	timed /dev/null "$work/gocapsule-$v" "${patterns[@]}"
done
for _ in $(seq "$runs"); do
	for v in base head; do
		timed "$work/command-$v.txt" "$work/gocapsule-$v" "${patterns[@]}"
	done
done
compare_times command
echo

echo "== 3. Analyzer only (BenchmarkPackages, -count=$count each, in two rounds)"
failed=
for _ in 1 2; do
	for v in base head; do
		dir=$([ $v = base ] && echo "$work/base" || echo "$repo")
		if ! (cd "$dir" && GOCAPSULE_BENCH_DIR="$target" GOCAPSULE_BENCH_PATTERNS="${patterns[*]}" \
			go test -run='^$' -bench=Packages -count=$(((count + 1) / 2)) -timeout=0 ./gocapsule/) >>"$work/bench-$v.txt" 2>&1; then
			failed=$v
			break 2
		fi
	done
done
if [ -n "$failed" ]; then
	echo "  The benchmark failed. The end of $work/bench-$failed.txt:"
	tail -n 20 "$work/bench-$failed.txt" | sed 's/^/    /'
else
	grep -m 1 '^skipping' "$work/bench-head.txt" | sed 's/^/  /' || true
	for unit in gocapsule-cpu-ns/op gocapsule-ns/op gocapsule-allocs/op gocapsule-B/op ns/op; do
		b=$(awk -v u="$unit" '/^BenchmarkPackages/ { for (i = 3; i < NF; i++) if ($(i + 1) == u) print $i }' "$work/bench-base.txt" | median)
		h=$(awk -v u="$unit" '/^BenchmarkPackages/ { for (i = 3; i < NF; i++) if ($(i + 1) == u) print $i }' "$work/bench-head.txt" | median)
		printf "  %-20s %s: %14.0f   working tree: %14.0f   %s\n" "$unit" "$base" "$b" "$h" "$(change "$b" "$h")"
	done
	if command -v benchstat >/dev/null; then
		benchstat "$base=$work/bench-base.txt" "working-tree=$work/bench-head.txt"
	fi
fi
echo

if [ -n "${GOLANGCI:-}" ]; then
	echo "== 4. golangci-lint $GOLANGCI with only gocapsule ($runs runs each, alternately, without the cache)"
	for v in base head; do
		mkdir -p "$work/golangci-$v"
		cat >"$work/golangci-$v/.custom-gcl.yml" <<EOF
version: $GOLANGCI
name: custom-gcl
destination: .
plugins:
  - module: 'github.com/YuitoSato/gocapsule'
    import: 'github.com/YuitoSato/gocapsule/gocapsule'
    path: $([ $v = base ] && echo "$work/base" || echo "$repo")
EOF
		(cd "$work/golangci-$v" && golangci-lint custom)
	done
	cat >"$work/golangci.yml" <<EOF
version: "2"
linters:
  default: none
  enable:
    - gocapsule
  settings:
    custom:
      gocapsule:
        type: module
issues:
  max-issues-per-linter: 0
  max-same-issues: 0
EOF
	# The warm-up runs show that each build reports as the command does
	for v in base head; do
		mkdir -p "$work/golangci-cache/$v-0"
		(cd "$target" && GOLANGCI_LINT_CACHE="$work/golangci-cache/$v-0" "$work/golangci-$v/custom-gcl" run \
			--config "$work/golangci.yml" --issues-exit-code=0 "${patterns[@]}" 2>&1 || true) >"$work/golangci-issues-$v.txt"
		echo "  $([ $v = base ] && echo "$base" || echo "working tree"): $(grep -c '(gocapsule)$' "$work/golangci-issues-$v.txt" || true) issues"
	done
	for i in $(seq "$runs"); do
		for v in base head; do
			# Each run starts with an empty cache, so that it analyzes every package
			mkdir -p "$work/golangci-cache/$v-$i"
			timed "$work/golangci-$v.txt" env GOLANGCI_LINT_CACHE="$work/golangci-cache/$v-$i" \
				"$work/golangci-$v/custom-gcl" run --config "$work/golangci.yml" --issues-exit-code=0 "${patterns[@]}"
		done
	done
	compare_times golangci
fi
