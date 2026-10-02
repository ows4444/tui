package ansi

import (
	"strings"
	"testing"
)

func TestHyperlink(t *testing.T) {
	tests := []struct {
		name, text, url string
	}{
		{"simple", "click me", "https://example.com"},
		{"empty text", "", "https://example.com"},
		{"query string url", "docs", "https://example.com/x?a=1&b=2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Hyperlink(test.text, test.url)
			want := "\x1b]8;;" + test.url + "\x1b\\" + test.text + "\x1b]8;;\x1b\\"
			if got != want {
				t.Errorf("Hyperlink(%q, %q) = %q, want %q", test.text, test.url, got, want)
			}
			if !strings.HasPrefix(got, "\x1b]8;;"+test.url+"\x1b\\") {
				t.Errorf("Hyperlink(%q, %q) = %q, want it to open with the OSC 8 open sequence embedding url", test.text, test.url, got)
			}
			if !strings.HasSuffix(got, "\x1b]8;;\x1b\\") {
				t.Errorf("Hyperlink(%q, %q) = %q, want it to end with the OSC 8 close sequence", test.text, test.url, got)
			}
			if !strings.Contains(got, test.text) {
				t.Errorf("Hyperlink(%q, %q) = %q, want it to contain the original text", test.text, test.url, got)
			}
		})
	}
}

func TestSafeLinkTarget(t *testing.T) {
	ok := []string{"http://a.test", "https://a.test/x?y=1", "mailto:a@b.test", "file:///tmp/x", "HTTPS://A.TEST", "MailTo:a@b.test"}
	bad := []string{"", "x", "/rel/path", "//host/x", "./a", "javascript:alert(1)", "data:text/plain,x", "ftp://a.test", "vbscript:x",
		" https://a.test", "https://a.test ", "\thttps://a.test", "https://a.test/\x1b\\", "https://a.test/\x07", "https://a.test/\x1b]8;;x",
		"https://a.test/\x00", "https://a.test/\n", "https://a.test/\x7f", "https://a.test/\u009c", "https://a b.test", "https\u0000://x", "ht\x1btp://x"}
	for _, u := range ok {
		if !SafeLinkTarget(u) {
			t.Errorf("SafeLinkTarget(%q) = false, want true", u)
		}
		if got := Hyperlink("t", u); !strings.Contains(got, "\x1b]8;;"+u+"\x1b\\") {
			t.Errorf("Hyperlink(%q) = %q, want OSC 8", "t", got)
		}
	}
	for _, u := range bad {
		if SafeLinkTarget(u) {
			t.Errorf("SafeLinkTarget(%q) = true, want false", u)
		}
		if got := Hyperlink("t", u); got != "t" {
			t.Errorf("Hyperlink(t, %q) = %q, want plain text", u, got)
		}
	}
}
