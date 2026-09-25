package ui

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/input"
)

// Chord is a single-key shortcut with modifiers.
type Chord struct {
	Key  input.Key
	Mods input.Mod
	// primary is set when the chord was parsed with Mod/⌘/Cmd/Ctrl so Matches
	// accepts either Super or Ctrl.
	primary bool
}

// ParseChord parses strings like "⌘S", "Mod+K", "Ctrl+Shift+P", "F5".
// Mod, ⌘, Cmd, and Ctrl all mean the platform primary modifier.
func ParseChord(s string) (Chord, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Chord{}, fmt.Errorf("empty chord")
	}
	parts := splitChord(s)
	if len(parts) == 0 {
		return Chord{}, fmt.Errorf("empty chord")
	}
	var c Chord
	keyPart := parts[len(parts)-1]
	for _, p := range parts[:len(parts)-1] {
		switch normalizeModToken(p) {
		case "shift":
			c.Mods |= input.ModShift
		case "alt", "option", "opt":
			c.Mods |= input.ModAlt
		case "mod", "cmd", "command", "meta", "super", "ctrl", "control", "⌘":
			c.primary = true
			c.Mods |= input.ModSuper // Label prefers ⌘; Matches uses Primary()
		default:
			return Chord{}, fmt.Errorf("unknown modifier %q", p)
		}
	}
	key, ok := parseKeyToken(keyPart)
	if !ok {
		return Chord{}, fmt.Errorf("unknown key %q", keyPart)
	}
	c.Key = key
	return c, nil
}

// MustParseChord is ParseChord that panics on error.
func MustParseChord(s string) Chord {
	c, err := ParseChord(s)
	if err != nil {
		panic(err)
	}
	return c
}

