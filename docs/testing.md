# Testing

## Run the tests

Run one package, or one test, while you work:

```console
$ go test ./tuitest -run '^Example$'
ok  	github.com/ows4444/tui/tuitest	0.399s
```

Before you push, run what the `test` job in
[.github/workflows/ci.yml](../.github/workflows/ci.yml) runs:

```sh
go build ./...
go vet ./...
go test -race ./...
go test -coverprofile=cover.out -coverpkg=./... ./...
go run ./internal/tools/covercheck cover.out
go run ./internal/tools/doccheck
```

`covercheck` fails when a library package is under 90% statement coverage;
[CONTRIBUTING.md](../CONTRIBUTING.md#coverage) lists what it skips.

`-short` skips the slow tests: cross-compiling the module for every supported
GOOS (`TestBuildsOnSupportedPlatforms`), building every example, the 1M-line
`logview` heap test, PTY latency runs, and the wall-clock comparisons. CI does
not pass `-short`.

## Where tests live

Tests sit next to the code as `*_test.go`, in the package itself or in a
`<pkg>_test` package. Runnable Examples are in `example_test.go` in most
packages (`focus` and `hittest` keep theirs in other test files). Golden files are under the package's `testdata/`. The
module-wide rules have their own test-only packages:

| Package | What it checks |
|---|---|
| `internal/archtest` | Import direction, stdlib-only imports, file size, widget method rules, `.golangci.yml` is current |
| `internal/isolation` | `Update` on a Model never changes a value copy taken before it (`TestCopyIsolationSweep`) |
| `internal/tools/doccheck` | Doc comments, and README and `docs/` claims against the code |

The root package's `TestEveryPackageHasExample` fails when a public package has
no `Example` function.

## Testing a model with tuitest

`tuitest.New` runs a model in a real `tui.Program` against an in-memory
terminal; keys, pastes, clicks, resizes and Msgs go in, and the emulated screen
comes out. This is `Example` from
[tuitest/example_test.go](../tuitest/example_test.go):

```go
s := tuitest.New(typer{}, 20, 3)
defer s.Close()
s.Keys("h", "i")
fmt.Println(strings.TrimRight(s.Screen()[0], " "))
fmt.Printf("%q\n", s.Cell(2, 0).Grapheme)
// Output:
// > hi
// "h"
```

`tuitest.NewFakeClock` with `tuitest.WithClock` lets a test advance time by
hand (`Session.Advance`) instead of waiting on the wall clock.
`Session.Golden(t, name)` compares the screen with `testdata/<name>.golden`.
See the package godoc for the rest of `Session`.

Most widget tests don't need a Program: they call `Update` with a `tui.Key` or
other Msg and check the returned Model and `View`, as
[examples/counter/main_test.go](../examples/counter/main_test.go) does.

## Golden files

A golden test fails with a diff when the output changes. How to accept the new
output depends on the helper that wrote the golden:

| Goldens | Rewrite with |
|---|---|
| `testdata/size-<W>x<H>.golden` from `testutil.SizeMatrix` (every example) | `TUITEST_UPDATE=1 go test ./examples/<name>`, or `-update-sizes` |
| `tuitest.Session.Golden` | `TUITEST_UPDATE=1`, or `-update` where the test package declares that flag |
| Packages with their own `-update` flag (`widgets`, `internal/highlight`, several examples) | `go test ./<pkg> -update` |

`-update` on a package that doesn't declare it fails with "flag provided but
not defined"; use `TUITEST_UPDATE=1` there. Review the rewritten files in the
diff before committing. `.gitattributes` keeps `*.golden` at LF line endings on
Windows, since goldens are compared byte for byte.

## Environment variables

| Variable | Effect |
|---|---|
| `TUITEST_UPDATE=1` | Rewrite goldens, as above |
| `TUI_TIMING_TESTS=1` | Run the wall-clock tests (`frame_budget_test.go`, `e2e_bench_test.go`, `TestClockDriftFree` in `motion`, `TestStreamLinear` in `markdown`); most also skip under `-race` and `-short` |
| `ARCHTEST_UPDATE=1` | Regenerate `.golangci.yml` from the `internal/archtest` rules: `ARCHTEST_UPDATE=1 go test ./internal/archtest -run TestGolangciMirror` |
| `BIDICHARACTERTEST=<file>` | Run the full Unicode `BidiCharacterTest.txt` in `internal/bidi` instead of the built-in cases |

The library itself reads `NO_COLOR`, `TERM`, `TUI_NO_CLUSTERS`, `NO_ANIMATION`,
`REDUCE_MOTION` and others (see the README's
[Platforms and limitations](../README.md#platforms-and-limitations)), so a variable set in your shell can change what a test sees. For instance,
`TestReadmeUnicodeClaimsMatchClusterDefault` skips when `TUI_NO_CLUSTERS` is
set.

## PTY tests

Files named `*_pty_test.go`, and the tests in `term` and
`internal/cancelreader` that need a terminal, open a real pseudo-terminal
through `internal/ptytest`. They build only on Linux and macOS; on other
platforms those files are not compiled. Some re-run the test binary as a
helper child process (a test that skips with "helper process for ..." is that
child). `program_pty_test.go` skips when it can't open a pty.

On Windows, `TestProbeUnderConPTY` in `examples/probe` drives the probe
example inside a Windows pseudo console. CI runs it in its own job:

```sh
go test -count=1 -v -run 'TestProbeUnderConPTY' ./examples/probe
```

## Race detector

CI runs `go test -race ./...` on Linux, macOS and Windows. Wall-clock tests
skip under `-race`: the root package, `ansi`, `layout`, `markdown` and
`streamtext` each define a `raceEnabled` constant in `race_on_test.go` and
`race_off_test.go` for this.

## Fuzzing

Fuzz targets run their seed corpus as ordinary tests in `go test`. To fuzz one:

```sh
go test -run '^$' -fuzz '^FuzzReadEvent$' -fuzztime 60s ./input
```

| Package | Targets |
|---|---|
| root | `FuzzRendererEquivalence` (cell and line renderers draw the same screen; corpus in `testdata/fuzz/`) |
| `ansi` | `FuzzStripANSI`, `FuzzClusterWidth`, `FuzzPlainASCIIWidth`, `FuzzWidthMatchesReference`, `FuzzTruncateMatchesReference`, `FuzzASCIIFastPathMatchesSlowPath`, `FuzzDowngradeString` |
| `input` | `FuzzReadEvent` |
| `layout` | `FuzzSolveFlex`, `FuzzBlockRender` |
| `markdown` | `FuzzRender` |
| `internal/highlight` | `FuzzLines` |

No workflow runs fuzzing.

## Benchmarks

CI's `test` job runs a smoke pass on Linux:

```sh
go test -run '^$' -bench 'VirtualList10k|RenderDiff|Frame|Repaint|AvatarView|FacesView' -benchtime=100x ./...
```

The allocation gate against `bench/baseline.txt` is described in
[CONTRIBUTING.md](../CONTRIBUTING.md#benchmarks).

## CI

[.github/workflows/ci.yml](../.github/workflows/ci.yml) runs on pushes to
`code` and on pull requests. Beyond a local run, it adds:

- `test` on `ubuntu-latest`, `macos-latest` and `windows-latest`, with
  `fail-fast: false`. Coverage, `doccheck` and the benchmark smoke run on Linux
  only.
- `conpty`: the ConPTY test above, on Windows.
- `freebsd`: `go build`, `go vet` and `go test ./...` in a FreeBSD 14.2 VM.
- `api`: `api.txt` is current, and the CHANGELOG names every breaking change
  since the latest tag.
- `widthtables`: the Unicode tables match a fresh generation.
- `lint` (golangci-lint), `security` (gosec) and `staticcheck`.

[.github/workflows/bench.yml](../.github/workflows/bench.yml) runs the
allocation gate.
