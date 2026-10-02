package tui

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// styledScreen builds a cols x rows view of SGR-styled rows; tick changes one
// digit per row so a frame differs from the last.
func styledScreen(cols, rows, tick int) string {
	var sb strings.Builder
	for r := 0; r < rows; r++ {
		if r > 0 {
			sb.WriteByte('\n')
		}
		n := 0
		for n < cols {
			seg := "seg" + strconv.Itoa((r+tick)%10) + " "
			if n+len(seg) > cols {
				seg = strings.Repeat("-", cols-n)
			}
			sb.WriteString("\x1b[3" + strconv.Itoa(1+(n/8)%6) + "m" + seg + "\x1b[0m")
			n += len(seg)
		}
	}
	return sb.String()
}

func benchCellProgram(b *testing.B, cols, rows int, v string) *Program {
	b.Helper()
	out, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("open %s: %v", os.DevNull, err)
	}
	b.Cleanup(func() { out.Close() })
	p := NewProgram(staticModel{view: v}, WithOutput(out), WithCellRenderer(true))
	p.width, p.height = cols, rows
	p.render()
	return p
}

// BenchmarkFrame200x60Styled measures a steady-state cell-renderer frame at
// 200x60 where every row's styled content changes.
func BenchmarkFrame200x60Styled(b *testing.B) {
	views := [2]string{styledScreen(200, 60, 0), styledScreen(200, 60, 1)}
	p := benchCellProgram(b, 200, 60, views[0])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.model = staticModel{view: views[(i+1)%2]}
		p.render()
	}
}

// BenchmarkFrame300x80Styled is BenchmarkFrame200x60Styled at 300x80.
func BenchmarkFrame300x80Styled(b *testing.B) {
	views := [2]string{styledScreen(300, 80, 0), styledScreen(300, 80, 1)}
	p := benchCellProgram(b, 300, 80, views[0])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.model = staticModel{view: views[(i+1)%2]}
		p.render()
	}
}

// BenchmarkOneCellChange200x60 measures a cell-renderer frame at 200x60 where
// a single cell differs from the previous frame.
func BenchmarkOneCellChange200x60(b *testing.B) {
	base := styledScreen(200, 60, 0)
	alt := strings.Replace(base, "seg0", "segX", 1)
	views := [2]string{base, alt}
	p := benchCellProgram(b, 200, 60, views[0])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.model = staticModel{view: views[(i+1)%2]}
		p.render()
	}
}
