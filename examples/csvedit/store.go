package main

import "os"

// store is where files live. The program uses the disk; the tests use a map,
// so no test touches the file system.
type store interface {
	read(path string) (string, error)
	write(path, text string) error
}

type diskStore struct{}

func (diskStore) read(path string) (string, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- the user names the file to open
	return string(b), err
}

func (diskStore) write(path, text string) error {
	return os.WriteFile(path, []byte(text), 0o600)
}

type memStore map[string]string

func (m memStore) read(path string) (string, error) {
	t, ok := m[path]
	if !ok {
		return "", os.ErrNotExist
	}
	return t, nil
}

func (m memStore) write(path, text string) error { m[path] = text; return nil }
