package ui

// WindowHost abstracts platform window operations for custom title bars.
// The runtime injects an implementation via SetWindow; headless builds leave it nil.
type WindowHost interface {
	CustomTitleBar() bool
	NativeControls() bool   // true when the OS draws window controls (macOS traffic lights)
	ControlsInset() float32 // leading px reserved for native controls; 0 elsewhere
	Close()
	Minimize()
	ToggleMaximize()
	IsMaximized() bool
	BeginMove() // start a window drag from the current pointer position
}
