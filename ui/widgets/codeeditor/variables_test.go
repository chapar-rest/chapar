package codeeditor

import "testing"

func TestFindVariableAtRuneOffset(t *testing.T) {
	text := `{"url": "{{baseURL}}/users/{{userId}}"}`

	tests := []struct {
		name    string
		runeOff int
		want    string
		ok      bool
	}{
		{name: "inside baseURL", runeOff: 10, want: "baseURL", ok: true},
		{name: "inside userId", runeOff: 28, want: "userId", ok: true},
		{name: "outside variable", runeOff: 0, ok: false},
		{name: "on slash", runeOff: 21, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := findVariableAtRuneOffset(text, tt.runeOff)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("name = %q, want %q", got, tt.want)
			}
		})
	}
}
