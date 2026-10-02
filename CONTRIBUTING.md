# Contributing

Most conventions here are enforced by a test or a CI job; each section names
the check that fails when a rule is broken. How to run the tests, and what the
test-only environment variables do, is in [docs/testing.md](docs/testing.md).

## Before you push

These are the commands the `test` job in
[.github/workflows/ci.yml](.github/workflows/ci.yml) runs (the last three on
Linux only):

```sh
go build ./...
go vet ./...
go test -race ./...
go test -coverprofile=cover.out -coverpkg=./... ./...
go run ./internal/tools/covercheck cover.out
go run ./internal/tools/doccheck
```

CI also checks that the generated files below are current, and runs
`golangci-lint run ./...` (v2.1.6) and `gosec ./...` (v2.29.0). Both are dev
tools that CI installs with `go install`; they are not module dependencies.

## No dependencies

Every package, examples and tools included, imports only the standard library
and this module, and `go.mod` has no `require` directive.
`TestLibraryImportsStdlibOnly` in `internal/archtest` fails otherwise. CI
actions (such as `vmactions/freebsd-vm`) and dev tools installed in CI are
not module dependencies.

## Package rules

- **Import direction.** Packages are layered in tiers, and an import may only
  point down. `TestImportDirection` in `internal/archtest` is the authoritative
  check; `.golangci.yml` mirrors it for golangci-lint. Every package under
  `internal/` needs a tier (`TestEveryInternalPackageHasATier`). The tiers are
  described in [docs/architecture/overview.md](docs/architecture/overview.md).
- **Test support stays in tests.** A non-test file must not import a
  test-support package (`TestTestHelpersStayOutOfProductionCode`).
- **No exported mutable package variables**, except `Err*` sentinels. Export a
  function that returns a copy instead (`TestNoExportedMutableVars`).
- **File size.** A hand-written non-test source file over 800 lines fails
  `TestFileSize` unless it is listed in `oversize` in
  `internal/archtest/layers_test.go` with a reason. Generated files are exempt.
- **No process-wide width toggle in library code.** Only an application may call
  `ansi.SetClusterWidth`; pass an `ansi.Measurer` instead
  (`TestNoGlobalWidthToggle`).

## Widget packages

A component is one package with a value-typed `Model` configured through
exported fields.

- No functional-option types in component packages: options belong to
  `tui.ProgramOption` (`TestComponentsHaveNoFunctionalOptions`).
- Every stateful widget's `Model` has `Linearize` and `LayoutNode` methods
  (`TestEveryStatefulWidgetHasLinearizeAndLayoutNode`). A package that
  legitimately lacks one goes in `a11yAllowlist` in
  `internal/archtest/a11y_test.go` with a real reason; the list is empty today.
- A widget that handles keys for navigation also handles the mouse
  (`TestKeyNavigationPackagesHandleMouse`), unless it is in `mouseAllowlist`
  with a reason.
- `Update` must not mutate a value copy of the Model. Add the new Model to
  `TestCopyIsolationSweep` in `internal/isolation/sweep_test.go`; the list is
  maintained by hand.

## Documentation

`go run ./internal/tools/doccheck` (CI, Linux) fails when:

- an exported identifier in a public package has no doc comment;
- a package, internal ones included, has no package comment;
- the `Experimental:` list in [doc.go](doc.go) and the packages whose comment
  says `Stability: experimental.` disagree. Mark a new experimental package in
  both places;
- a Markdown file or workflow names a dot-slash relative path or a
  `github.com/ows4444/tui/...` import path that does not exist.
  `CHANGELOG.md` is skipped.

`go test ./internal/tools/doccheck` also checks specific claims in
[README.md](README.md) against the code (colour detection variables, grapheme
clusters, unsupported platforms, the LayoutNode and Linearize claims, and that
unverified terminals and screen readers are marked so).

Every public package needs at least one runnable `Example` in a `_test.go`
file (`TestEveryPackageHasExample` in the root package, and
`TestEveryPublicPackageHasExample` in `internal/archtest`). Code shown in
Markdown should be copied from an Example so `go test` compiles it.

### Website

