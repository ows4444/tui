package highlight

import "strings"

// yamlDelims ends a plain scalar word.
const yamlDelims = " \t\n,:[]{}"

// yamlKey returns the end of the mapping key that starts at s[i] on its line,
// or false if the text there is not a key: a key is a quoted or plain scalar
// followed by a colon and then a space or the end of the line.
func yamlKey(s string, i int) (int, bool) {
	if strings.IndexByte("[{&*!|>%@#", s[i]) >= 0 || s[i] == '-' && (i+1 == len(s) || s[i+1] == ' ' || s[i+1] == '\n') {
		return 0, false
	}
	if s[i] == '"' || s[i] == '\'' {
		j := scanQuoted(s, i, s[i], false)
		k := j
		for k < len(s) && (s[k] == ' ' || s[k] == '\t') {
			k++
		}
		if k < len(s) && s[k] == ':' && (k+1 == len(s) || s[k+1] == ' ' || s[k+1] == '\n') {
			return j, true
		}
		return 0, false
	}
	for j := i; j < len(s) && s[j] != '\n'; j++ {
		if s[j] == '#' && j > i && s[j-1] == ' ' {
			return 0, false
		}
		if s[j] == ':' && (j+1 == len(s) || s[j+1] == ' ' || s[j+1] == '\n') {
			end := j
			for end > i && (s[end-1] == ' ' || s[end-1] == '\t') {
				end--
			}
			return end, end > i
		}
	}
	return 0, false
}

// yamlRun returns the end of the run of non-delimiter bytes starting at i.
func yamlRun(s string, i int) int {
	for i < len(s) && strings.IndexByte(yamlDelims, s[i]) < 0 {
		i++
	}
	return i
}

func lexYAML(s string) []Span {
	var e emitter
	lineStart := true // only indentation and "- " seen so far on this line
	wordStart := true
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\n':
			e.add("\n", Plain)
			i, lineStart, wordStart = i+1, true, true
		case c == ' ' || c == '\t':
			e.add(s[i:i+1], Plain)
			i, wordStart = i+1, true
		case c == '#' && wordStart:
			j := strings.IndexByte(s[i:], '\n')
			if j < 0 {
				j = len(s) - i
			}
			e.add(s[i:i+j], Comment)
			i += j
		case (i == 0 || s[i-1] == '\n') && (strings.HasPrefix(s[i:], "---") || strings.HasPrefix(s[i:], "...")) &&
			(i+3 == len(s) || s[i+3] == ' ' || s[i+3] == '\n'):
			e.add(s[i:i+3], Keyword)
			i, lineStart, wordStart = i+3, false, true
		case c == '-' && lineStart && (i+1 == len(s) || s[i+1] == ' ' || s[i+1] == '\n'):
			e.add("-", Plain)
			i, wordStart = i+1, true
		default:
			if lineStart {
				if end, ok := yamlKey(s, i); ok {
					e.add(s[i:end], Key)
					i, lineStart, wordStart = end, false, false
					continue
				}
			}
			lineStart = false
			switch {
			case c == '"' || c == '\'':
				j := scanQuoted(s, i, c, false)
				e.add(s[i:j], String)
				i, wordStart = j, false
			case (c == '&' || c == '*') && wordStart:
				j := yamlRun(s, i+1)
				e.add(s[i:j], Variable)
				i, wordStart = j, false
			case c == '!' && wordStart:
				j := yamlRun(s, i+1)
				e.add(s[i:j], Type)
				i, wordStart = j, false
			case wordStart && (isDigit(c) || (c == '-' || c == '+') && i+1 < len(s) && isDigit(s[i+1])):
				j := scanNumber(s, i+1)
				if j == len(s) || strings.IndexByte(" \t\n,]}#", s[j]) >= 0 {
					e.add(s[i:j], Number)
				} else {
					e.add(s[i:j], Plain)
				}
				i, wordStart = j, false
			case strings.IndexByte(yamlDelims, c) < 0:
				j := yamlRun(s, i)
				switch word := s[i:j]; strings.ToLower(word) {
				case "true", "false", "null", "~":
					e.add(word, Literal)
				default:
					e.add(word, Plain)
				}
				i, wordStart = j, false
			default:
				e.add(s[i:i+1], Plain)
				i, wordStart = i+1, c == ',' || c == '[' || c == '{'
			}
		}
	}
	return e.result()
}
