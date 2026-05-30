package pages

import (
	"log"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"

	appevents "github.com/chapar-rest/chapar/internal/events"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/uiv2/widget"
	"github.com/google/uuid"
)

// EnvironmentPageDeps configures the environment editor tab.
type EnvironmentPageDeps struct {
	Env     *domain.Environment
	Repo    repository.RepositoryV2
	TabView *widget.TabView
	TabKey  string
}

// EnvironmentPage returns a builder for an environment detail tab body.
func EnvironmentPage(deps EnvironmentPageDeps) func(content *core.Frame) {
	env := deps.Env
	return func(content *core.Frame) {
		content.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Padding.Set(units.Dp(12))
			s.Gap.Set(units.Dp(6))
			s.Grow.Set(1, 1)
		})

		ensureKVIds := func() {
			for i := range env.Spec.Values {
				if env.Spec.Values[i].ID == "" {
					env.Spec.Values[i].ID = uuid.NewString()
				}
			}
		}

		save := func() {
			ensureKVIds()
			if err := deps.Repo.UpdateEnvironment(env); err != nil {
				log.Println("save environment:", err)
				core.MessageDialog(content, err.Error(), "Save failed")
				return
			}
			appevents.EnvironmentChangeTopic.Publish(env)
			if deps.TabView != nil && deps.TabKey != "" {
				deps.TabView.SetTabLabel(deps.TabKey, env.GetName())
			}
		}

		header := core.NewFrame(content)
		header.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Align.Items = styles.Center
			s.Gap.Set(units.Dp(8))
			s.Grow.Set(1, 0)
			s.Min.X.Dp(0)
		})

		titleWrap := core.NewFrame(header)
		titleWrap.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Min.X.Dp(0)
			s.Overflow.X = styles.OverflowHidden
		})
		title := widget.NewEditableLabel(titleWrap, env.GetName())
		title.OnCommit(func(name string) {
			env.SetName(name)
			save()
		})

		searchWrap := core.NewFrame(header)
		searchWrap.Styler(func(s *styles.Style) {
			s.Grow.Set(0, 0)
			s.Min.X.Dp(160)
			s.Max.X.Dp(220)
		})
		search := core.NewTextField(searchWrap)
		search.SetPlaceholder("Search items")
		search.SetTrailingIcon(icons.Search)
		search.SendChangeOnInput()

		filter := ""

		caption := core.NewFrame(content)
		caption.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Align.Items = styles.Center
			s.Justify.Content = styles.SpaceBetween
			s.Grow.Set(1, 0)
		})
		core.NewText(caption).
			SetType(core.TextBodySmall).
			SetText("Disabled items have no effect on your requests")

		tbl := core.NewTable(content)
		tbl.SetSlice(&env.Spec.Values)
		tbl.SetTableStyler(func(w core.Widget, s *styles.Style, row, col int) {
			if filter == "" || row < 0 || row >= len(env.Spec.Values) {
				return
			}
			kv := env.Spec.Values[row]
			if !kvMatchesFilter(kv, filter) {
				w.AsWidget().SetState(true, states.Invisible)
			}
		})
		tbl.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Gap.Set(units.Dp(2))
		})
		tbl.OnChange(func(e events.Event) {
			save()
		})

		addBtn := core.NewButton(caption).
			SetType(core.ButtonAction).
			SetIcon(icons.Add)
		addBtn.OnClick(func(e events.Event) {
			tbl.NewAt(-1)
			ensureKVIds()
			save()
		})

		search.OnChange(func(e events.Event) {
			filter = strings.TrimSpace(search.Text())
			tbl.Update()
		})
	}
}

func kvMatchesFilter(kv domain.KeyValue, q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(kv.Key), q) ||
		strings.Contains(strings.ToLower(kv.Value), q)
}
