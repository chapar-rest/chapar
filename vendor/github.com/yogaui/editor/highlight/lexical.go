//go:build !js

package highlight

import (
	"bytes"
	"strings"
	"sync"
)

// lexHighlighter colors text with a hand-written scanner instead of a
// grammar. It is for languages whose Tree-sitter grammar cannot be built from
// Go modules. Scanning runs on a worker like the Tree-sitter highlighters, so
// a large document never stalls typing.
type lexHighlighter struct {
	scan     func(src []byte) []Token
	maxBytes int

	mu       sync.Mutex
	src      []byte
	queued   bool
	oversize bool
	wake     chan struct{}
	results  chan []Token
	done     chan struct{}
}

func newLex(scan func([]byte) []Token) Highlighter {
	h := &lexHighlighter{
		scan:     scan,
		maxBytes: MaxBytes,
		wake:     make(chan struct{}, 1),
		results:  make(chan []Token, 1),
		done:     make(chan struct{}),
	}
	if h.maxBytes <= 0 {
		h.maxBytes = DefaultMaxBytes
	}
	go h.loop()
	return h
}

func (h *lexHighlighter) loop() {
	for {
		select {
		case <-h.done:
			return
		case <-h.wake:
		}
		h.mu.Lock()
		src, ok := h.src, h.queued
		h.queued = false
		h.mu.Unlock()
		if !ok {
			continue
		}
		deliver(h.results, h.scan(src))
	}
}

func (h *lexHighlighter) Update(source []byte) {
	h.mu.Lock()
	over := len(source) > h.maxBytes
	wasOver := h.oversize
	h.oversize = over
	if over {
		h.mu.Unlock()
		if !wasOver {
			deliver(h.results, nil)
		}
		return
	}
	h.src = append(h.src[:0:0], source...)
	h.queued = true
	h.mu.Unlock()
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

func (h *lexHighlighter) UpdateEdit(source []byte, _ Edit) { h.Update(source) }

func (h *lexHighlighter) Poll() ([]Token, bool) {
	select {
	case t := <-h.results:
		return t, true
	default:
		return nil, false
	}
}

func (h *lexHighlighter) Close() { close(h.done) }

func (h *lexHighlighter) Oversize() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.oversize
}

func (h *lexHighlighter) SetMaxBytes(n int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n <= 0 {
		n = DefaultMaxBytes
	}
	h.maxBytes = n
}

var _ SizeLimited = (*lexHighlighter)(nil)

// NewSQL returns a highlighter for SQL: keywords (any case), strings,
// quoted identifiers, numbers and comments.
func NewSQL() Highlighter { return newLex(scanSQL) }

var sqlKeywords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`select from where and or not insert into values update set delete
		create table alter drop index view primary key foreign references unique default null is in
		like ilike between exists join inner left right full outer cross on using group by order asc desc
		having limit offset union all distinct as case when then else end begin commit rollback
		transaction if returning with recursive cascade constraint check grant revoke schema database
		function procedure trigger language returns declare true false int integer bigint smallint
		serial bigserial text varchar char boolean bool date time timestamp timestamptz interval uuid
		jsonb json numeric decimal real float double precision bytea extension sequence column add
		rename to owner type enum coalesce count sum avg min max now replace temporary temp only`) {
		m[w] = true
	}
	return m
}()

func scanSQL(src []byte) []Token {
	var toks []Token
	add := func(s, e int, c ColorClass) { toks = append(toks, Token{Start: s, End: e, Class: c}) }
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			j := lineEnd(src, i)
			add(i, j, ClassComment)
			i = j
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			j := bytes.Index(src[i+2:], []byte("*/"))
			end := len(src)
			if j >= 0 {
				end = i + 2 + j + 2
			}
			add(i, end, ClassComment)
			i = end
		case c == '\'' || c == '"' || c == '`':
			j := i + 1
			for j < len(src) && src[j] != c {
				j++
			}
			j = min(j+1, len(src))
			class := ClassString
			if c != '\'' {
				class = ClassType // quoted identifier
			}
			add(i, j, class)
			i = j
		case isDigit(c):
			j := i
			for j < len(src) && (isDigit(src[j]) || src[j] == '.') {
				j++
			}
			add(i, j, ClassNumber)
			i = j
		case isWordStart(c):
			j := i
			for j < len(src) && isWord(src[j]) {
				j++
			}
			if sqlKeywords[strings.ToLower(string(src[i:j]))] {
				add(i, j, ClassKeyword)
			}
			i = j
		default:
			i++
		}
	}
	return toks
}

