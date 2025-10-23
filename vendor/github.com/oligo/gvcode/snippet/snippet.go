package snippet

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	snippetPattern = regexp.MustCompile(`\$((\d+)|(\w+))|\$\{([^}]+)\}`)
)

type bytesOff struct {
	start int
	end   int
}

type runesOff struct {
	start int
	end   int
}

// TabStop is the tabstop defined in LSP protocol:
type TabStop struct {
	content string
	// location is the bytes offset of the tabstop in the snippet.
	location bytesOff
	// index of the snippet tabstop, e.g.,0, 1, 2, etc.
	idx int
	// placeholder value of the tabstop.
	placeholder string
	choices     []string
	// variable name of the tabstop.
	variable        string
	variableDefault string
}

func (ts TabStop) IsFinal() bool {
	return ts.idx == 0 && ts.variable == ""
}

func (sc TabStop) String() string {
	return fmt.Sprintf("TabStop(%d-%d)[content: %s, idx: %d, placeholder: %s, choices: %v, variable: %s, variableDefault: %s]",
		sc.location.start, sc.location.end, sc.content, sc.idx, sc.placeholder, sc.choices, sc.variable, sc.variableDefault)
}

// Snippet holds the parsed data structure of the snippet format defined in LSP protocol:
//
//	https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#snippet_syntax
//
// Usually this is used along with auto-completion.
type Snippet struct {
	raw string
	// A template is the 'default' content inserted into the editor.
	template  string
	tabStops  []*TabStop
	locations map[*TabStop]runesOff
}

func NewSnippet(content string) *Snippet {
	return &Snippet{raw: content}
}

func (s *Snippet) Parse() error {
	err := s.parseTabstops()
	if err != nil {
		return err
	}

	s.buildTemplate()

	// sort by idx in ascending order and specially:
	// 	1. put variables at the end of the slice.
	//  2. then followed by $0 tabstops.
	slices.SortFunc(s.tabStops, func(a, b *TabStop) int {
		if a.IsFinal() && !b.IsFinal() {
			return 1
		} else if !a.IsFinal() && b.IsFinal() {
			return -1
		} else if a.IsFinal() && b.IsFinal() {
			return 0
		}

		if a.variable != "" && b.variable == "" {
			return 1
		} else if a.variable == "" && b.variable != "" {
			return -1
		} else if a.variable != "" && b.variable != "" {
			return -1
		}

		return cmp.Compare(a.idx, b.idx)
	})

	addFinal := false
	if len(s.tabStops) == 0 {
		addFinal = true
	} else {
		lastTabStop := s.tabStops[len(s.tabStops)-1]
		if !lastTabStop.IsFinal() {
			addFinal = true
		}
	}

	if addFinal {
		snippetLen := len(s.raw)
		final := &TabStop{idx: 0, location: bytesOff{start: snippetLen, end: snippetLen}}
		s.tabStops = append(s.tabStops, final)
		templateRunes := utf8.RuneCountInString(s.template)
		s.locations[final] = runesOff{start: templateRunes, end: templateRunes}
	}

	return nil
}

func (s *Snippet) parseTabstops() error {
	for _, matches := range snippetPattern.FindAllStringSubmatchIndex(s.raw, -1) {
		if len(matches) == 0 {
			continue
		}
		// The tabstop content in the snippet.
		content := s.raw[matches[0]:matches[1]]

		// As the RE pattern uses nested group for the first kind of tabstop,
		// we should skip the parent group, and just check the sub capture groups.
		if matches[4] >= 0 && matches[5] >= 0 {
			tabStopIdx, err := strconv.Atoi(s.raw[matches[4]:matches[5]])
			if err != nil {
				return err
			}

			ts := &TabStop{
				content:  content,
				idx:      tabStopIdx,
				location: bytesOff{start: matches[0], end: matches[1]},
			}
			s.tabStops = append(s.tabStops, ts)
			continue
		}

		// check the second sub capture group.
		if matches[6] >= 0 && matches[7] >= 0 {
			// A variable name is found.
			ts := &TabStop{
				content:  content,
				variable: s.raw[matches[6]:matches[7]],
				location: bytesOff{start: matches[0], end: matches[1]},
			}
			s.tabStops = append(s.tabStops, ts)
			continue
		}

		// check the third capture group. It can be placeholder tabstop, variable
		// with default value and choices.
		if matches[8] >= 0 && matches[9] >= 0 {
			matchedText := s.raw[matches[8]:matches[9]]
			ts := &TabStop{
				content:  content,
				location: bytesOff{start: matches[0], end: matches[1]},
			}
			ts, err := s.parseSubText(ts, matchedText)
			if err != nil {
				return err
			}

			s.tabStops = append(s.tabStops, ts)
			continue
		}

	}

	return nil
}

