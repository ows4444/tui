package commandpalette

import "fmt"

func commandList(n int) []Command {
	out := make([]Command, n)
	for i := range out {
		out[i] = Command{Name: fmt.Sprintf("cmd%02d", i), Description: "does something"}
	}
	return out
}
