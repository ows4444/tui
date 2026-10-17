package highlight

import "strings"

// clikeCfg describes one member of the C family: JavaScript, TypeScript,
// Rust, C, C++, Java, C#, Kotlin, Swift and PHP share one scanner and differ
// in these switches.
type clikeCfg struct {
	keywords, types, literals map[string]bool

	backtick   bool // `template` strings spanning lines (JS, TS)
	regex      bool // /re/flags literals (JS, TS)
	rust       bool // raw strings, lifetimes, nested block comments, `..` after a number
	preproc    bool // #include / #define directives (C, C++)
	cppRaw     bool // R"delim(...)delim" raw strings (C++)
	textBlock  bool // """ text blocks (Java)
	annotation bool // @Name annotations (Java, TS decorators)
	digitSep   bool // 1'000 digit separators (C++)
	capsTypes  bool // Capitalised identifiers are types (Java, TS, Rust)

	nestComments bool // /* nested /* block */ comments */ (Kotlin, Swift)
	csStrings    bool // @"verbatim", $"interpolated" and $@"both" strings (C#)
	php          bool // $variables, # comments, #[attributes], <?php ?> tags
	foldCase     bool // keywords and literals match in any case (PHP)
	capsNotCalls bool // a Capitalised word followed by ( is a call, not a type (C# methods)
}

