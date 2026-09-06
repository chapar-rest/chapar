package container

import (
	"fmt"
	"strconv"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// PrePostOpts configures pre/post request UI options.
type PrePostOpts struct {
	ID            string
	AllowPython   bool
	AllowSetEnv   bool
	FromOptions   []ui.SelectOption
	DefaultFrom   string
	DefaultStatus int
}

// PreRequestPane renders pre-request configuration.
func PreRequestPane(th *theme.Theme, deps Deps, pre *domain.PreRequest, opts PrePostOpts, scriptEd **ui.Editor, markDirty func()) ui.View {
	if pre == nil {
		return ui.Column()
	}
	typeOpts := []ui.SelectOption{
		{Label: "None", Value: domain.PrePostTypeNone},
		{Label: "Trigger request", Value: domain.PrePostTypeTriggerRequest},
	}
	if opts.AllowPython {
		typeOpts = append(typeOpts, ui.SelectOption{Label: "Python", Value: domain.PrePostTypePython})
	}
	if pre.Type == "" {
		pre.Type = domain.PrePostTypeNone
	}
	rows := []ui.View{
		ui.Select("pre-type-"+opts.ID, typeOpts).Width(200).
			Selected(optionIndex(pre.Type, typeOpts)).
			OnChange(func(v string) { pre.Type = v; markDirty() }),
	}
	switch pre.Type {
	case domain.PrePostTypeTriggerRequest:
		if pre.TriggerRequest == nil {
			pre.TriggerRequest = &domain.TriggerRequest{}
		}
		rows = append(rows, TriggerRequestPicker(th, deps, opts.ID, pre.TriggerRequest, markDirty))
	case domain.PrePostTypePython:
		if *scriptEd == nil {
			*scriptEd = ui.NewEditor([]byte(pre.Script), highlight.Noop{})
		}
		rows = append(rows, ui.ViewOf(*scriptEd).Grow(1))
	}
	return ui.Column(rows...).Gap(th.Spacing.S).Grow(1)
}

// PostRequestPane renders post-request configuration.
func PostRequestPane(th *theme.Theme, deps Deps, post *domain.PostRequest, opts PrePostOpts, scriptEd **ui.Editor, preview string, markDirty func()) ui.View {
	if post == nil {
		return ui.Column()
	}
	typeOpts := []ui.SelectOption{
		{Label: "None", Value: domain.PrePostTypeNone},
	}
	if opts.AllowSetEnv {
		typeOpts = append(typeOpts, ui.SelectOption{Label: "Set environment", Value: domain.PrePostTypeSetEnv})
	}
	if opts.AllowPython {
		typeOpts = append(typeOpts, ui.SelectOption{Label: "Python", Value: domain.PrePostTypePython})
	}
	if post.Type == "" {
		post.Type = domain.PrePostTypeNone
	}
	rows := []ui.View{
		ui.Select("post-type-"+opts.ID, typeOpts).Width(200).
			Selected(optionIndex(post.Type, typeOpts)).
			OnChange(func(v string) { post.Type = v; markDirty() }),
	}
	switch post.Type {
	case domain.PrePostTypeSetEnv:
		rows = append(rows, SetEnvForm(th, opts.ID, &post.PostRequestSet, opts.FromOptions, opts.DefaultFrom, markDirty))
		if preview != "" {
			rows = append(rows, ui.Text("Preview: "+preview).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
		}
	case domain.PrePostTypePython:
		if *scriptEd == nil {
			*scriptEd = ui.NewEditor([]byte(post.Script), highlight.Noop{})
		}
		rows = append(rows, ui.ViewOf(*scriptEd).Grow(1))
	}
	return ui.Column(rows...).Gap(th.Spacing.S).Grow(1)
}

func TriggerRequestPicker(th *theme.Theme, deps Deps, id string, tr *domain.TriggerRequest, markDirty func()) ui.View {
	opts := []ui.SelectOption{{Label: "None", Value: ""}}
	if deps.Catalog != nil {
		for _, col := range deps.Catalog.AllCollections() {
			for _, r := range col.Spec.Requests {
				opts = append(opts, ui.SelectOption{
					Label: col.MetaData.Name + " / " + domain.RequestDisplayName(r),
					Value: r.MetaData.ID + "|" + col.MetaData.ID,
				})
			}
		}
		for _, r := range deps.Catalog.StandaloneRequests() {
			opts = append(opts, ui.SelectOption{Label: domain.RequestDisplayName(r), Value: r.MetaData.ID + "|"})
		}
	}
	sel := ""
	if tr.RequestID != "" {
		sel = tr.RequestID + "|" + tr.CollectionID
	}
	return ui.Select("trigger-"+id, opts).Width(320).
		Selected(optionIndex(sel, opts)).
		OnChange(func(v string) {
			parts := splitTriggerValue(v)
			tr.RequestID = parts[0]
			tr.CollectionID = parts[1]
			markDirty()
		})
}

func splitTriggerValue(v string) [2]string {
	if v == "" {
		return [2]string{"", ""}
	}
	if idx := indexByte(v, '|'); idx >= 0 {
		return [2]string{v[:idx], v[idx+1:]}
	}
	return [2]string{v, ""}
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func SetEnvForm(th *theme.Theme, id string, set *domain.PostRequestSet, fromOpts []ui.SelectOption, defaultFrom string, markDirty func()) ui.View {
	if set.From == "" && defaultFrom != "" {
		set.From = defaultFrom
	}
	statusStr := ""
	if set.StatusCode > 0 {
		statusStr = strconv.Itoa(set.StatusCode)
	}
	pathLabel := "JSON Path"
	if set.From != domain.PostRequestSetFromResponseBody && set.From != "" {
		pathLabel = "Header / Key"
	}
	return ui.Column(
		ui.TextField("set-target-"+id, set.Target).Placeholder("Env key").
			OnChange(func(s string) { set.Target = s; markDirty() }),
		ui.TextField("set-status-"+id, statusStr).Placeholder("Status code (e.g. 200)").
			OnChange(func(s string) {
				n, _ := strconv.Atoi(s)
				set.StatusCode = n
				markDirty()
			}),
		ui.Select("set-from-"+id, fromOpts).Width(180).
			Selected(optionIndex(set.From, fromOpts)).
			OnChange(func(v string) { set.From = v; markDirty() }),
		ui.TextField("set-path-"+id, set.FromKey).Placeholder(pathLabel).
			OnChange(func(s string) { set.FromKey = s; markDirty() }),
	).Gap(th.Spacing.S)
}

// PreviewPostSet returns a preview value for a post-request set-env rule.
func PreviewPostSet(set domain.PostRequestSet, res *egress.Response) string {
	if res == nil || !set.IsValid() {
		return ""
	}
	code := res.StatusCode
	if code == 0 {
		code = res.StatueCode
	}
	if code != set.StatusCode {
		return ""
	}
	switch set.From {
	case domain.PostRequestSetFromResponseBody:
		if res.JSON != "" {
			data, err := jsonpath.Get(res.JSON, set.FromKey)
			if err == nil {
				if s, ok := data.(string); ok {
					return s
				}
				return fmt.Sprintf("%v", data)
			}
		}
	case domain.PostRequestSetFromResponseHeader:
		return res.ResponseHeaders[set.FromKey]
	case domain.PostRequestSetFromResponseCookie:
		for _, c := range res.Cookies {
			if c.Name == set.FromKey {
				return c.Value
			}
		}
	case domain.PostRequestSetFromResponseMetaData:
		for _, item := range res.ResponseMetadata {
			if item.Key == set.FromKey {
				return item.Value
			}
		}
	case domain.PostRequestSetFromResponseTrailers:
		for _, item := range res.Trailers {
			if item.Key == set.FromKey {
				return item.Value
			}
		}
	}
	return ""
}

func FlushPreScript(pre *domain.PreRequest, ed *ui.Editor) {
	if pre != nil && ed != nil && pre.Type == domain.PrePostTypePython {
		pre.Script = string(ed.Bytes())
	}
}

func FlushPostScript(post *domain.PostRequest, ed *ui.Editor) {
	if post != nil && ed != nil && post.Type == domain.PrePostTypePython {
		post.Script = string(ed.Bytes())
	}
}

func HTTPPostFromOptions() []ui.SelectOption {
	return []ui.SelectOption{
		{Label: "Body", Value: domain.PostRequestSetFromResponseBody},
		{Label: "Header", Value: domain.PostRequestSetFromResponseHeader},
		{Label: "Cookie", Value: domain.PostRequestSetFromResponseCookie},
	}
}

func GRPCPostFromOptions() []ui.SelectOption {
	return []ui.SelectOption{
		{Label: "Body", Value: domain.PostRequestSetFromResponseBody},
		{Label: "Metadata", Value: domain.PostRequestSetFromResponseMetaData},
		{Label: "Trailers", Value: domain.PostRequestSetFromResponseTrailers},
	}
}
