//go:build !android && !ios
// +build !android,!ios

package component

import "gioui.org/widget/material"

// PlatformTooltip creates a tooltip styled to the current platform
// (desktop or mobile) by choosing based on the OS. This choice may
// not always be appropriate as it only uses the OS to decide.
func PlatformTooltip(th *material.Theme, text string) Tooltip {
	return DesktopTooltip(th, text)
}