var (
	jsKeywords = setOf("as", "async", "await", "break", "case", "catch", "class", "const",
		"continue", "debugger", "default", "delete", "do", "else", "export", "extends",
		"finally", "for", "from", "function", "get", "if", "import", "in", "instanceof", "let",
		"new", "of", "return", "set", "static", "super", "switch", "throw", "try",
		"typeof", "var", "void", "while", "with", "yield")
	jsLiterals = setOf("true", "false", "null", "undefined", "NaN", "Infinity", "this")
	jsTypes    = setOf("Array", "BigInt", "Boolean", "Date", "Error", "Map", "Number", "Object",
		"Promise", "RegExp", "Set", "String", "Symbol")

	tsKeywords = merge(jsKeywords, setOf("abstract", "declare", "enum", "implements", "interface",
		"keyof", "namespace", "private", "protected", "public", "readonly", "type", "is",
		"satisfies", "infer", "override", "module"))
	tsTypes = merge(jsTypes, setOf("any", "bigint", "boolean", "never", "number", "object",
		"string", "symbol", "unknown"))

	rustKeywords = setOf("as", "async", "await", "break", "const", "continue", "crate", "dyn",
		"else", "enum", "extern", "fn", "for", "if", "impl", "in", "let", "loop", "match", "mod",
		"move", "mut", "pub", "ref", "return", "static", "struct", "super", "trait", "type",
		"union", "unsafe", "use", "where", "while")
	rustTypes = setOf("bool", "char", "f32", "f64", "i8", "i16", "i32", "i64", "i128", "isize",
		"str", "u8", "u16", "u32", "u64", "u128", "usize", "Self", "String", "Vec", "Option",
		"Result", "Box")
	rustLiterals = setOf("true", "false", "None", "self")

	cKeywords = setOf("auto", "break", "case", "const", "continue", "default", "do", "else",
		"enum", "extern", "for", "goto", "if", "inline", "register", "restrict", "return",
		"sizeof", "static", "struct", "switch", "typedef", "union", "volatile", "while")
	cTypes = setOf("char", "double", "float", "int", "long", "short", "signed", "unsigned",
		"void", "size_t", "bool", "int8_t", "int16_t", "int32_t", "int64_t", "uint8_t",
		"uint16_t", "uint32_t", "uint64_t")
	cLiterals = setOf("true", "false", "NULL")

	cppKeywords = merge(cKeywords, setOf("catch", "class", "constexpr", "delete", "explicit",
		"final", "friend", "namespace", "new", "noexcept", "operator", "override", "private",
		"protected", "public", "template", "this", "throw", "try", "typename", "using",
		"virtual", "mutable", "static_cast", "dynamic_cast", "const_cast", "reinterpret_cast"))
	cppTypes    = merge(cTypes, setOf("string", "wchar_t", "char8_t", "char16_t", "char32_t"))
	cppLiterals = setOf("true", "false", "nullptr", "NULL")

	javaKeywords = setOf("abstract", "assert", "break", "case", "catch", "class", "const",
		"continue", "default", "do", "else", "enum", "extends", "final", "finally", "for",
		"goto", "if", "implements", "import", "instanceof", "interface", "native", "new",
		"package", "private", "protected", "public", "return", "static", "strictfp", "super",
		"switch", "synchronized", "this", "throw", "throws", "transient", "try", "var",
		"void", "volatile", "while", "record", "sealed", "permits")
	javaTypes = setOf("boolean", "byte", "char", "double", "float", "int", "long", "short",
		"String", "Object", "Integer", "Long", "List", "Map")
	javaLiterals = setOf("true", "false", "null")

	csKeywords = setOf("abstract", "as", "async", "await", "base", "break", "case", "catch",
		"checked", "class", "const", "continue", "default", "delegate", "do", "dynamic", "else",
		"enum", "event", "explicit", "extern", "finally", "fixed", "for", "foreach", "get", "goto",
		"if", "implicit", "in", "init", "interface", "internal", "is", "lock", "nameof",
		"namespace", "new", "operator", "out", "override", "params", "private", "protected",
		"public", "readonly", "record", "ref", "required", "return", "sealed", "set", "sizeof",
		"stackalloc", "static", "struct", "switch", "throw", "try", "typeof", "unchecked",
		"unsafe", "using", "var", "virtual", "volatile", "when", "where", "while", "with", "yield")
	csTypes = setOf("bool", "byte", "char", "decimal", "double", "float", "int", "long", "nint",
		"nuint", "object", "sbyte", "short", "string", "uint", "ulong", "ushort", "void")
	csLiterals = setOf("true", "false", "null", "this")

	kotlinKeywords = setOf("abstract", "annotation", "as", "break", "by", "catch", "class",
		"companion", "const", "constructor", "continue", "crossinline", "data", "do", "else",
		"enum", "external", "final", "finally", "for", "fun", "get", "if", "import", "in", "infix",
		"init", "inline", "inner", "interface", "internal", "is", "lateinit", "noinline", "object",
		"open", "operator", "out", "override", "package", "private", "protected", "public",
		"reified", "return", "sealed", "set", "super", "suspend", "tailrec", "throw", "try",
		"typealias", "val", "var", "vararg", "when", "where", "while")
	kotlinLiterals = setOf("true", "false", "null", "this")

	swiftKeywords = setOf("actor", "any", "as", "associatedtype", "async", "await", "break",
		"case", "catch", "class", "continue", "convenience", "default", "defer", "deinit", "do",
		"else", "enum", "extension", "fallthrough", "fileprivate", "final", "for", "func", "guard",
		"if", "import", "in", "init", "inout", "internal", "is", "lazy", "let", "mutating",
		"nonisolated", "open", "operator", "override", "private", "protocol", "public", "repeat",
		"required", "rethrows", "return", "some", "static", "struct", "subscript", "switch",
		"throw", "throws", "try", "typealias", "var", "weak", "where", "while")
	swiftLiterals = setOf("true", "false", "nil", "self", "super")

	phpKeywords = setOf("abstract", "and", "array", "as", "break", "callable", "case", "catch",
		"class", "clone", "const", "continue", "declare", "default", "do", "echo", "else",
		"elseif", "empty", "enddeclare", "endfor", "endforeach", "endif", "endswitch",
		"endwhile", "enum", "extends", "final", "finally", "fn", "for", "foreach", "function",
		"global", "goto", "if", "implements", "include", "include_once", "instanceof",
		"insteadof", "interface", "isset", "list", "match", "namespace", "new", "or", "print",
		"private", "protected", "public", "readonly", "require", "require_once", "return",
		"static", "switch", "throw", "trait", "try", "unset", "use", "var", "while", "xor",
		"yield")
	phpTypes    = setOf("bool", "float", "int", "iterable", "mixed", "never", "object", "string", "void")
	phpLiterals = setOf("true", "false", "null")
)

var clikeLangs = map[string]*clikeCfg{
	"js": {keywords: jsKeywords, types: jsTypes, literals: jsLiterals, backtick: true, regex: true},
	"ts": {keywords: tsKeywords, types: tsTypes, literals: jsLiterals, backtick: true, regex: true,
		annotation: true, capsTypes: true},
	"rust": {keywords: rustKeywords, types: rustTypes, literals: rustLiterals, rust: true, capsTypes: true},
	"c":    {keywords: cKeywords, types: cTypes, literals: cLiterals, preproc: true},
	"cpp": {keywords: cppKeywords, types: cppTypes, literals: cppLiterals, preproc: true,
		cppRaw: true, digitSep: true},
	"java": {keywords: javaKeywords, types: javaTypes, literals: javaLiterals, textBlock: true,
		annotation: true, capsTypes: true},
	"csharp": {keywords: csKeywords, types: csTypes, literals: csLiterals, preproc: true,
		textBlock: true, csStrings: true, capsTypes: true, capsNotCalls: true},
	"kotlin": {keywords: kotlinKeywords, literals: kotlinLiterals, textBlock: true,
		annotation: true, capsTypes: true, nestComments: true},
	"swift": {keywords: swiftKeywords, literals: swiftLiterals, textBlock: true, annotation: true,
		capsTypes: true, nestComments: true, preproc: true},
	"php": {keywords: phpKeywords, types: phpTypes, literals: phpLiterals, php: true,
		foldCase: true, capsTypes: true},
}