The site at <https://tui.nizaami.com> is an [Astro Starlight](https://starlight.astro.build/)
project in `site/`. Its npm packages are site tooling, not module dependencies.
It has no pages of its own except the landing page
(`site/src/content/docs/index.mdx`): `site/scripts/sync-docs.mjs` generates the
rest from `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md` and `docs/`, and the
Examples page from each `examples/*/testdata/size-80x24.golden`. Edit those
sources, not the generated files, which are gitignored. A page added to `docs/`
needs an entry in `pages` in the sync script and in the sidebar in
`site/astro.config.mjs`. The sync fails on a relative link that points nowhere.

For search engines and AI assistants the sync also gives each page a
descriptive `<title>` (`seoTitle` in `pages`), a meta description (the
`description` in `pages`, or else the first prose sentences), JSON-LD, and a
plain Markdown copy at `/<page>.md`, indexed by `/llms.txt` and joined in
`/llms-full.txt`. Site-wide JSON-LD and the social image tags are in
`site/astro.config.mjs`; the landing page's FAQ JSON-LD in
`site/src/content/docs/index.mdx` must match the FAQ text on that page.
`site/public/og.png` is rendered from `site/scripts/og-image.html`, whose header
comment has the command.

```console
$ cd site
$ npm ci
$ npm run dev     # http://localhost:4321, regenerates pages on start
$ npm run build   # static site in site/dist/
```

## Coverage

Each library package needs at least 90% statement coverage, counted per
package with coverage credited across packages (`-coverpkg=./...`).
`internal/tools/covercheck` skips `examples/` and `internal/tools/`, and any
`internal/` package whose non-test source has a
`//covercheck:helper <reason>` line; the reason is required.

## Generated files

Don't edit these by hand. Regenerate, then commit the result.

| File | Regenerate | CI check |
|---|---|---|
| `api.txt` | `go run ./internal/tools/apilist -w` | `apilist -check` in the `api` job |
| `.golangci.yml` | `ARCHTEST_UPDATE=1 go test ./internal/archtest -run TestGolangciMirror` | `TestGolangciMirror` in `go test ./...` |
| `ansi/runewidth_tables.go` | `go run ./internal/tools/genwidth -o ansi/runewidth_tables.go` | `widthtables` job |
| `ansi/graphemebreak_tables.go` | `go run ./internal/tools/gengrapheme -o ansi/graphemebreak_tables.go` | `widthtables` job |
| `internal/bidi/tables.go` | `go run ./internal/tools/genbidi -o internal/bidi/tables.go` | `widthtables` job |
| `testdata/*.golden` | `TUITEST_UPDATE=1` or `-update`, depending on the helper; see [docs/testing.md](docs/testing.md#golden-files) | the package's tests |

The three Unicode generators fetch pinned Unicode 17.0.0 files from
unicode.org and verify each against a SHA-256 pin, so they need network access;
`-dir` reads local copies instead. To move to a newer Unicode version, change
`unicodeVersion` in the generator and run it with `-print-hashes` to get the
new pins.

`bench/baseline.txt` is the allocation baseline for the `bench` workflow.
No script in the repository regenerates it.

`api.txt` and golden files are checked out with LF line endings on every
platform ([.gitattributes](.gitattributes)).

## Changelog

Add user-visible changes to [CHANGELOG.md](CHANGELOG.md) under
`## [Unreleased]`. Removing or changing an exported identifier needs a
`- BREAKING:` bullet in that section naming the identifier as a whole word
(`Style.Render` or `Render`), and saying what to use instead. Once a tag
exists, the `api` CI job runs
`go run ./internal/tools/apilist -since-tag -changelog CHANGELOG.md`, which
fails on a removed or changed line of `api.txt` that no such bullet names.
With no tag yet it compares nothing.

## Benchmarks

The `bench` workflow ([.github/workflows/bench.yml](.github/workflows/bench.yml))
runs on pull requests that touch a `.go` file, `bench/` or the workflow
itself, every Monday at 06:00 UTC, and on demand. It fails
when allocs/op or B/op grow more than 10% over `bench/baseline.txt`
(`internal/tools/benchgate`); ns/op is not gated. To compare locally:

```sh
go test -run '^$' -bench . -benchmem -count=6 . ./ansi ./layout ./virtuallist ./logview ./viewport ./textarea ./markdown ./datatable ./streamtext > bench/new.txt
go run ./internal/tools/benchgate -base bench/baseline.txt -new bench/new.txt
```

`bench/new.txt` is git-ignored.
