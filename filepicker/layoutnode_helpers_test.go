package filepicker

import "github.com/ows4444/tui/internal/fsutil"

func entryOf(name string, dir bool) fsutil.Entry { return fsutil.Entry{Name: name, IsDir: dir} }