func merge(a, b map[string]bool) map[string]bool {
	m := make(map[string]bool, len(a)+len(b))
	for k := range a {
		m[k] = true
	}
	for k := range b {
		m[k] = true
	}
	return m
}

func clikeLexer(name string) lexer {
	cfg := clikeLangs[name]
	return func(s string) []Span { return lexClike(s, cfg) }
}

// isCapsType reports whether word starts upper case and has a lower-case
// letter (so CONSTANTS and single letters are not types).
func isCapsType(word string) bool {
	if word[0] < 'A' || word[0] > 'Z' {
		return false
	}
	for i := 1; i < len(word); i++ {
		if word[i] >= 'a' && word[i] <= 'z' {
			return true
		}
	}
	return false
}

// regexAllowed reports whether a '/' after the token ending in lastSig (and,
// for a word, lastWord) starts a regular expression rather than a division.
func regexAllowed(lastSig byte, lastWord string) bool {
	if lastWord != "" {
		switch lastWord {
		case "return", "typeof", "case", "in", "of", "delete", "void", "throw", "new", "else", "do", "yield", "await":
			return true
		}
		return false
	}
	return lastSig == 0 || strings.IndexByte("(,=:[!&|?{};+-*%<>~^", lastSig) >= 0
}

// scanRegex returns the end of a regex literal starting at s[i] ('/'), or -1
// when the line holds no closing slash.
func scanRegex(s string, i int) int {
	inClass := false
	for j := i + 1; j < len(s); j++ {
		switch c := s[j]; {
		case c == '\n':
			return -1
		case c == '\\':
			j++
		case c == '[':
			inClass = true
		case c == ']':
			inClass = false
		case c == '/' && !inClass:
			j++
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			return j
		}
	}
	return -1
}

// scanRustRaw returns the end of a raw string whose 'r' is at s[i], or -1 if
// s[i:] is not one. hashes counts the '#' marks.
func scanRustRaw(s string, i int) int {
	j := i + 1
	for j < len(s) && s[j] == '#' {
		j++
	}
	if j >= len(s) || s[j] != '"' {
		return -1
	}
	closer := "\"" + s[i+1:j]
	k := strings.Index(s[j+1:], closer)
	if k < 0 {
		return len(s)
	}
	return j + 1 + k + len(closer)
}

// scanCppRaw returns the end of R"delim( ... )delim" with R at s[i], or -1.
func scanCppRaw(s string, i int) int {
	if i+1 >= len(s) || s[i+1] != '"' {
		return -1
	}
	open := strings.IndexByte(s[i+2:], '(')
	if open < 0 || open > 16 || strings.ContainsAny(s[i+2:i+2+open], " \n\\\")") {
		return -1
	}
	closer := ")" + s[i+2:i+2+open] + "\""
	k := strings.Index(s[i+3+open:], closer)
	if k < 0 {
		return len(s)
	}
	return i + 3 + open + k + len(closer)
}

// scanRustQuote classifies a Rust single quote at s[i]: a char literal (end
// index, true) or a lifetime (end index, false).
func scanRustQuote(s string, i int) (int, bool) {
	j := i + 1
	if j < len(s) && s[j] == '\\' {
		return scanQuoted(s, i, '\'', false), true
	}
	if j < len(s) {
		k := j + 1
		for k < len(s) && s[k]&0xC0 == 0x80 {
			k++
		}
		if k < len(s) && s[k] == '\'' {
			return k + 1, true
		}
	}
	for j < len(s) && isIdentPart(s[j]) {
		j++
	}
	return j, false
}

func lineEnd(s string, i int) int {
	if j := strings.IndexByte(s[i:], '\n'); j >= 0 {
		return i + j
	}
	return len(s)
}

// clikeScan is the state of lexClike: where it is, and what the last
// significant token was (which decides whether a "/" starts a regex).
type clikeScan struct {
	s         string
	cfg       *clikeCfg
	e         emitter
	i         int
	lastSig   byte
	lastWord  string
	lineStart bool // only blanks seen on this line so far
}

