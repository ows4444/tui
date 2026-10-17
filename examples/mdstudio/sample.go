package main

// sampleDoc is what the studio opens when it is given no file.
const sampleDoc = `# Release notes

Welcome to **mdstudio**. Edit on the left and the preview on the right follows
you. Press ` + "`ctrl+b`" + ` to make the selection **bold**.

## What changed

- A new ` + "`pager`" + ` example reads files like less
- The table is *much* faster
- Mouse support everywhere

### Numbers

| Area      | Before | After |
|-----------|--------|-------|
| Frame     | 120 µs | 64 µs |
| Startup   | 11 ms  | 4 ms  |

## Try it

1. Move the cursor into another section
2. Click a heading in the outline
3. Save with ` + "`ctrl+s`" + `

> Tip: the outline lists every heading, and the preview scrolls to the one you are in.

` + "```go" + `
func main() {
	fmt.Println("hello")
}
` + "```" + `

## Links

See [the tutorial](docs/tutorial.md) or jump to [what changed](#what-changed).
`
