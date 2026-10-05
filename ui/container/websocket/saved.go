package websocket

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/container"
)

// The Saved tab keeps messages with the request, to send again later.

func (c *Container) newSavedTable() *ui.Table {
	t := ui.NewTable([]ui.TableColumn{
		{ID: "name", Label: "Name", Kind: ui.TableColEditable, Width: 160},
		{ID: "format", Label: "Format", Kind: ui.TableColText, Width: 80, Locked: true},
		{ID: "body", Label: "Message", Kind: ui.TableColText, Locked: true},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 100, Locked: true},
	}, []ui.TableAction{
		{Icon: icons.SendHorizontal, Tooltip: "Send", OnClick: c.sendSaved},
		{Icon: icons.FileInput, Tooltip: "Load into the editor", OnClick: c.loadSavedIntoEditor},
		{Icon: icons.Trash2, Tooltip: "Delete", OnClick: c.deleteSaved},
	})
	t.MinHeight = 120
	t.OnCellChange = func(rowID, colID, value string) {
		if colID != "name" {
			return
		}
		if m := c.savedMessage(rowID); m != nil {
			m.Name = value
			c.markDirty()
		}
	}
	t.OnRowActivate = c.loadSavedIntoEditor
	return t
}

func (c *Container) loadSaved() {
	msgs := c.req.Spec.WebSocket.SavedMessages
	rows := make([]ui.TableRow, 0, len(msgs))
	for _, m := range msgs {
		rows = append(rows, ui.TableRow{
			ID: m.ID,
			Cells: map[string]string{
				"name":   m.Name,
				"format": formatLabel(m.Format),
				"body":   oneLine(m.Body),
			},
		})
	}
	c.saved.SetRows(rows)
}

func (c *Container) savedMessage(id string) *domain.WebSocketMessage {
	msgs := c.req.Spec.WebSocket.SavedMessages
	for i := range msgs {
		if msgs[i].ID == id {
			return &msgs[i]
		}
	}
	return nil
}

// saveDraft keeps the message being written.
func (c *Container) saveDraft() {
	w := c.req.Spec.WebSocket
	body := string(c.msgEd.Bytes())
	if body == "" {
		c.deps.Toast("Write a message first")
		return
	}
	w.SavedMessages = append(w.SavedMessages, domain.WebSocketMessage{
		ID:     uuid.NewString(),
		Name:   fmt.Sprintf("Message %d", len(w.SavedMessages)+1),
		Body:   body,
		Format: w.MessageFormat,
	})
	c.loadSaved()
	c.markDirty()
	c.deps.Toast("Message saved; rename it in the Saved tab")
}

func (c *Container) sendSaved(id string) {
	if m := c.savedMessage(id); m != nil {
		c.sendMessage(m.Format, m.Body)
	}
}

func (c *Container) loadSavedIntoEditor(id string) {
	m := c.savedMessage(id)
	if m == nil {
		return
	}
	c.setFormat(m.Format)
	c.msgEd.SetText(m.Body)
	c.reqActive = tabMessage
	c.markDirty()
}

func (c *Container) deleteSaved(id string) {
	w := c.req.Spec.WebSocket
	for i := range w.SavedMessages {
		if w.SavedMessages[i].ID == id {
			w.SavedMessages = append(w.SavedMessages[:i], w.SavedMessages[i+1:]...)
			break
		}
	}
	c.loadSaved()
	c.markDirty()
}

func (c *Container) savedPane(th *theme.Theme) ui.View {
	if len(c.req.Spec.WebSocket.SavedMessages) == 0 {
		return ui.Column(
			container.MutedParagraph("No saved messages. Write one in the Message tab and choose Save message."),
		).Grow(1)
	}
	return ui.Column(
		container.MutedParagraph("Double-click a row to load it into the editor."),
		ui.ViewOf(c.saved).Grow(1),
	).Gap(th.Spacing.S).Grow(1)
}

func formatLabel(format string) string {
	switch format {
	case domain.WebSocketFormatJSON:
		return "JSON"
	case domain.WebSocketFormatBinary:
		return "Binary"
	}
	return "Text"
}
