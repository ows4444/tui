// Command survey is a short questionnaire at the command line, with no Model
// of its own: every question is one call of package prompt, and nothing takes
// over the screen. It asks for a name and an email (checked as you go), a
// number, one role from a grouped list, several tools, a score and an optional
// comment, then confirms, "sends" the answers with a progress bar and prints
// a summary. Nothing is sent anywhere: the progress is a pause.
//
// Keys: Enter answers, Up and Down move through a list, Space ticks an entry
// of a multiple choice, typing narrows a list, Esc or Ctrl+C cancels (exit 130).
//
//	survey          # asks, then prints a summary
//	survey -json    # the summary as JSON on stdout, the questions on the terminal
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui/prompt"
)

type answers struct {
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Years   int      `json:"years"`
	Role    string   `json:"role"`
	Tools   []string `json:"tools"`
	Score   int      `json:"score"`
	Comment string   `json:"comment,omitempty"`
	Send    bool     `json:"-"`
}

var roles = []prompt.Choice[string]{
	{Label: "Backend engineer", Value: "backend", Group: "Work"},
	{Label: "Frontend engineer", Value: "frontend", Group: "Work"},
	{Label: "Platform or SRE", Value: "platform", Group: "Work"},
	{Label: "Student", Value: "student", Group: "Learning"},
	{Label: "Hobbyist", Value: "hobbyist", Group: "Learning"},
}

var tools = []string{"Editor plugins", "Linters", "Test frameworks", "Profilers", "Debuggers", "Code generators"}

// questions is how many prompts ask(cfg) opens, so a test can give each one
// its own input.
const questions = 8

func validName(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("your name cannot be empty")
	}
	return nil
}

func parseEmail(s string) (string, error) {
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s {
		return "", errors.New("that does not look like an email address")
	}
	return s, nil
}

func parseYears(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 60 {
		return 0, errors.New("a whole number from 0 to 60")
	}
	return n, nil
}

func parseScore(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 10 {
		return 0, errors.New("a whole number from 0 to 10")
	}
	return n, nil
}

// ask runs the questions. cfg returns the prompt.Config of question i, so a
// test can give each its own input and output.
func ask(cfg func(i int) prompt.Config) (answers, error) {
	var a answers
	var err error

	c := cfg(0)
	c.Validate, c.Placeholder = validName, "Ada Lovelace"
	if a.Name, err = prompt.Input("Your name:", c); err != nil {
		return a, err
	}
	c = cfg(1)
	c.Placeholder = "ada@example.com"
	if a.Email, err = prompt.Parse("Email:", parseEmail, c); err != nil {
		return a, err
	}
	if a.Years, err = prompt.Parse("Years writing Go (0 to 60):", parseYears, cfg(2)); err != nil {
		return a, err
	}
	if a.Role, err = prompt.Choose("What describes you best?", roles, cfg(3)); err != nil {
		return a, err
	}
	picked, err := prompt.MultiSelect("Tools you use (space ticks):", tools, cfg(4))
	if err != nil {
		return a, err
	}
	for _, i := range picked {
		a.Tools = append(a.Tools, tools[i])
	}
	if a.Score, err = prompt.Parse("How likely are you to recommend it, 0 to 10?", parseScore, cfg(5)); err != nil {
		return a, err
	}
	c = cfg(6)
	c.Placeholder = "optional"
	if a.Comment, err = prompt.Input("Anything to add?", c); err != nil {
		return a, err
	}
	a.Comment = strings.TrimSpace(a.Comment)
	a.Send, err = prompt.Confirm("Send your answers?", true, cfg(7))
	return a, err
}

// send shows a progress bar for the pretend upload, one unit a step. It stops
// early, with the context's error, when the person cancels.
func send(cfg prompt.Config, step time.Duration) error {
	const units = 20
	return prompt.Progress("Sending", units, func(ctx context.Context, report func(int)) error {
		for i := 1; i <= units; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(step):
			}
			report(i)
		}
		return nil
	}, cfg)
}

// summary writes the answers, as text or as JSON.
func summary(w io.Writer, a answers, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(a)
	}
	row := func(k, v string) { fmt.Fprintf(w, "  %-9s %s\n", k, v) }
	fmt.Fprintf(w, "Thank you, %s. Here is what you sent:\n", a.Name)
	row("Email", a.Email)
	row("Years", strconv.Itoa(a.Years))
	row("Role", a.Role)
	tl := "none"
	if len(a.Tools) > 0 {
		tl = strings.Join(a.Tools, ", ")
	}
	row("Tools", tl)
	row("Score", strconv.Itoa(a.Score)+" / 10")
	if a.Comment != "" {
		row("Comment", a.Comment)
	}
	return nil
}

func main() {
	asJSON := flag.Bool("json", false, "print the summary as JSON on stdout")
	flag.Parse()
	a, err := ask(func(int) prompt.Config { return prompt.Config{} })
	if err == nil && a.Send {
		err = send(prompt.Config{}, 60*time.Millisecond)
	}
	switch {
	case errors.Is(err, prompt.ErrCanceled), errors.Is(err, context.Canceled):
		fmt.Fprintln(os.Stderr, "canceled")
		os.Exit(130)
	case err != nil:
		fmt.Fprintln(os.Stderr, "survey:", err)
		os.Exit(1)
	case !a.Send:
		fmt.Fprintln(os.Stderr, "not sent")
		return
	}
	if err := summary(os.Stdout, a, *asJSON); err != nil {
		fmt.Fprintln(os.Stderr, "survey:", err)
		os.Exit(1)
	}
}
