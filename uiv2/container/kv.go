package container

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/google/uuid"
	"github.com/mirzakhany/yoga/ui"
)

func NewKVTable(idPrefix string, onChange func()) *ui.Table {
	t := ui.NewTable([]ui.TableColumn{
		{ID: "en", Label: "", Kind: ui.TableColCheckbox, Width: 36},
		{ID: "key", Label: "Key", Kind: ui.TableColEditable, Width: 0},
		{ID: "value", Label: "Value", Kind: ui.TableColEditable, Width: 0},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: "delete", Tooltip: "Delete"}})
	t.Actions[0].OnClick = func(rowID string) {
		t.RemoveRow(rowID)
		if onChange != nil {
			onChange()
		}
	}
	t.OnCellChange = func(_, _, _ string) {
		if onChange != nil {
			onChange()
		}
	}
	t.OnSelectionChange = func() {
		if onChange != nil {
			onChange()
		}
	}
	_ = idPrefix
	return t
}

func LoadKV(t *ui.Table, items []domain.KeyValue) {
	rows := make([]ui.TableRow, 0, len(items))
	for _, kv := range items {
		id := kv.ID
		if id == "" {
			id = uuid.NewString()
		}
		rows = append(rows, ui.TableRow{
			ID:       id,
			Selected: kv.Enable,
			Cells:    map[string]string{"key": kv.Key, "value": kv.Value},
		})
	}
	t.SetRows(rows)
}

func DumpKV(t *ui.Table) []domain.KeyValue {
	out := make([]domain.KeyValue, 0, len(t.Rows))
	for _, row := range t.Rows {
		out = append(out, domain.KeyValue{
			ID:     row.ID,
			Key:    row.Cells["key"],
			Value:  row.Cells["value"],
			Enable: row.Selected,
		})
	}
	return out
}

func AddKVRow(t *ui.Table, onChange func()) {
	t.AddRow(ui.TableRow{
		ID:       uuid.NewString(),
		Selected: true,
		Cells:    map[string]string{"key": "", "value": ""},
	})
	if onChange != nil {
		onChange()
	}
}
