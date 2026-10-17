package main

import (
	"fmt"
	"os"
)

// store is where documents live. The program uses the disk; the tests use a
// map, so no test touches the file system.
type store interface {
	read(path string) (string, error)
	write(path, text string) error
}

type diskStore struct{}

func (diskStore) read(path string) (string, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- the user names the file to open
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (diskStore) write(path, text string) error {
	return os.WriteFile(path, []byte(text), 0o600)
}

type memStore map[string]string

func (s memStore) read(path string) (string, error) {
	text, ok := s[path]
	if !ok {
		return "", fmt.Errorf("%s: no such file", path)
	}
	return text, nil
}

func (s memStore) write(path, text string) error {
	if path == "readonly.txt" {
		return fmt.Errorf("%s: permission denied", path)
	}
	s[path] = text
	return nil
}
