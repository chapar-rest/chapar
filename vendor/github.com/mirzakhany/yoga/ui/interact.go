package ui

// interactStateFor builds interaction state for spec.resolve. When disabled,
// hover/press are suppressed so When(Disabled) patches apply cleanly.
func interactStateFor(disabled, hovered, pressed, focused bool) interactState {
	if disabled {
		return interactState{disabled: true}
	}
	return interactState{hovered: hovered, pressed: pressed, focused: focused}
}

func suppressHoverPressIfDisabled(disabled bool, hovered, pressed *bool) {
	if disabled {
		*hovered, *pressed = false, false
	}
}

// trackHover updates *hovered and marks paint when the value changes.
func trackHover(c *Ctx, hovered *bool, inside bool) {
	if *hovered == inside {
		return
	}
	*hovered = inside
	if c != nil {
		c.MarkNeedsPaint()
	}
}

// trackHoverIdx updates a hover index and marks paint when it changes.
func trackHoverIdx(c *Ctx, dst *int, v int) {
	if *dst == v {
		return
	}
	*dst = v
	if c != nil {
		c.MarkNeedsPaint()
	}
}

// trackBool updates *dst and marks paint when the value changes.
func trackBool(c *Ctx, dst *bool, v bool) {
	if *dst == v {
		return
	}
	*dst = v
	if c != nil {
		c.MarkNeedsPaint()
	}
}
