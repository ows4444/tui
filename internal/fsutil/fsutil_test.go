package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListDir(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string)
		want  []Entry
	}{
		{
			name:  "empty",
			setup: func(dir string) {},
			want:  []Entry{},
		},
		{
			name: "dirs before files, alphabetical within group",
			setup: func(dir string) {
				must(t, os.WriteFile(filepath.Join(dir, "b.txt"), nil, 0o644))
				must(t, os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o644))
				must(t, os.Mkdir(filepath.Join(dir, "zdir"), 0o755))
				must(t, os.Mkdir(filepath.Join(dir, "adir"), 0o755))
			},
			want: []Entry{
				{Name: "adir", IsDir: true},
				{Name: "zdir", IsDir: true},
				{Name: "a.txt", IsDir: false},
				{Name: "b.txt", IsDir: false},
			},
		},
		{
			name: "nested dir only lists its own level",
			setup: func(dir string) {
				must(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
				must(t, os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), nil, 0o644))
			},
			want: []Entry{{Name: "sub", IsDir: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			got, err := ListDir(dir)
			if err != nil {
				t.Fatalf("ListDir: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ListDir got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("entry %d: got %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestListDirUnreadable(t *testing.T) {
	_, err := ListDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("ListDir on a missing dir: want error, got nil")
	}
}

func TestHasExt(t *testing.T) {
	tests := []struct {
		name string
		exts []string
		want bool
	}{
		{"file.go", nil, true},
		{"file.go", []string{}, true},
		{"file.go", []string{".go"}, true},
		{"file.go", []string{".txt"}, false},
		{"file.go", []string{".txt", ".go"}, true},
		{"file", []string{".go"}, false},
	}
	for _, tt := range tests {
		if got := HasExt(tt.name, tt.exts); got != tt.want {
			t.Errorf("HasExt(%q, %v) = %v, want %v", tt.name, tt.exts, got, tt.want)
		}
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
