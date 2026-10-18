package avatar

import "testing"

func BenchmarkPNG(b *testing.B) {
	for _, size := range []int{64, 128, 256} {
		m := Model{Name: "ada", Expression: ExpressionHappy}
		b.Run(sizeLabel(size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = m.PNG(size)
			}
		})
	}
}

func sizeLabel(n int) string {
	switch n {
	case 64:
		return "64px"
	case 128:
		return "128px"
	}
	return "256px"
}
