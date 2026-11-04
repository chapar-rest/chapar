//go:build !js

package highlight

import (
	"bytes"
	"strings"
)

// The scanners below color small line-based formats: ignore files, dotenv,
// INI, Dockerfiles, go.mod and diffs. Their grammars either have no Go
// bindings or are not worth a parser for formats this simple.

// NewGitignore returns a highlighter for .gitignore and similar ignore files:
// comments, ! negation, wildcards and escapes.
func NewGitignore() Highlighter { return newLex(scanGitignore) }

func scanGitignore(src []byte) []Token {
	var toks []Token
	add := func(s, e int, c ColorClass) { toks = append(toks, Token{Start: s, End: e, Class: c}) }
	for i := 0; i < len(src); {
		end := lineEnd(src, i)
		switch {
		case i < end && src[i] == '#':
			add(i, end, ClassComment)
		default:
			j := i
			if j < end && src[j] == '!' {
				add(j, j+1, ClassKeyword)
				j++
			}
			for ; j < end; j++ {
				switch src[j] {
				case '\\':
					e := min(j+2, end)
					add(j, e, ClassConstant)
					j = e - 1
				case '*', '?':
					add(j, j+1, ClassOperator)
				case '[':
					if k := bytes.IndexByte(src[j:end], ']'); k > 0 {
						add(j, j+k+1, ClassOperator)
						j += k
					}
				case '/':
					add(j, j+1, ClassPunctuation)
				}
			}
		}
		i = min(end+1, len(src))
	}
	return toks
}

// NewDotenv returns a highlighter for .env files: comments, export, names,
// values and ${VAR} references.
func NewDotenv() Highlighter { return newLex(layered(scanDotenv)) }

func scanDotenv(src []byte) []Token {
	var toks []Token
	add := func(s, e int, c ColorClass) {
		if e > s {
			toks = append(toks, Token{Start: s, End: e, Class: c})
		}
	}
	for i := 0; i < len(src); {
		end := lineEnd(src, i)
		j := skipBlank(src, i, end)
		switch {
		case j < end && src[j] == '#':
			add(j, end, ClassComment)
		case j < end:
			if bytes.HasPrefix(src[j:end], []byte("export ")) {
				add(j, j+6, ClassKeyword)
				j = skipBlank(src, j+6, end)
			}
			k := j
			for k < end && src[k] != '=' && src[k] != ' ' && src[k] != '\t' {
				k++
			}
			add(j, k, ClassProperty)
			eq := skipBlank(src, k, end)
			if eq < end && src[eq] == '=' {
				add(eq, eq+1, ClassOperator)
				end = scanEnvValue(src, skipBlank(src, eq+1, end), end, add)
			}
		}
		i = min(end+1, len(src))
	}
	return toks
}

// scanEnvValue colors a value from v and returns where its line ends: a
// quoted value may run over several lines.
func scanEnvValue(src []byte, v, end int, add func(s, e int, c ColorClass)) int {
	if v >= end {
		return end
	}
	if q := src[v]; q == '"' || q == '\'' || q == '`' {
		k := v + 1
		for k < len(src) && src[k] != q {
			if src[k] == '\\' && q != '\'' {
				k++
			}
			k++
		}
		k = min(k+1, len(src))
		add(v, k, ClassString)
		if q != '\'' {
			scanEnvRefs(src, v, k, add)
		}
		rest := lineEnd(src, k)
		if c := bytes.IndexByte(src[k:rest], '#'); c >= 0 {
			add(k+c, rest, ClassComment)
		}
		return rest
	}
	stop := end
	for k := v; k < end; k++ {
		if src[k] == '#' && (src[k-1] == ' ' || src[k-1] == '\t') {
			stop = k
			add(k, end, ClassComment)
			break
		}
	}
	add(v, stop, ClassString)
	scanEnvRefs(src, v, stop, add)
	return end
}

// scanEnvRefs colors $NAME and ${NAME} references in src[lo:hi]. They are
// added after the string around them, so they are drawn on top of it.
func scanEnvRefs(src []byte, lo, hi int, add func(s, e int, c ColorClass)) {
	for k := lo; k < hi; k++ {
		if src[k] != '$' || k+1 >= hi {
			continue
		}
		if src[k+1] == '{' {
			if c := bytes.IndexByte(src[k:hi], '}'); c > 0 {
				add(k, k+c+1, ClassVariable)
				k += c
			}
			continue
		}
		e := k + 1
		for e < hi && isWord(src[e]) {
			e++
		}
		if e > k+1 {
			add(k, e, ClassVariable)
			k = e - 1
		}
	}
}

// NewINI returns a highlighter for INI-style files such as .editorconfig and
// .gitconfig: comments, [sections], keys and values.
func NewINI() Highlighter { return newLex(scanINI) }