// NewMarkdown returns a highlighter for Markdown: headings, fenced and inline
// code, quotes, list markers, emphasis and links.
func NewMarkdown() Highlighter { return newLex(scanMarkdown) }

func scanMarkdown(src []byte) []Token {
	var toks []Token
	add := func(s, e int, c ColorClass) {
		if e > s {
			toks = append(toks, Token{Start: s, End: e, Class: c})
		}
	}
	inFence := false
	fenceStart := 0
	var fence []byte
	for i := 0; i < len(src); {
		end := lineEnd(src, i)
		line := src[i:end]
		trim := bytes.TrimLeft(line, " ")
		indent := len(line) - len(trim)
		next := min(end+1, len(src))
		if f := fenceMarker(trim); f != nil && indent < 4 {
			if !inFence {
				inFence, fence, fenceStart = true, f, i
			} else if bytes.HasPrefix(trim, fence) {
				inFence = false
				add(fenceStart, end, ClassString)
			}
			i = next
			continue
		}
		if inFence {
			i = next
			continue
		}
		switch {
		case len(trim) > 0 && trim[0] == '#':
			add(i, end, ClassKeyword)
		case len(trim) > 0 && trim[0] == '>':
			add(i, end, ClassComment)
		default:
			if m := listMarker(trim); m > 0 {
				add(i+indent, i+indent+m, ClassNumber)
			}
			scanInline(src, i, end, add)
		}
		i = next
	}
	if inFence {
		add(fenceStart, len(src), ClassString)
	}
	return toks
}

func fenceMarker(line []byte) []byte {
	for _, f := range [][]byte{[]byte("```"), []byte("~~~")} {
		if bytes.HasPrefix(line, f) {
			return f
		}
	}
	return nil
}

// listMarker returns the width of a "- ", "* ", "+ " or "12. " marker.
func listMarker(line []byte) int {
	if len(line) >= 2 && (line[0] == '-' || line[0] == '*' || line[0] == '+') && line[1] == ' ' {
		return 1
	}
	j := 0
	for j < len(line) && isDigit(line[j]) {
		j++
	}
	if j > 0 && j+1 < len(line) && (line[j] == '.' || line[j] == ')') && line[j+1] == ' ' {
		return j + 1
	}
	return 0
}

// scanInline colors `code`, **strong**, *emphasis* and [text](url) links in
// src[lo:hi].
func scanInline(src []byte, lo, hi int, add func(s, e int, c ColorClass)) {
	for i := lo; i < hi; i++ {
		switch src[i] {
		case '`':
			if j := bytes.IndexByte(src[i+1:hi], '`'); j >= 0 {
				add(i, i+j+2, ClassString)
				i += j + 1
			}
		case '*', '_':
			mark := src[i : i+1]
			if i+1 < hi && src[i+1] == src[i] {
				mark = src[i : i+2]
			}
			if j := bytes.Index(src[i+len(mark):hi], mark); j > 0 {
				add(i, i+len(mark)+j+len(mark), ClassType)
				i += len(mark) + j + len(mark) - 1
			}
		case '[':
			close := bytes.IndexByte(src[i:hi], ']')
			if close > 0 && i+close+1 < hi && src[i+close+1] == '(' {
				if paren := bytes.IndexByte(src[i+close+1:hi], ')'); paren > 0 {
					urlStart := i + close + 1
					add(urlStart, urlStart+paren+1, ClassType)
					i = urlStart + paren
				}
			}
		}
	}
}

func lineEnd(src []byte, i int) int {
	if j := bytes.IndexByte(src[i:], '\n'); j >= 0 {
		return i + j
	}
	return len(src)
}

func isDigit(c byte) bool     { return c >= '0' && c <= '9' }
func isWordStart(c byte) bool { return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isWord(c byte) bool      { return isWordStart(c) || isDigit(c) }
