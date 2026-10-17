package tui

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// EditedMsg is delivered to Update once the editor started by EditInEditor has
// closed and the terminal is back.
type EditedMsg struct {
	// ID is the id EditInEditor was given, to tell one field's edit from
	// another's.
	ID string
	// Text is the text in the file when the editor closed. On an error it is the
	// text that was sent.
	Text string
	// Changed reports whether Text differs from the text that was sent.
	Changed bool
	// Err is why the edit failed: no temp file, the editor would not start or
	// exited with an error, or the file could not be read back. Text is then the
	// original and Changed false, so the caller can carry on with what it had.
	Err error
}

// editorExt is what a file extension passed to EditInEditor must look like: a
// dot and a few letters or digits. Anything else is dropped, so the name given
// to the editor can never hold a path separator or a space.
var editorExt = regexp.MustCompile(`^\.[A-Za-z0-9]{1,16}$`)

// lookupEnv reads the environment for EditInEditor; a test replaces it.
var lookupEnv = os.Getenv

// editorCommand returns the program and arguments to run: $VISUAL, then
// $EDITOR (either may carry arguments, such as "code -w"; they are split on
// spaces and not passed to a shell), then notepad on Windows and vi elsewhere.
func editorCommand() (string, []string) {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if f := strings.Fields(lookupEnv(name)); len(f) > 0 {
			return f[0], f[1:]
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad", nil
	}
	return "vi", nil
}

// EditInEditor returns a Cmd that hands text to the user's editor and delivers
// an EditedMsg with what comes back. It writes text to a private temporary
// file (named with ext, for example ".md" or ".json", so the editor can pick a
// syntax; "" for none), releases the terminal as Suspend does, runs $VISUAL or
// $EDITOR (vi, or notepad on Windows, if neither is set) on the file with the
// real standard input, output and error, restores the terminal, reads the file
// back and removes it. The editor must wait until the user is done: "code -w",
// not "code".
//
// It needs the process's own terminal, so it does nothing useful under
// WithInput or WithOutput pointing elsewhere.
func EditInEditor(id, text, ext string) Cmd {
	return func() Msg {
		res := EditedMsg{ID: id, Text: text}
		return suspendMsg{
			fn: func() error {
				out, err := runEditor(text, ext)
				if err != nil {
					res.Err = err
					return err
				}
				res.Text, res.Changed = out, out != text
				return nil
			},
			then: func(err error) Msg {
				// A failure of the terminal itself (it could not go back to raw
				// mode) is the caller's to hear about too.
				if res.Err == nil && err != nil {
					res.Err = err
				}
				return res
			},
		}
	}
}

// runEditor does EditInEditor's work on the temporary file.
func runEditor(text, ext string) (string, error) {
	if !editorExt.MatchString(ext) {
		ext = ""
	}
	f, err := os.CreateTemp("", "tui-edit-*"+ext)
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()
	if _, err := f.WriteString(text); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	name, args := editorCommand()
	cmd := exec.Command(name, append(args, path)...) // #nosec G204 -- running the editor the user chose in $VISUAL or $EDITOR is the point
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", errors.Join(errors.New("tui: the editor "+name+" failed"), err)
	}
	b, err := os.ReadFile(path) // #nosec G304 -- the file this function just created
	if err != nil {
		return "", err
	}
	return string(b), nil
}
