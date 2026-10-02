//go:build darwin || linux

package ptytest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"syscall"
	"time"
)

// Config describes the program under test.
type Config struct {
	Path    string        // executable to run on a fresh pty
	Args    []string      // its arguments
	Env     []string      // environment; nil means the current one plus TERM=xterm-256color
	Cols    uint16        // initial window width (default 80)
	Rows    uint16        // initial window height (default 24)
	Timeout time.Duration // give up waiting for output after this long (default 10s)
}

// chunk is one read from the pty master with its arrival time.
type chunk struct {
	at time.Time
	n  int
}

// Session is a running program on a pty. Reads happen on a background
// goroutine so arrival times are not skewed by the caller.
type Session struct {
	cmd    *exec.Cmd
	master *os.File
	chunks chan chunk
	start  time.Time
	cfg    Config
}

// Start launches the program on a new pty. Close it when done.
func Start(cfg Config) (*Session, error) {
	if cfg.Cols == 0 {
		cfg.Cols = 80
	}
	if cfg.Rows == 0 {
		cfg.Rows = 24
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	master, slave, err := Open()
	if err != nil {
		return nil, err
	}
	if err := SetSize(master, cfg.Cols, cfg.Rows); err != nil {
		_ = master.Close()
		_ = slave.Close()
		return nil, err
	}
	cmd := exec.Command(cfg.Path, cfg.Args...) // #nosec G204 -- test harness runs a caller-chosen binary
	cmd.Env = cfg.Env
	if cmd.Env == nil {
		cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	s := &Session{cmd: cmd, master: master, chunks: make(chan chunk, 1024), cfg: cfg}
	s.start = time.Now()
	if err := cmd.Start(); err != nil {
		_ = master.Close()
		_ = slave.Close()
		return nil, err
	}
	_ = slave.Close() // the child holds its own copy
	go func() {
		buf := make([]byte, 32<<10)
		for {
			n, err := master.Read(buf)
			if n > 0 {
				select {
				case s.chunks <- chunk{at: time.Now(), n: n}:
				default: // nobody is listening; drop rather than block the reader
				}
			}
			if err != nil {
				close(s.chunks)
				return
			}
		}
	}()
	return s, nil
}

// Write sends bytes to the program's input.
func (s *Session) Write(b []byte) error {
	_, err := s.master.Write(b)
	return err
}

// Close kills the program and releases the pty.
func (s *Session) Close() {
	_ = s.cmd.Process.Kill()
	_ = s.master.Close()
	_ = s.cmd.Wait()
}

// next returns the arrival time of the next output chunk.
func (s *Session) next() (time.Time, error) {
	select {
	case c, ok := <-s.chunks:
		if !ok {
			return time.Time{}, errors.New("ptytest: program exited without output")
		}
		return c.at, nil
	case <-time.After(s.cfg.Timeout):
		return time.Time{}, errors.New("ptytest: timed out waiting for output")
	}
}

// settle drains output until none arrives for quiet.
func (s *Session) settle(quiet time.Duration) error {
	for {
		select {
		case _, ok := <-s.chunks:
			if !ok {
				return errors.New("ptytest: program exited while settling")
			}
		case <-time.After(quiet):
			return nil
		}
	}
}

// ColdStart measures the time from launching the program to its first byte
// of output (the first frame). If reply is non-nil it is called after the
// first byte, for programs that wait on a terminal response.
func ColdStart(cfg Config, reply func(*Session) error) (time.Duration, error) {
	s, err := Start(cfg)
	if err != nil {
		return 0, err
	}
	defer s.Close()
	at, err := s.next()
	if err != nil {
		return 0, err
	}
	d := at.Sub(s.start)
	if reply != nil {
		if err := reply(s); err != nil {
			return 0, err
		}
	}
	return d, nil
}

// KeyLatency starts the program, waits for its first frame and for output to
// go quiet for settle, sends key, and measures the time to the next byte.
func KeyLatency(cfg Config, key []byte, settle time.Duration) (time.Duration, error) {
	s, err := Start(cfg)
	if err != nil {
		return 0, err
	}
	defer s.Close()
	if _, err := s.next(); err != nil {
		return 0, err
	}
	if err := s.settle(settle); err != nil {
		return 0, err
	}
	sent := time.Now()
	if err := s.Write(key); err != nil {
		return 0, err
	}
	at, err := s.next()
	if err != nil {
		return 0, err
	}
	return at.Sub(sent), nil
}

// Collect runs n cold-start and n key-latency measurements.
func Collect(cfg Config, key []byte, n int, settle time.Duration) (cold, keys []time.Duration, err error) {
	for i := 0; i < n; i++ {
		d, err := ColdStart(cfg, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("cold start run %d: %w", i, err)
		}
		cold = append(cold, d)
		k, err := KeyLatency(cfg, key, settle)
		if err != nil {
			return nil, nil, fmt.Errorf("key latency run %d: %w", i, err)
		}
		keys = append(keys, k)
	}
	return cold, keys, nil
}

// Percentile returns the nearest-rank p-th percentile (0..100) of samples
// without modifying them. Empty input gives 0.
func Percentile(samples []time.Duration, p float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	s := append([]time.Duration(nil), samples...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	rank := int(p/100*float64(len(s)) + 0.999999999)
	if rank < 1 {
		rank = 1
	}
	if rank > len(s) {
		rank = len(s)
	}
	return s[rank-1]
}

// Stats summarises a set of latency samples.
type Stats struct {
	N        int
	Min, Max time.Duration
	P50, P99 time.Duration
}

// Summarize computes Stats for samples.
func Summarize(samples []time.Duration) Stats {
	st := Stats{N: len(samples)}
	if st.N == 0 {
		return st
	}
	st.Min = Percentile(samples, 0)
	st.Max = Percentile(samples, 100)
	st.P50 = Percentile(samples, 50)
	st.P99 = Percentile(samples, 99)
	return st
}

func (s Stats) String() string {
	if s.N == 0 {
		return "no samples"
	}
	return fmt.Sprintf("n=%d p50=%v p99=%v min=%v max=%v", s.N, s.P50, s.P99, s.Min, s.Max)
}
