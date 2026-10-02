package autocomplete

import "fmt"

func suggestionList(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("suggestion %02d", i)
	}
	return out
}