func splitChord(s string) []string {
	// Unicode symbols without separators: ⌘⇧⌥⌃ followed by a key token.
	if strings.ContainsAny(s, "⌘⇧⌥⌃") && !strings.Contains(s, "+") {
		var parts []string
		rest := s
		for len(rest) > 0 {
			r, size := utf8.DecodeRuneInString(rest)
			switch r {
			case '⌘':
				parts = append(parts, "⌘")
				rest = rest[size:]
			case '⇧':
				parts = append(parts, "Shift")
				rest = rest[size:]
			case '⌥':
				parts = append(parts, "Alt")
				rest = rest[size:]
			case '⌃':
				parts = append(parts, "Ctrl")
				rest = rest[size:]
			default:
				parts = append(parts, rest)
				return parts
			}
		}
		return parts
	}
	raw := strings.Split(s, "+")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func normalizeModToken(p string) string {
	p = strings.TrimSpace(p)
	if p == "⌘" {
		return "⌘"
	}
	return strings.ToLower(p)
}

func parseKeyToken(p string) (input.Key, bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return input.KeyNone, false
	}
	lower := strings.ToLower(p)
	switch lower {
	case "left", "arrowleft":
		return input.KeyLeft, true
	case "right", "arrowright":
		return input.KeyRight, true
	case "up", "arrowup":
		return input.KeyUp, true
	case "down", "arrowdown":
		return input.KeyDown, true
	case "home":
		return input.KeyHome, true
	case "end":
		return input.KeyEnd, true
	case "backspace", "bksp":
		return input.KeyBackspace, true
	case "delete", "del":
		return input.KeyDelete, true
	case "enter", "return", "↵":
		return input.KeyEnter, true
	case "tab":
		return input.KeyTab, true
	case "escape", "esc":
		return input.KeyEscape, true
	case "space", " ":
		return input.KeySpace, true
	case "comma", ",":
		return input.KeyComma, true
	case "period", ".", "dot":
		return input.KeyPeriod, true
	case "slash", "/":
		return input.KeySlash, true
	case "minus", "-", "dash":
		return input.KeyMinus, true
	case "equal", "=", "equals":
		return input.KeyEqual, true
	case "[", "leftbracket":
		return input.KeyLeftBracket, true
	case "]", "rightbracket":
		return input.KeyRightBracket, true
	case "`", "backtick", "grave":
		return input.KeyBacktick, true
	case ";", "semicolon":
		return input.KeySemicolon, true
	case "'", "apostrophe", "quote":
		return input.KeyApostrophe, true
	case `\`, "backslash":
		return input.KeyBackslash, true
	case "f1":
		return input.KeyF1, true
	case "f2":
		return input.KeyF2, true
	case "f3":
		return input.KeyF3, true
	case "f4":
		return input.KeyF4, true
	case "f5":
		return input.KeyF5, true
	case "f6":
		return input.KeyF6, true
	case "f7":
		return input.KeyF7, true
	case "f8":
		return input.KeyF8, true
	case "f9":
		return input.KeyF9, true
	case "f10":
		return input.KeyF10, true
	case "f11":
		return input.KeyF11, true
	case "f12":
		return input.KeyF12, true
	}
	if len(p) == 1 {
		r := unicode.ToUpper(rune(p[0]))
		if r >= '0' && r <= '9' {
			return input.Key0 + input.Key(r-'0'), true
		}
		if r >= 'A' && r <= 'Z' {
			return letterKey(r), true
		}
	}
	return input.KeyNone, false
}

func letterKey(r rune) input.Key {
	switch r {
	case 'A':
		return input.KeyA
	case 'B':
		return input.KeyB
	case 'C':
		return input.KeyC
	case 'D':
		return input.KeyD
	case 'E':
		return input.KeyE
	case 'F':
		return input.KeyF
	case 'G':
		return input.KeyG
	case 'H':
		return input.KeyH
	case 'I':
		return input.KeyI
	case 'J':
		return input.KeyJ
	case 'K':
		return input.KeyK
	case 'L':
		return input.KeyL
	case 'M':
		return input.KeyM
	case 'N':
		return input.KeyN
	case 'O':
		return input.KeyO
	case 'P':
		return input.KeyP
	case 'Q':
		return input.KeyQ
	case 'R':
		return input.KeyR
	case 'S':
		return input.KeyS
	case 'T':
		return input.KeyT
	case 'U':
		return input.KeyU
	case 'V':
		return input.KeyV
	case 'W':
		return input.KeyW
	case 'X':
		return input.KeyX
	case 'Y':
		return input.KeyY
	case 'Z':
		return input.KeyZ
	}
	return input.KeyNone
}

// Label returns a compact display string for Kbd chips (e.g. "⌘S").
func (c Chord) Label() string {
	if c.Key == input.KeyNone {
		return ""
	}
	var b strings.Builder
	if c.primary || c.Mods.Has(input.ModSuper) || c.Mods.Has(input.ModCtrl) {
		b.WriteRune('⌘')
	}
	if c.Mods.Has(input.ModAlt) {
		b.WriteRune('⌥')
	}
	if c.Mods.Has(input.ModShift) {
		b.WriteRune('⇧')
	}
	b.WriteString(keyLabel(c.Key))
	return b.String()
}

func keyLabel(k input.Key) string {
	switch k {
	case input.KeyLeft:
		return "←"
	case input.KeyRight:
		return "→"
	case input.KeyUp:
		return "↑"
	case input.KeyDown:
		return "↓"
	case input.KeyHome:
		return "Home"
	case input.KeyEnd:
		return "End"
	case input.KeyBackspace:
		return "⌫"
	case input.KeyDelete:
		return "⌦"
	case input.KeyEnter:
		return "↵"
	case input.KeyTab:
		return "⇥"
	case input.KeyEscape:
		return "Esc"
	case input.KeySpace:
		return "Space"
	case input.KeyComma:
		return ","
	case input.KeyPeriod:
		return "."
	case input.KeySlash:
		return "/"
	case input.KeyMinus:
		return "-"
	case input.KeyEqual:
		return "="
	case input.KeyLeftBracket:
		return "["
	case input.KeyRightBracket:
		return "]"
	case input.KeyBacktick:
		return "`"
	case input.KeySemicolon:
		return ";"
	case input.KeyApostrophe:
		return "'"
	case input.KeyBackslash:
		return `\`
	case input.KeyF1:
		return "F1"
	case input.KeyF2:
		return "F2"
	case input.KeyF3:
		return "F3"
	case input.KeyF4:
		return "F4"
	case input.KeyF5:
		return "F5"
	case input.KeyF6:
		return "F6"
	case input.KeyF7:
		return "F7"
	case input.KeyF8:
		return "F8"
	case input.KeyF9:
		return "F9"
	case input.KeyF10:
		return "F10"
	case input.KeyF11:
		return "F11"
	case input.KeyF12:
		return "F12"
	case input.Key0, input.Key1, input.Key2, input.Key3, input.Key4,
		input.Key5, input.Key6, input.Key7, input.Key8, input.Key9:
		return string(rune('0' + int(k-input.Key0)))
	}
	if s := letterLabel(k); s != "" {
		return s
	}
	return "?"
}

func letterLabel(k input.Key) string {
	for r := 'A'; r <= 'Z'; r++ {
		if letterKey(r) == k {
			return string(r)
		}
	}
	return ""
}

// Matches reports whether the key event activates this chord.
// Primary (Mod/⌘/Cmd/Ctrl) matches Super or Ctrl; Shift and Alt must match exactly.
func (c Chord) Matches(ev input.KeyEvent) bool {
	if c.Key == input.KeyNone || ev.Key != c.Key {
		return false
	}
	wantShift := c.Mods.Has(input.ModShift)
	wantAlt := c.Mods.Has(input.ModAlt)
	gotShift := ev.Mods.Has(input.ModShift)
	gotAlt := ev.Mods.Has(input.ModAlt)
	if wantShift != gotShift || wantAlt != gotAlt {
		return false
	}
	if c.primary {
		return ev.Mods.Primary()
	}
	wantPrimary := c.Mods.Has(input.ModSuper) || c.Mods.Has(input.ModCtrl)
	if wantPrimary {
		return ev.Mods.Primary()
	}
	// No primary required: reject if primary is held.
	return !ev.Mods.Primary()
}

// Valid reports whether the chord has a key.
func (c Chord) Valid() bool { return c.Key != input.KeyNone }
