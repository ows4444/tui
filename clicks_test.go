package tui

import (
	"io"
	"sync"
	"testing"
	"time"
)

func TestClickCounter(t *testing.T) {
	var c clickCounter
	t0 := time.Unix(1000, 0)
	press := func(x, y int, b MouseButton) MouseEvent {
		return MouseEvent{X: x, Y: y, Button: b, Action: MouseActionPress}
	}
	w := 400 * time.Millisecond
	steps := []struct {
		ev   MouseEvent
		at   time.Duration
		want int
	}{
		{press(3, 4, MouseButtonLeft), 0, 1},
		{MouseEvent{X: 3, Y: 4, Button: MouseButtonLeft, Action: MouseActionRelease}, 10 * time.Millisecond, 1},
		{press(3, 4, MouseButtonLeft), 200 * time.Millisecond, 2},
		{MouseEvent{X: 5, Y: 4, Button: MouseButtonLeft, Action: MouseActionMotion}, 210 * time.Millisecond, 2},
		{press(3, 4, MouseButtonLeft), 400 * time.Millisecond, 3},
		{press(3, 4, MouseButtonLeft), 500 * time.Millisecond, 1},   // a fourth starts again
		{press(3, 4, MouseButtonLeft), 1500 * time.Millisecond, 1},  // too slow
		{press(4, 4, MouseButtonLeft), 1600 * time.Millisecond, 1},  // another cell
		{press(4, 4, MouseButtonRight), 1700 * time.Millisecond, 1}, // another button
		{MouseEvent{X: 4, Y: 4, Button: MouseButtonWheelDown, Action: MouseActionPress}, 1710 * time.Millisecond, 0},
	}
	for i, s := range steps {
		if got := c.count1(s.ev, t0.Add(s.at), w); got != s.want {
			t.Errorf("step %d: Clicks = %d; want %d", i, got, s.want)
		}
	}
	if c.count1(press(1, 1, MouseButtonLeft), t0, 0) != 0 {
		t.Error("a window of 0 must turn counting off")
	}
}

type clickModel struct {
	mu  *sync.Mutex
	got *[]int
}

func (clickModel) Init() Cmd { return nil }
func (m clickModel) Update(msg Msg) (Model, Cmd) {
	if ev, ok := msg.(MouseEvent); ok && ev.Action == MouseActionPress {
		m.mu.Lock()
		*m.got = append(*m.got, ev.Clicks)
		m.mu.Unlock()
	}
	return m, nil
}
func (clickModel) View() string { return "x" }

func TestProgramCountsClicks(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts []ProgramOption
		want []int
	}{
		{"default", nil, []int{1, 2, 3}},
		{"off", []ProgramOption{WithMultiClick(0)}, []int{0, 0, 0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pr, pw := io.Pipe()
			defer pw.Close()
			var mu sync.Mutex
			var got []int
			opts := append([]ProgramOption{WithInput(pr), WithOutput(&unsyncBuffer{}), WithMouse(MouseCellMotion)}, tt.opts...)
			p := NewProgram(clickModel{mu: &mu, got: &got}, opts...)
			errc := make(chan error, 1)
			go func() { _, err := p.Run(); errc <- err }()
			p.Send(ResizeMsg{Width: 20, Height: 5})
			for i := 0; i < 3; i++ {
				_, _ = io.WriteString(pw, "\x1b[<0;5;3M\x1b[<0;5;3m")
			}
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				mu.Lock()
				n := len(got)
				mu.Unlock()
				if n == 3 {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			p.Quit()
			<-errc
			mu.Lock()
			defer mu.Unlock()
			if len(got) != 3 || got[0] != tt.want[0] || got[1] != tt.want[1] || got[2] != tt.want[2] {
				t.Fatalf("Clicks %v; want %v", got, tt.want)
			}
		})
	}
}
