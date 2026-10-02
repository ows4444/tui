package tui

// NewBenchProgram returns an unstarted Program of w x h cells writing to
// opts' output, with one frame already drawn, for the external end-to-end
// benchmarks that cannot reach the unexported render path.
func NewBenchProgram(m Model, w, h int, opts ...ProgramOption) *Program {
	p := NewProgram(m, opts...)
	p.width, p.height = w, h
	p.render()
	return p
}

// BenchStep delivers msg to the model as the event loop does, then draws the
// frame as the loop's per-message path does.
func (p *Program) BenchStep(msg Msg) {
	p.model, _ = p.model.Update(msg)
	p.renderIfChanged()
}