func (l *clikeScan) sig(b byte, word string) { l.lastSig, l.lastWord, l.lineStart = b, word, false }

// take emits s[i:j] as c and moves to j, without counting it as a token.
func (l *clikeScan) take(j int, c Class) {
	l.e.add(l.s[l.i:j], c)
	l.i = j
}

func lexClike(s string, cfg *clikeCfg) []Span {
	l := &clikeScan{s: s, cfg: cfg, lineStart: true}
	for l.i < len(s) {
		l.next()
	}
	return l.e.result()
}

// next lexes one token.
func (l *clikeScan) next() {
	s, i, cfg := l.s, l.i, l.cfg
	c := s[i]
	switch {
	case c == '\n':
		l.e.add("\n", Plain)
		l.i++
		l.lineStart = true
	case c == ' ' || c == '\t' || c == '\r':
		l.e.add(s[i:i+1], Plain)
		l.i++
	case c == '/' && i+1 < len(s) && s[i+1] == '/':
		l.take(lineEnd(s, i), Comment)
		l.lineStart = false
	case c == '/' && i+1 < len(s) && s[i+1] == '*':
		l.take(scanBlockComment(s, i, cfg.rust || cfg.nestComments), Comment)
		l.lineStart = false
	case cfg.php && l.phpToken(c):
	case cfg.csStrings && (c == '@' || c == '$') && csStringStart(s, i) > i:
		k := csStringStart(s, i)
		j := scanQuoted(s, k, '"', false)
		if strings.Contains(s[i:k], "@") {
			j = scanVerbatim(s, k)
		}
		l.take(j, String)
		l.sig('"', "")
	case c == '#' && cfg.preproc && l.lineStart:
		l.preprocessor()
	case c == '"' && cfg.textBlock && strings.HasPrefix(s[i:], `"""`):
		l.take(scanPyString(s, i), String)
		l.sig(c, "")
	case c == '"' || c == '\'' && !cfg.rust:
		l.take(scanQuoted(s, i, c, false), String)
		l.sig(c, "")
	case c == '\'' && cfg.rust:
		j, isChar := scanRustQuote(s, i)
		if isChar {
			l.take(j, String)
		} else {
			l.take(j, Type)
		}
		l.sig(c, "")
	case c == '`' && cfg.backtick:
		l.take(scanQuoted(s, i, c, true), String)
		l.sig(c, "")
	case c == '/' && cfg.regex && regexAllowed(l.lastSig, l.lastWord):
		if j := scanRegex(s, i); j >= 0 {
			l.take(j, String)
			l.sig('/', "")
			return
		}
		l.e.add("/", Plain)
		l.i++
		l.sig(c, "")
	case c == '@' && cfg.annotation && i+1 < len(s) && isIdentStart(s[i+1]):
		j := i + 1
		for j < len(s) && (isIdentPart(s[j]) || s[j] == '.') {
			j++
		}
		l.take(j, Keyword)
		l.sig('a', "")
	case isDigit(c) || c == '.' && i+1 < len(s) && isDigit(s[i+1]):
		l.number()
	case isIdentStart(c):
		l.word()
	default:
		l.e.add(s[i:i+1], Plain)
		l.i++
		l.sig(c, "")
	}
}

// phpToken lexes what only PHP has: "#" comments and attributes, the open and
// close tags, and variables. It reports whether c began one.
func (l *clikeScan) phpToken(c byte) bool {
	s, i := l.s, l.i
	switch {
	case c == '#':
		if i+1 < len(s) && s[i+1] == '[' {
			l.e.add("#[", Keyword) // a PHP 8 attribute
			l.i += 2
			l.sig('[', "")
			return true
		}
		l.take(lineEnd(s, i), Comment)
		l.lineStart = false
		return true
	case strings.HasPrefix(s[i:], "<?php") || strings.HasPrefix(s[i:], "<?=") || strings.HasPrefix(s[i:], "?>"):
		n := 2
		switch {
		case strings.HasPrefix(s[i:], "<?php"):
			n = 5
		case strings.HasPrefix(s[i:], "<?="):
			n = 3
		}
		l.take(i+n, Keyword)
		l.sig(';', "")
		return true
	case c == '$' && i+1 < len(s) && isIdentStart(s[i+1]):
		j := i + 1
		for j < len(s) && isIdentPart(s[j]) {
			j++
		}
		l.take(j, Variable)
		l.sig('a', "")
		return true
	}
	return false
}

