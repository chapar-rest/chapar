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
