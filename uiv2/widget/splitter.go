package widget

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// SplitterConfig controls the appearance of handles on a [Splitter].
type SplitterConfig struct {
	// LineWidth is the thickness of the divider line along the split axis.
	LineWidth units.Value
}

// DefaultSplitterConfig returns the standard Chapar splitter styling.
func DefaultSplitterConfig() SplitterConfig {
	return SplitterConfig{LineWidth: units.Dp(2)}
}

// Splitter is a [core.Splits] with Chapar line-style, hideable divider handles.
type Splitter struct {
	*core.Splits
	lineWidth      units.Value
	handlesVisible bool
}

// NewSplits creates a splits widget with styled handles using [DefaultSplitterConfig].
// Use [Splitter.SetHandlesVisible] to show or hide handles when a pane is collapsed.
func NewSplits(parent ...tree.Node) *Splitter {
	return NewSplitsWithConfig(DefaultSplitterConfig(), parent...)
}

// NewSplitsWithConfig creates a splits widget with styled handles and the given config.
func NewSplitsWithConfig(cfg SplitterConfig, parent ...tree.Node) *Splitter {
	lineWidth := cfg.LineWidth
	if lineWidth.Value == 0 && lineWidth.Custom == nil {
		lineWidth = units.Dp(2)
	}

	s := &Splitter{
		Splits:         core.NewSplits(parent...),
		lineWidth:      lineWidth,
		handlesVisible: true,
	}
	s.applyHandleStyle()
	return s
}

func (s *Splitter) applyHandleStyle() {
	s.UpdateWidget()
	if s.Parts == nil {
		return
	}

	styleHandle := func(h *core.Handle) {
		h.FinalStyler(func(st *styles.Style) {
			st.Border.Radius.Zero()
			st.Margin.Zero()
			st.Padding.Zero()

			if !s.handlesVisible {
				st.Min.Zero()
				st.Max.Zero()
				st.Grow.Set(0, 0)
				st.Background = nil
				return
			}

			if st.Direction == styles.Row {
				// Vertical line between columns: fixed width, full height.
				st.Min.X = s.lineWidth
				st.Max.X = s.lineWidth
				st.Grow.Set(0, 1)
				st.Min.Y.Dp(0)
				st.Max.Y.Dp(0)
			} else {
				// Horizontal line between rows: fixed height, full width.
				st.Min.Y = s.lineWidth
				st.Max.Y = s.lineWidth
				st.Grow.Set(1, 0)
				st.Min.X.Dp(0)
				st.Max.X.Dp(0)
			}

			if st.Is(states.Hovered) || st.Is(states.Active) || st.Is(states.Sliding) {
				st.Background = colors.Scheme.Primary.Base
			} else {
				st.Background = colors.Scheme.OutlineVariant
			}
		})
		s.syncHandleVisible(h)
	}

	s.Parts.SetOnChildAdded(func(n tree.Node) {
		if h, ok := n.(*core.Handle); ok {
			styleHandle(h)
		}
	})
	for i := range s.Parts.NumChildren() {
		if h, ok := s.Parts.Child(i).(*core.Handle); ok {
			styleHandle(h)
		}
	}
}

func (s *Splitter) syncHandleVisible(h *core.Handle) {
	if h == nil {
		return
	}
	h.SetState(!s.handlesVisible, states.Invisible)
	h.Restyle()
}

// OnResize fires whenever the user drags a handle, with the current split proportions.
func (s *Splitter) OnResize(fn func(splits []float32)) {
	s.OnChange(func(e events.Event) { fn(s.Splits.Splits()) })
}

// SetHandlesVisible shows or hides all split handles without affecting pane content.
func (s *Splitter) SetHandlesVisible(visible bool) {
	if s == nil || s.Splits == nil {
		return
	}
	s.handlesVisible = visible
	s.UpdateWidget()
	if s.Parts == nil {
		return
	}
	for i := range s.Parts.NumChildren() {
		if h, ok := s.Parts.Child(i).(*core.Handle); ok {
			s.syncHandleVisible(h)
		}
	}
	s.NeedsLayout()
	s.Update()
}
