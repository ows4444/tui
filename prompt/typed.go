package prompt

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Parse asks for a line of text and turns it into a T with parse, which is
// given the answer with the spaces at both ends removed. When parse returns an
// error its text is shown under the question and the prompt stays open, so the
// caller gets a T and never has to ask again. Config.Validate, if set, runs on
// the raw answer after parse has accepted it; Default and Placeholder work as
// for Input.
//
//	port, err := prompt.Parse("Port:", func(s string) (uint16, error) {
//		n, err := strconv.ParseUint(s, 10, 16)
//		return uint16(n), err
//	})
func Parse[T any](label string, parse func(string) (T, error), cfg ...Config) (T, error) {
	var zero T
	c := newConfig(cfg)
	var (
		last     T
		lastText string
		have     bool
		user     = c.Validate
	)
	c.Validate = func(s string) error {
		v, err := parse(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		last, lastText, have = v, s, true
		if user != nil {
			return user(s)
		}
		return nil
	}
	s, err := askText(label, false, []Config{c})
	if err != nil {
		return zero, err
	}
	if have && s == lastText {
		return last, nil // the value parse gave when the answer was accepted
	}
	return parse(strings.TrimSpace(s))
}

// Int asks for a whole number. Anything else is refused with a message and
// the question stays open.
func Int(label string, cfg ...Config) (int, error) {
	return Parse(label, func(s string) (int, error) {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, errors.New("enter a whole number")
		}
		return n, nil
	}, cfg...)
}

// Float asks for a number, with or without a decimal part.
func Float(label string, cfg ...Config) (float64, error) {
	return Parse(label, func(s string) (float64, error) {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, errors.New("enter a number")
		}
		return f, nil
	}, cfg...)
}

// Duration asks for a length of time in the form time.ParseDuration reads:
// "90s", "1h30m", "250ms".
func Duration(label string, cfg ...Config) (time.Duration, error) {
	return Parse(label, func(s string) (time.Duration, error) {
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, errors.New("enter a duration such as 90s or 1h30m")
		}
		return d, nil
	}, cfg...)
}

// Path asks for a file path. A leading "~" stands for the home directory, and
// the answer is cleaned (filepath.Clean). With mustExist the path has to name
// something that exists; otherwise any non-empty path is accepted.
func Path(label string, mustExist bool, cfg ...Config) (string, error) {
	return Parse(label, func(s string) (string, error) {
		if s == "" {
			return "", errors.New("enter a path")
		}
		p := expandHome(s)
		if mustExist {
			if _, err := os.Stat(p); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return "", errors.New("no such file or directory")
				}
				return "", err
			}
		}
		return filepath.Clean(p), nil
	}, cfg...)
}

// expandHome replaces a leading "~" or "~/" with the home directory.
func expandHome(s string) string {
	if s != "~" && !strings.HasPrefix(s, "~/") && !strings.HasPrefix(s, `~\`) {
		return s
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return s
	}
	return filepath.Join(home, s[1:])
}
