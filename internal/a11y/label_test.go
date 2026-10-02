package a11y

import "testing"

func TestPromptLabel(t *testing.T) {
	cases := map[string]string{
		"Name: ":                     "Name",
		"Name:  ":                    "Name",
		"> ":                         "",
		"$ ":                         "",
		"»":                          "",
		"":                           "",
		"   ":                        "",
		"Full name (required): ":     "Full name (required)",
		"(optional) Nickname: ":      "(optional) Nickname",
		"[1] Pick one > ":            "[1] Pick one",
		`"Quoted" label: `:           `"Quoted" label`,
		"E-mail address: ":           "E-mail address",
		"Größe: ":                    "Größe",
		"名前: ":                       "名前",
		"Password":                   "Password",
		">>> Enter your answer >>> ": "Enter your answer",
	}
	for in, want := range cases {
		if got := PromptLabel(in); got != want {
			t.Errorf("PromptLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
