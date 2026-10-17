package main

import (
	"fmt"
	"os"
)

// store is where files live. The program uses the disk; the tests use a map.
type store interface {
	read(path string) ([]byte, error)
	write(path string, data []byte) error
}

const maxFile = 16 << 20 // larger files are refused: the whole file is held in memory

type diskStore struct{}

func (diskStore) read(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxFile {
		return nil, fmt.Errorf("%s is %d MiB; the editor opens files up to %d MiB", path, info.Size()>>20, maxFile>>20)
	}
	return os.ReadFile(path) // #nosec G304 G703 -- the user names the file to open
}

func (diskStore) write(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600) // an existing file keeps its permissions
}

type memStore map[string][]byte

func (s memStore) read(path string) ([]byte, error) {
	b, ok := s[path]
	if !ok {
		return nil, fmt.Errorf("%s: no such file", path)
	}
	return append([]byte(nil), b...), nil
}

func (s memStore) write(path string, data []byte) error {
	if path == "readonly.bin" {
		return fmt.Errorf("%s: permission denied", path)
	}
	s[path] = append([]byte(nil), data...)
	return nil
}

// sampleBytes is what the editor opens with no file: a GIF header, some text
// and a run of every byte value, so there is something of each kind to look at.
func sampleBytes() []byte {
	b := []byte("GIF89a")
	b = append(b, 0x40, 0x01, 0xf0, 0x00, 0xf7, 0x00, 0x00)
	b = append(b, []byte("\nThe quick brown fox jumps over the lazy dog.\n\x00\x00\x00\x00")...)
	for i := 0; i < 256; i++ {
		b = append(b, byte(i))
	}
	return b
}
