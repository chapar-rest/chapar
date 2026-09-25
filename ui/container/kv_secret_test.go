package container

import (
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

func TestSecretKVTableRoundTrip(t *testing.T) {
	tbl := NewSecretKVTable("env-1", nil, SecretsUI{})
	LoadKV(tbl, []domain.KeyValue{
		{ID: "1", Key: "host", Value: "example.com", Enable: true},
		{ID: "2", Key: "token", Value: "swordfish", Enable: true, Secret: true},
	})

	values := DumpKV(tbl)
	if len(values) != 2 {
		t.Fatalf("values: %+v", values)
	}
	if values[0].Secret {
		t.Fatal("a plain row should not come back secret")
	}
	if !values[1].Secret || values[1].Value != "swordfish" {
		t.Fatalf("secret row: %+v", values[1])
	}
}

func TestSecretKVTableMasksUntilRevealed(t *testing.T) {
	revealed := map[string]bool{}
	tbl := NewSecretKVTable("env-1", nil, SecretsUI{
		Revealed:     func(rowID string) bool { return revealed[rowID] },
		ToggleReveal: func(rowID string) { revealed[rowID] = !revealed[rowID] },
	})
	LoadKV(tbl, []domain.KeyValue{
		{ID: "1", Key: "host", Value: "example.com", Enable: true},
		{ID: "2", Key: "token", Value: "swordfish", Enable: true, Secret: true},
	})

	if tbl.Masked("1", "value") {
		t.Fatal("a plain value should not be masked")
	}
	if !tbl.Masked("2", "value") {
		t.Fatal("a secret value should be masked")
	}
	if tbl.Masked("2", "key") {
		t.Fatal("only the value column is masked")
	}

	revealed["2"] = true
	if tbl.Masked("2", "value") {
		t.Fatal("a revealed value should not be masked")
	}
}

func TestSecretKVTableMarkingSecretAsksForKey(t *testing.T) {
	var marked []string
	tbl := NewSecretKVTable("env-1", nil, SecretsUI{
		OnMarkSecret: func(rowID string) { marked = append(marked, rowID) },
	})
	LoadKV(tbl, []domain.KeyValue{
		{ID: "1", Key: "token", Value: "swordfish", Enable: true},
	})

	tbl.OnCellChange("1", KVColSecret, "1")
	if len(marked) != 1 || marked[0] != "1" {
		t.Fatalf("marking a row secret should ask for a key: %v", marked)
	}

	tbl.OnCellChange("1", KVColSecret, "")
	if len(marked) != 1 {
		t.Fatalf("unmarking should not ask for a key: %v", marked)
	}
}

func TestSecretKVTableRevealHiddenOnPlainAndLockedRows(t *testing.T) {
	locked := map[string]bool{"3": true}
	tbl := NewSecretKVTable("env-1", nil, SecretsUI{
		Locked: func(rowID string) bool { return locked[rowID] },
	})
	LoadKV(tbl, []domain.KeyValue{
		{ID: "1", Key: "host", Value: "example.com", Enable: true},
		{ID: "2", Key: "token", Value: "swordfish", Enable: true, Secret: true},
		{ID: "3", Key: "sealed", Value: "enc:v1:x", Enable: true, Secret: true},
	})

	reveal := tbl.Actions[0]
	if reveal.Visible("1") {
		t.Fatal("a plain row has nothing to reveal")
	}
	if !reveal.Visible("2") {
		t.Fatal("a secret row should offer reveal")
	}
	if reveal.Visible("3") {
		t.Fatal("a locked row cannot be revealed")
	}
}