// preprocessor lexes a "#" directive at the start of a line, and the <header>
// of an include or import.
func (l *clikeScan) preprocessor() {
	s, i := l.s, l.i
	j := i + 1
	for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
		j++
	}
	for j < len(s) && isIdentPart(s[j]) {
		j++
	}
	word := strings.TrimLeft(s[i+1:j], " \t")
	l.take(j, Keyword)
	if word == "include" || word == "import" {
		k := l.i
		for k < len(s) && (s[k] == ' ' || s[k] == '\t') {
			k++
		}
		if k < len(s) && s[k] == '<' {
			if end := strings.IndexAny(s[k:], ">\n"); end >= 0 && s[k+end] == '>' {
				l.e.add(s[l.i:k], Plain)
				l.e.add(s[k:k+end+1], String)
				l.i = k + end + 1
			}
		}
	}
	l.lineStart = false
}

// number lexes a numeric literal.
func (l *clikeScan) number() {
	s, i, cfg := l.s, l.i, l.cfg
	j := scanNumber(s, i)
	if cfg.rust {
		if k := strings.Index(s[i:j], ".."); k >= 0 {
			j = i + k
		}
	}
	if cfg.digitSep {
		for j+1 < len(s) && s[j] == '\'' && isIdentPart(s[j+1]) {
			j = scanNumber(s, j+1)
		}
	}
	l.take(j, Number)
	l.sig('0', "")
}

// word lexes an identifier, keyword or prefixed string.
func (l *clikeScan) word() {
	s, i, cfg := l.s, l.i, l.cfg
	j := i
	for j < len(s) && isIdentPart(s[j]) {
		j++
	}
	word := s[i:j]
	if end := scanPrefixedString(s, i, j, cfg); end > 0 {
		l.take(end, String)
		l.sig('"', "")
		return
	}
	key := word
	if cfg.foldCase {
		key = strings.ToLower(word)
	}
	switch {
	case cfg.keywords[key]:
		l.e.add(word, Keyword)
	case cfg.types[key] || cfg.capsTypes && isCapsType(word) && !(cfg.capsNotCalls && j < len(s) && s[j] == '('):
		l.e.add(word, Type)
	case cfg.literals[key]:
		l.e.add(word, Literal)
	default:
		l.e.add(word, Plain)
	}
	l.i = j
	l.sig('a', word)
}

// scanPrefixedString handles string forms that begin with a word: Rust raw
// and byte strings and C++ raw strings. word is s[i:j]; it returns the end of
// the literal or 0.
func scanPrefixedString(s string, i, j int, cfg *clikeCfg) int {
	word := s[i:j]
	switch {
	case cfg.rust && (word == "r" || word == "br" || word == "cr") && j < len(s) && (s[j] == '"' || s[j] == '#'):
		if end := scanRustRaw(s, j-1); end >= 0 {
			return end
		}
	case cfg.rust && (word == "b" || word == "c") && j < len(s) && s[j] == '"':
		return scanQuoted(s, j, '"', false)
	case cfg.rust && word == "b" && j < len(s) && s[j] == '\'':
		return scanQuoted(s, j, '\'', false)
	case cfg.cppRaw && (word == "R" || word == "u8R" || word == "uR" || word == "UR" || word == "LR") && j < len(s) && s[j] == '"':
		if end := scanCppRaw(s, j-1); end >= 0 {
			return end
		}
	}
	return 0
}

// scanBlockComment returns the end of the block comment at s[i]; Rust nests.
func scanBlockComment(s string, i int, nested bool) int {
	depth := 1
	for j := i + 2; j < len(s); {
		switch {
		case strings.HasPrefix(s[j:], "*/"):
			depth--
			j += 2
			if depth == 0 {
				return j
			}
		case nested && strings.HasPrefix(s[j:], "/*"):
			depth++
			j += 2
		default:
			j++
		}
	}
	return len(s)
}

// csStringStart returns the index of the opening quote of a C# string whose
// prefix ($, @, $@, @$ or $$ for raw interpolation) starts at s[i], or i when
// s[i:] is not one.
func csStringStart(s string, i int) int {
	k := i
	for k < len(s) && k-i < 3 && (s[k] == '@' || s[k] == '$') {
		k++
	}
	if k > i && k < len(s) && s[k] == '"' {
		return k
	}
	return i
}

// scanVerbatim returns the end of a C# verbatim string whose opening quote is
// s[k]: it may span lines, has no backslash escapes, and "" is a quote.
func scanVerbatim(s string, k int) int {
	for j := k + 1; j < len(s); j++ {
		if s[j] == '"' {
			if j+1 < len(s) && s[j+1] == '"' {
				j++
				continue
			}
			return j + 1
		}
	}
	return len(s)
}
