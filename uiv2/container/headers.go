package container

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// HeadersPane shows an editable headers table and optional inherited collection headers.
func HeadersPane(th *theme.Theme, id string, headers *ui.Table, collectionID string, catalog Catalog, markDirty func()) ui.View {
	rows := []ui.View{
		ui.Row(
			ui.Text("Headers"),
			ui.Spacer(),
			ui.IconButton("hdr-add-"+id, icons.Plus).OnClick(func() {
				AddKVRow(headers, markDirty)
			}),
		).PaddingXY(0, th.Spacing.S),
		ui.ViewOf(headers).Grow(1),
	}
	if collectionID != "" && catalog != nil {
		if col := catalog.CollectionByID(collectionID); col != nil && len(col.Spec.Headers) > 0 {
			inherited := ui.Column(
				ui.Text("Inherited from collection").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
				ui.Caption("Local headers have higher priority than inherited headers."),
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
