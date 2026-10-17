package main

const sampleSource = `package main

import "fmt"

// greet says hello to each name.
func greet(names []string) {
    for _, n := range names {
        fmt.Println("hello,", n)
    }
}

func main() {
    greet([]string{"Ada", "Grace", "Linus"})
}
`

// sampleStore holds the files the tests open.
func sampleStore() memStore {
	return memStore{
		"main.go":      sampleSource,
		"notes.txt":    "buy milk\nbuy tea\nwrite tests\n",
		"readonly.txt": "locked\n",
	}
}