func scanINI(src []byte) []Token {
	var toks []Token
	add := func(s, e int, c ColorClass) {
		if e > s {
			toks = append(toks, Token{Start: s, End: e, Class: c})
		}
	}
	for i := 0; i < len(src); {
		end := lineEnd(src, i)
		j := skipBlank(src, i, end)
		switch {
		case j >= end:
		case src[j] == '#' || src[j] == ';':
			add(j, end, ClassComment)
		case src[j] == '[':
			e := end
			if k := bytes.IndexByte(src[j:end], ']'); k > 0 {
				e = j + k + 1
			}
			add(j, e, ClassType)
		default:
			k := j
			for k < end && src[k] != '=' && src[k] != ':' {
				k++
			}
			if k == end {
				add(j, trimRight(src, j, end), ClassProperty) // a bare key, as in git's boolean options
				break
			}
			add(j, trimRight(src, j, k), ClassProperty)
			add(k, k+1, ClassOperator)
			scanINIValue(src, skipBlank(src, k+1, end), end, add)
		}
		i = min(end+1, len(src))
	}
	return toks
}

var iniConstants = map[string]bool{
	"true": true, "false": true, "yes": true, "no": true, "on": true, "off": true, "none": true,
}

func scanINIValue(src []byte, v, end int, add func(s, e int, c ColorClass)) {
	comment := end
	for k := v; k < end; k++ {
		if (src[k] == '#' || src[k] == ';') && k > v && (src[k-1] == ' ' || src[k-1] == '\t') {
			comment = k
			break
		}
	}
	stop := trimRight(src, v, comment)
	val := src[v:stop]
	switch {
	case len(val) == 0:
	case iniConstants[strings.ToLower(string(val))]:
		add(v, stop, ClassConstant)
	case isNumber(val):
		add(v, stop, ClassNumber)
	default:
		add(v, stop, ClassString)
	}
	add(comment, end, ClassComment)
}

// NewDockerfile returns a highlighter for Dockerfiles: instructions,
// --flags, strings, $variables and comments.
func NewDockerfile() Highlighter { return newLex(layered(scanDockerfile)) }

var dockerInstructions = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`from run cmd label maintainer expose env add copy entrypoint volume
		user workdir arg onbuild stopsignal healthcheck shell`) {
		m[w] = true
	}
	return m
}()

func scanDockerfile(src []byte) []Token {
	var toks []Token
	add := func(s, e int, c ColorClass) {
		if e > s {
			toks = append(toks, Token{Start: s, End: e, Class: c})
		}
	}
	cont := false // the previous line ended with a \ continuation
	for i := 0; i < len(src); {
		end := lineEnd(src, i)
		j := skipBlank(src, i, end)
		if j < end && src[j] == '#' {
			add(j, end, ClassComment) // comments may sit between continued lines too
			i = min(end+1, len(src))
			continue
		}
		if !cont && j < end {
			k := j
			for k < end && isWord(src[k]) {
				k++
			}
			word := strings.ToLower(string(src[j:k]))
			if dockerInstructions[word] {
				add(j, k, ClassKeyword)
				j = k
				if word == "from" {
					if a := asKeyword(src, j, end); a >= 0 {
						add(a, a+2, ClassKeyword)
					}
				}
			}
		}
		scanDockerArgs(src, j, end, add)
		t := trimRight(src, i, end)
		cont = t > i && src[t-1] == '\\'
		i = min(end+1, len(src))
	}
	return toks
}

// asKeyword finds the " AS " of a FROM line in src[lo:hi], or returns -1.
func asKeyword(src []byte, lo, hi int) int {
	for k := lo; k+3 < hi; k++ {
		if (src[k] == ' ' || src[k] == '\t') && strings.EqualFold(string(src[k+1:k+3]), "as") &&
			(src[k+3] == ' ' || src[k+3] == '\t') {
			return k + 1
		}
	}
	return -1
}

func scanDockerArgs(src []byte, lo, hi int, add func(s, e int, c ColorClass)) {
	for k := lo; k < hi; k++ {
		switch c := src[k]; {
		case c == '"' || c == '\'':
			e := k + 1
			for e < hi && src[e] != c {
				if src[e] == '\\' {
					e++
				}
				e++
			}
			e = min(e+1, hi)
			add(k, e, ClassString)
			if c == '"' {
				scanEnvRefs(src, k, e, add)
			}
			k = e - 1
		case c == '$':
			e := k + 1
			if e < hi && src[e] == '{' {
				if b := bytes.IndexByte(src[e:hi], '}'); b > 0 {
					add(k, e+b+1, ClassVariable)
					k = e + b
				}
				continue
			}
			for e < hi && isWord(src[e]) {
				e++
			}
			add(k, e, ClassVariable)
			k = max(k, e-1)
		case c == '-' && k+1 < hi && src[k+1] == '-' && (k == lo || src[k-1] == ' ' || src[k-1] == '\t'):
			e := k + 2
			for e < hi && (isWord(src[e]) || src[e] == '-') {
				e++
			}
			add(k, e, ClassAttribute)
			k = e - 1
		}
	}
}

