package container

import (
	"github.com/google/uuid"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
)

// Column IDs shared by the key/value tables.
const (
	kvColEnable = "en"
	kvColKey    = "key"
	kvColValue  = "value"
	// KVColSecret marks a row whose value is encrypted at rest.
	KVColSecret = "sec"
	kvColAct    = "act"
)

func NewKVTable(idPrefix string, onChange func()) *ui.Table {
	t := ui.NewTable([]ui.TableColumn{
		{ID: kvColEnable, Label: "", Kind: ui.TableColCheckbox, Width: 36},
		{ID: kvColKey, Label: "Key", Kind: ui.TableColEditable, Width: 0},
		{ID: kvColValue, Label: "Value", Kind: ui.TableColEditable, Width: 0},
		{ID: kvColAct, Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: icons.Trash2, Tooltip: "Delete"}})
	t.HighlightSelected = false
	t.CollapseEmpty = true
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

// SecretsUI is how a table learns about secret rows: which ones the user chose
// to reveal, and what to do when a row is marked secret.
type SecretsUI struct {
	// Revealed reports whether the user asked to see this row's value.
	Revealed func(rowID string) bool
	// ToggleReveal flips that.
	ToggleReveal func(rowID string)
	// OnMarkSecret runs when a row is switched to secret, so the caller can
	// make sure a key exists before the value has to be encrypted.
	OnMarkSecret func(rowID string)
	// Locked reports a row whose value is still encrypted because the key is
	// missing; such a row cannot be edited or revealed.
	Locked func(rowID string) bool
}

// NewSecretKVTable is NewKVTable plus a lock column that marks a row secret and
// an eye that reveals a secret value.
func NewSecretKVTable(idPrefix string, onChange func(), s SecretsUI) *ui.Table {
	// The actions read the table they belong to, so it is declared first.
	var t *ui.Table
	t = ui.NewTable([]ui.TableColumn{
		{ID: kvColEnable, Label: "", Kind: ui.TableColCheckbox, Width: 36},
		{ID: kvColKey, Label: "Key", Kind: ui.TableColEditable, Width: 0},
		{ID: kvColValue, Label: "Value", Kind: ui.TableColEditable, Width: 0},
		{
			ID: KVColSecret, Label: "", Kind: ui.TableColToggle, Width: 40, Locked: true,
			IconOn: icons.Lock, IconOff: icons.LockOpen,
			TooltipOn: "Secret: encrypted when saved", TooltipOff: "Mark as secret",
		},
		{ID: kvColAct, Label: "", Kind: ui.TableColActions, Width: 76, Locked: true},
	}, []ui.TableAction{
		{
			Icon:    icons.Eye,
			Tooltip: "Show value",
			Visible: func(rowID string) bool { return rowIsSecret(t, rowID) && !s.isLocked(rowID) },
			IconFor: func(rowID string) icons.Icon {
				if s.revealed(rowID) {
					return icons.EyeOff
				}
				return icons.Eye
			},
			TooltipFor: func(rowID string) string {
				if s.revealed(rowID) {
					return "Hide value"
				}
				return "Show value"
			},
			OnClick: func(rowID string) {
				if s.ToggleReveal != nil {
					s.ToggleReveal(rowID)
				}
			},
		},
		{Icon: icons.Trash2, Tooltip: "Delete"},
	})
	t.HighlightSelected = false
	t.CollapseEmpty = true
	t.Masked = func(rowID, colID string) bool {
		if colID != kvColValue {
			return false
		}
		return rowIsSecret(t, rowID) && !s.revealed(rowID)
	}
	t.Actions[1].OnClick = func(rowID string) {
		t.RemoveRow(rowID)
		if onChange != nil {
			onChange()
		}
	}
	t.OnCellChange = func(rowID, colID, value string) {
		if colID == KVColSecret && value == "1" && s.OnMarkSecret != nil {
			s.OnMarkSecret(rowID)
		}
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

func (s SecretsUI) revealed(rowID string) bool {
	return s.Revealed != nil && s.Revealed(rowID)
}

func (s SecretsUI) isLocked(rowID string) bool {
	return s.Locked != nil && s.Locked(rowID)
}

func rowIsSecret(t *ui.Table, rowID string) bool {
	if t == nil {
		return false
	}
	for _, row := range t.Rows {
		if row.ID == rowID {
			return row.Cells[KVColSecret] == "1"
		}
	}
	return false
}

func LoadKV(t *ui.Table, items []domain.KeyValue) {
	rows := make([]ui.TableRow, 0, len(items))
	for _, kv := range items {
		id := kv.ID
		if id == "" {
			id = uuid.NewString()
		}
		cells := map[string]string{kvColKey: kv.Key, kvColValue: kv.Value}
		if kv.Secret {
			cells[KVColSecret] = "1"
		}
		rows = append(rows, ui.TableRow{
			ID:       id,
			Selected: kv.Enable,
			Cells:    cells,
		})
	}
	t.SetRows(rows)
}

func DumpKV(t *ui.Table) []domain.KeyValue {
	out := make([]domain.KeyValue, 0, len(t.Rows))
	for _, row := range t.Rows {
		out = append(out, domain.KeyValue{
			ID:     row.ID,
			Key:    row.Cells[kvColKey],
			Value:  row.Cells[kvColValue],
			Enable: row.Selected,
			Secret: row.Cells[KVColSecret] == "1",
		})
	}
	return out
}

func AddKVRow(t *ui.Table, onChange func()) {
	t.AddRow(ui.TableRow{
		ID:       uuid.NewString(),
		Selected: true,
		Cells:    map[string]string{kvColKey: "", kvColValue: ""},
	})
	if onChange != nil {
		onChange()
	}
}