func (s *Snippet) parseSubText(tabstop *TabStop, subtext string) (*TabStop, error) {
	idx := strings.Index(subtext, ":")
	if idx > 0 && idx <= len(subtext)-1 {
		prefix := subtext[0:idx]
		suffix := subtext[idx+1:]
		tabstopIdx, err := strconv.Atoi(prefix)
		if err != nil {
			// subtext is a variable with default value: varname:defaultValue.
			tabstop.variable = prefix
			tabstop.variableDefault = suffix
		} else {
			tabstop.idx = tabstopIdx
			tabstop.placeholder = suffix
		}
		return tabstop, nil
	}

	startPipeIdx := strings.Index(subtext, "|")
	endPipeIdx := strings.LastIndex(subtext, "|")
	if startPipeIdx > 0 && (endPipeIdx == len(subtext)-1) {
		// The text defines a tabstop with choices
		tabstopIdx, err := strconv.Atoi(subtext[0:startPipeIdx])
		if err != nil {
			return nil, err
		}
		choiceStr := subtext[startPipeIdx+1 : endPipeIdx]
		tabstop.idx = tabstopIdx
		tabstop.choices = strings.Split(choiceStr, ",")
		return tabstop, nil
	}

	return nil, errors.New("invalid subtext format")
}

func (s *Snippet) buildTemplate() {
	s.template = s.raw
	if s.locations == nil {
		s.locations = make(map[*TabStop]runesOff)
	} else {
		clear(s.locations)
	}

	total := len(s.tabStops)
	if total <= 0 {
		return
	}

	bytesOffDelta := 0
	for _, st := range s.tabStops {
		var updatedStr string
		var offset runesOff
		var delta int

		if st.variable != "" {
			// TODO: inject variable value here. Use default value for now.
			updatedStr, offset, delta = replaceAtIndex(
				s.template,
				st.variableDefault,
				bytesOffDelta+st.location.start,
				bytesOffDelta+st.location.end)
			bytesOffDelta += delta
		} else if st.idx >= 0 {
			if st.placeholder != "" {
				updatedStr, offset, delta = replaceAtIndex(
					s.template,
					st.placeholder,
					bytesOffDelta+st.location.start,
					bytesOffDelta+st.location.end)
				bytesOffDelta += delta

			} else if len(st.choices) > 0 {
				// We don't handle choices for now, so we just use the first choice.
				updatedStr, offset, delta = replaceAtIndex(
					s.template,
					st.choices[0],
					bytesOffDelta+st.location.start,
					bytesOffDelta+st.location.end)
				bytesOffDelta += delta

			} else {
				updatedStr, offset, delta = replaceAtIndex(
					s.template,
					"",
					bytesOffDelta+st.location.start,
					bytesOffDelta+st.location.end)
				bytesOffDelta += delta
			}
		}

		s.template = updatedStr
		s.locations[st] = offset
	}
}

func (s *Snippet) Raw() string {
	return s.raw
}

func (s *Snippet) Template() string {
	return s.template
}

func (s *Snippet) TabStops() []*TabStop {
	return s.tabStops
}

func (s *Snippet) TabStopSize() int {
	return len(s.tabStops)
}

func (s *Snippet) TabStopAt(idx int) *TabStop {
	idx = max(0, idx)
	idx = min(idx, len(s.tabStops)-1)
	return s.tabStops[idx]
}

func (s *Snippet) TabStopOff(idx int) (int, int) {
	idx = max(0, idx)
	idx = min(idx, len(s.tabStops)-1)
	ts := s.tabStops[idx]
	loc := s.locations[ts]
	return loc.start, loc.end
}

func replaceAtIndex(text string, replacement string, start, end int) (string, runesOff, int) {
	start = min(start, end)
	end = max(start, end)
	newText := text[:start] + replacement + text[end:]

	startOff := utf8.RuneCountInString(text[:start])
	endOff := utf8.RuneCountInString(replacement) + startOff
	off := runesOff{
		start: startOff,
		end:   endOff,
	}

	delta := len(replacement) - (end - start)

	return newText, off, delta
}
