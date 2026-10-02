package form

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Required rejects an empty or all-whitespace value. It is the only built-in
// validator that rejects an empty value: the others pass it, so a field can be
// optional but well-formed when filled in. Pair them with Required to demand
// a value.
func Required() Validator {
	return func(v string) error {
		if strings.TrimSpace(v) == "" {
			return errors.New("required")
		}
		return nil
	}
}

// MinLen rejects a non-empty value shorter than n characters (runes).
func MinLen(n int) Validator {
	return func(v string) error {
		if v != "" && utf8.RuneCountInString(v) < n {
			return fmt.Errorf("must be at least %d characters", n)
		}
		return nil
	}
}

// MaxLen rejects a value longer than n characters (runes).
func MaxLen(n int) Validator {
	return func(v string) error {
		if utf8.RuneCountInString(v) > n {
			return fmt.Errorf("must be at most %d characters", n)
		}
		return nil
	}
}

// Email rejects a non-empty value that is not a plain address such as
// name@example.com: it must parse as an address with nothing else around it
// and a dotted domain. It is a pragmatic check, not full RFC 5322.
func Email() Validator {
	return func(v string) error {
		if v == "" {
			return nil
		}
		a, err := mail.ParseAddress(v)
		if err != nil || a.Address != v {
			return errors.New("must be a valid email address")
		}
		at := strings.LastIndexByte(v, '@')
		if !strings.Contains(v[at+1:], ".") {
			return errors.New("must be a valid email address")
		}
		return nil
	}
}

// Match rejects a non-empty value that re does not match, with msg as the
// error text. It panics if re is nil.
func Match(re *regexp.Regexp, msg string) Validator {
	if re == nil {
		panic("form: Match requires a non-nil regexp")
	}
	return func(v string) error {
		if v != "" && !re.MatchString(v) {
			return errors.New(msg)
		}
		return nil
	}
}
