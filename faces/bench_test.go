package faces

import "testing"

// BenchmarkFacesView measures View at both sizes. CI runs it in the
// benchmark smoke pass.
func BenchmarkFacesView(b *testing.B) {
	for name, size := range map[string]Size{"small": Small, "large": Large} {
		m := New()
		m.Size = size
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = m.View()
			}
		})
	}
}
