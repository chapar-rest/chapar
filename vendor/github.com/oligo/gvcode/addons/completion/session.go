package completion

import (
	"slices"

	"github.com/oligo/gvcode"
)

type triggerKind uint8

const (
	autoTrigger triggerKind = iota
	charTrigger
	keyTrigger
)

type triggerState struct {
	triggerKind triggerKind
	// the activated completor.
	completor    *delegatedCompletor
	triggered    bool
	triggerChars string
}

// A session is started when some trigger is activated, and is destroyed when
// the completion is canceled or confirmed.
type session struct {
	ctx      gvcode.CompletionContext
	state    *triggerState
	canceled bool
	// buffered text while the user types.
	prefix []rune
	// input range of the cursor since when the session started and when completion
	// confirmed.
	prefixRange gvcode.EditRange
	// Full candidates from the completor.
	candidates []gvcode.CompletionCandidate
}

func newSession(completor *delegatedCompletor, kind triggerKind) *session {
	return &session{
		state: &triggerState{
			triggerKind: kind,
			completor:   completor,
			triggered:   true,
		},
	}
}

var terminatingChars = []rune{
	'{', '}', '(', ')', ',', ';', ' ', '\n', '\t',
}

func hasTerminateChar(input string) bool {
	if input == "" {
		return false
	}

	return slices.Contains(terminatingChars, []rune(input)[0])
}

func (s *session) Update(ctx gvcode.CompletionContext) []gvcode.CompletionCandidate {
	if s.canceled {
		return nil
	}

	if s.state.triggered {
		s.candidates = s.state.completor.Suggest(ctx)
		s.state.triggerChars = ctx.Input
		s.state.triggered = false
		s.prefix = s.prefix[:0]
		s.prefixRange = gvcode.EditRange{}
		//log.Printf("triggered new completion, candidates: %d, trigger char: %s", len(s.candidates), s.state.triggerChars)
	}

	if hasTerminateChar(ctx.Input) {
		s.makeInvalid()
		return nil
	}

	s.ctx = ctx

	if ctx.Input != "" && isSymbolChar([]rune(s.ctx.Input)[0]) {
		s.prefix = append(s.prefix, []rune(ctx.Input)...)
		if s.prefixRange == (gvcode.EditRange{}) {
			start := ctx.Position
			start.Column = max(0, start.Column-len([]rune(ctx.Input)))
			start.Runes = 0
			s.prefixRange.Start = start
		}
		s.prefixRange.End = ctx.Position
		s.prefixRange.End.Runes = 0
	}

	return s.state.completor.FilterAndRank(string(s.prefix), s.candidates)

}

func (s *session) makeInvalid() {
	s.canceled = true
	s.prefix = s.prefix[:0]
	s.prefixRange = gvcode.EditRange{}
	s.candidates = s.candidates[:0]
}

func (s *session) IsValid() bool {
	return s != nil && s.state != nil && !s.canceled
}

// Prefix returns text buffered since the session is triggered.
func (s *session) Prefix() string {
	return string(s.prefix)
}

func (s *session) PrefixRange() gvcode.EditRange {
	return s.prefixRange
}

func (s *session) Completor() *delegatedCompletor {
	return s.state.completor
}
