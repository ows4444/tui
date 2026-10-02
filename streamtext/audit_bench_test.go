package streamtext

import "testing"

// BenchmarkStream1kTokensInline measures streaming 1,000 tokens into the
// model with a View after each one, as an inline (non-alt-screen) UI does.
func BenchmarkStream1kTokensInline(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := wrapTestModel(80)
		m.Start()
		for j := 0; j < 1000; j++ {
			m.Append("tok ")
			if j%20 == 19 {
				m.Append("\n")
			}
			_ = m.View()
		}
	}
}