// NewGoMod returns a highlighter for go.mod and go.work files: directives,
// module paths, versions and comments.
func NewGoMod() Highlighter { return newLex(scanGoMod) }

var goModDirectives = map[string]bool{
	"module": true, "go": true, "toolchain": true, "godebug": true, "require": true,
	"replace": true, "exclude": true, "retract": true, "tool": true, "ignore": true, "use": true,
}

func scanGoMod(src []byte) []Token {
	var toks []Token
	add := func(s, e int, c ColorClass) {
		if e > s {
			toks = append(toks, Token{Start: s, End: e, Class: c})
		}
	}
	for i := 0; i < len(src); {
		lineStop := lineEnd(src, i)
		end := lineStop
		if c := bytes.Index(src[i:end], []byte("//")); c >= 0 {
			end = i + c
		}
		first := true
		for k := skipBlank(src, i, end); k < end; k = skipBlank(src, k, end) {
			e := k
			if src[k] == '"' || src[k] == '`' {
				e++
				for e < end && src[e] != src[k] {
					e++
				}
				e = min(e+1, end)
			} else {
				for e < end && src[e] != ' ' && src[e] != '\t' {
					e++
				}
			}
			word := string(src[k:e])
			switch {
			case first && goModDirectives[word]:
				add(k, e, ClassKeyword)
			case word == "=>":
				add(k, e, ClassOperator)
			case word == "(" || word == ")" || word == "[" || word == "]" || word == ",":
				add(k, e, ClassPunctuation)
			case word[0] == 'v' && len(word) > 1 && isDigit(word[1]) || isDigit(word[0]):
				add(k, e, ClassNumber)
			case word[0] == '"' || word[0] == '`':
				add(k, e, ClassString)
			default:
				add(k, e, ClassNamespace)
			}
			first = false
			k = e
		}
		add(end, lineStop, ClassComment)
		i = min(lineStop+1, len(src))
	}
	return toks
}

// NewDiff returns a highlighter for unified diffs and patches: added and
// removed lines, hunk headers and file headers.
func NewDiff() Highlighter { return newLex(scanDiff) }

func scanDiff(src []byte) []Token {
	var toks []Token
	add := func(s, e int, c ColorClass) {
		if e > s {
			toks = append(toks, Token{Start: s, End: e, Class: c})
		}
	}
	for i := 0; i < len(src); {
		end := lineEnd(src, i)
		line := src[i:end]
		switch {
		case bytes.HasPrefix(line, []byte("+++ ")) || bytes.HasPrefix(line, []byte("--- ")):
			add(i, end, ClassKeyword)
		case bytes.HasPrefix(line, []byte("@@")):
			e := end
			if k := bytes.Index(line[2:], []byte("@@")); k >= 0 {
				e = i + 2 + k + 2
			}
			add(i, e, ClassType)
		case len(line) > 0 && line[0] == '+':
			add(i, end, ClassSuccess)
		case len(line) > 0 && line[0] == '-':
			add(i, end, ClassError)
		case len(line) > 0 && line[0] == '\\':
			add(i, end, ClassMuted) // "\ No newline at end of file"
		case len(line) > 0 && line[0] != ' ':
			add(i, end, ClassMuted) // diff --git, index, mode and rename lines
		}
		i = min(end+1, len(src))
	}
	return toks
}

// layered wraps a scanner whose tokens may nest, such as a $VAR inside a
// string, into one whose tokens do not overlap: the inner token wins.
func layered(scan func([]byte) []Token) func([]byte) []Token {
	return func(src []byte) []Token {
		toks := scan(src)
		spans := make([]span, len(toks))
		for i, t := range toks {
			spans[i] = span{start: t.Start, end: t.End, pattern: uint(i), class: t.Class}
		}
		return flatten(spans, 0, len(src))
	}
}

func skipBlank(src []byte, i, end int) int {
	for i < end && (src[i] == ' ' || src[i] == '\t' || src[i] == '\r') {
		i++
	}
	return i
}

func trimRight(src []byte, lo, hi int) int {
	for hi > lo && (src[hi-1] == ' ' || src[hi-1] == '\t' || src[hi-1] == '\r') {
		hi--
	}
	return hi
}

func isNumber(b []byte) bool {
	if len(b) > 0 && (b[0] == '-' || b[0] == '+') {
		b = b[1:]
	}
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if !isDigit(c) && c != '.' {
			return false
		}
	}
	return true
}
