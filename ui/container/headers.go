package container

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// HeadersPane shows an editable headers table and optional inherited collection headers.
func HeadersPane(th *theme.Theme, id string, headers *ui.Table, collectionID string, catalog Catalog, markDirty func()) ui.View {
	return kvPane(th, "hdr-add-"+id, "Headers", "Local headers have higher priority than inherited headers.",
		headers, collectionID, catalog, markDirty)
}

// MetadataPane is HeadersPane for gRPC metadata, without a title: the tab
// holds nothing else. Collection headers are sent as metadata too, so they
// are listed as inherited.
func MetadataPane(th *theme.Theme, id string, metadata *ui.Table, collectionID string, catalog Catalog, markDirty func()) ui.View {
	return kvPane(th, "md-add-"+id, "", "Local metadata has higher priority than inherited headers.",
		metadata, collectionID, catalog, markDirty)
}

func kvPane(th *theme.Theme, addID, title, priorityNote string, table *ui.Table, collectionID string, catalog Catalog, markDirty func()) ui.View {
	var head []ui.View
	if title != "" {
		head = append(head, ui.Text(title))
	}
	head = append(head,
		ui.Spacer(),
		ui.IconButton(addID, icons.Plus).Tooltip("Add").OnClick(func() {
			AddKVRow(table, markDirty)
		}),
	)
	headRow := ui.Row(head...)
	if title != "" {
		headRow = headRow.PaddingXY(0, th.Spacing.S)
	}
	rows := []ui.View{
		headRow,
		ui.ViewOf(table).Grow(1),
	}
	if collectionID != "" && catalog != nil {
		if col := catalog.CollectionByID(collectionID); col != nil && len(col.Spec.Headers) > 0 {
			inherited := ui.Column(
				ui.Text("Inherited from collection").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
				ui.Caption(priorityNote),
			).Gap(th.Spacing.XS).MarginTop(th.Spacing.M)
			for _, h := range col.Spec.Headers {
				if !h.Enable {
					continue
				}
				inherited = ui.Column(inherited, ui.Text(h.Key+": "+h.Value).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
			}
			rows = append(rows, inherited)
		}
	}
	return ui.Column(rows...).Gap(th.Spacing.S).Grow(1)
}
