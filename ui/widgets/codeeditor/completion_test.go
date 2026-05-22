package codeeditor

import "testing"

func TestTemplateContextBeforeCaret(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		runeOff     int
		wantOK      bool
		wantPartial string
		wantStart   int
	}{
		{
			name:      "after opening braces",
			text:      `"url": "{{`,
			runeOff:   len([]rune(`"url": "{{`)),
			wantOK:    true,
			wantStart: len([]rune(`"url": "`)),
		},
		{
			name:        "partial name",
			text:        `"url": "{{base`,
			runeOff:     len([]rune(`"url": "{{base`)),
			wantOK:      true,
			wantPartial: "base",
			wantStart:   len([]rune(`"url": "`)),
		},
		{
			name:    "single brace",
			text:    `"url": "{`,
			runeOff: len([]rune(`"url": "{`)),
			wantOK:  false,
		},
		{
			name:    "outside template",
			text:    `"url": "https://example.com"`,
			runeOff: len([]rune(`"url": "https://example.com"`)),
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, partial, ok := templateContextBeforeCaret(tt.text, tt.runeOff)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if partial != tt.wantPartial {
				t.Fatalf("partial = %q, want %q", partial, tt.wantPartial)
			}
			if ok && start != tt.wantStart {
				t.Fatalf("start = %d, want %d", start, tt.wantStart)
			}
		})
	}
}

func TestTemplateClosingSuffix(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		runeOff int
		want    string
	}{
		{
			name:    "both braces already present",
			text:    `{{}}`,
			runeOff: 2,
			want:    "",
		},
		{
			name:    "after partial with auto closed braces",
			text:    `{{base}}`,
			runeOff: len([]rune(`{{base`)),
			want:    "",
		},
		{
			name:    "no closing braces",
			text:    `{{`,
			runeOff: 2,
			want:    "}}",
		},
		{
			name:    "single closing brace present",
			text:    `{{}`,
			runeOff: 2,
			want:    "}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := templateClosingSuffix(tt.text, tt.runeOff)
			if got != tt.want {
				t.Fatalf("suffix = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAdvanceCaretPastClosingBraces(t *testing.T) {
	text := `{{baseURL}}`
	got := advanceCaretPastClosingBraces(text, len([]rune(`{{baseURL`)))
	want := len([]rune(text))
	if got != want {
		t.Fatalf("caret = %d, want %d", got, want)
	}
}
