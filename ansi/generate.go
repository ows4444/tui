package ansi

// The width tables in runewidth_tables.go are generated from a pinned version
// of the Unicode Character Database. Run `go generate ./ansi` to regenerate
// them; see internal/tools/genwidth.
//
//go:generate go run ../internal/tools/genwidth -o runewidth_tables.go
