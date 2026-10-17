module github.com/ows4444/tui/examples/ssh

go 1.26.0

require (
	github.com/ows4444/tui v0.0.0
	golang.org/x/crypto v0.58.0
)

require golang.org/x/sys v0.49.0 // indirect

// The example builds against the checkout it sits in, so it always matches the
// library it shows. A program of your own would require a released version.
replace github.com/ows4444/tui => ../..
