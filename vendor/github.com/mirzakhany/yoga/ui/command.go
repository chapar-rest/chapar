package ui

import "github.com/mirzakhany/yoga/icons"

// Command is a selectable palette entry: an action (with optional shortcut) or a
// navigation target such as a recent file. Everything in the palette is a
// Command; use Item for entries that are not keybound actions. Use Section for
// labeled separators between groups of entries.
type Command struct {
	id       string
	title    string
	detail   string // secondary line (path, description); preferred over group in the row
	group    string
	icon     icons.Icon
	shortcut Chord
	enabled  bool
	hidden   bool
	section  bool // non-selectable labeled separator
	run      func()
}

// Cmd starts a fluent command builder. Commands are enabled by default.
func Cmd(id string) *Command {
	return &Command{id: id, enabled: true}
}

// Item starts a palette entry that is not a keybound action (recent files,
// symbols, jumps, …). Same type as Cmd — only the name signals intent.
func Item(id string) *Command {
	return Cmd(id)
}

// Section starts a non-selectable labeled separator between palette groups.
// Arrow keys and clicks skip it; it is shown when at least one following entry
// (until the next Section) is visible.
func Section(title string) *Command {
	return &Command{
		id:      "__section:" + title,
		title:   title,
		section: true,
	}
}

// Title sets the palette display name (or section label).
func (c *Command) Title(s string) *Command { c.title = s; return c }

// Detail sets a secondary line under the title (file path, description, …).
// When set, it is shown instead of Group in the row; Group still participates
// in search filtering.
func (c *Command) Detail(s string) *Command { c.detail = s; return c }

// Group sets an optional category. Shown as the subtitle when Detail is empty.
func (c *Command) Group(s string) *Command { c.group = s; return c }

// Icon sets an optional leading icon.
func (c *Command) Icon(icon icons.Icon) *Command { c.icon = icon; return c }

// Shortcut parses and attaches a chord (e.g. "⌘S", "Mod+K"). Invalid strings are ignored.
func (c *Command) Shortcut(s string) *Command {
	if ch, err := ParseChord(s); err == nil {
		c.shortcut = ch
	}
	return c
}

// ShortcutChord attaches a pre-parsed chord.
func (c *Command) ShortcutChord(ch Chord) *Command {
	c.shortcut = ch
	return c
}

// Enable sets whether the command can run via shortcut or Enter.
func (c *Command) Enable(v bool) *Command { c.enabled = v; return c }

// Enabled is an alias for Enable.
func (c *Command) Enabled(v bool) *Command { return c.Enable(v) }

// Hide omits the command from the palette while keeping its shortcut active.
func (c *Command) Hide(v bool) *Command { c.hidden = v; return c }

// Hidden is an alias for Hide.
func (c *Command) Hidden(v bool) *Command { return c.Hide(v) }

// Run sets the action invoked when the command is selected or its shortcut fires.
func (c *Command) Run(fn func()) *Command { c.run = fn; return c }

// ID returns the command id.
func (c *Command) ID() string { return c.id }

// DisplayTitle returns the palette title (falls back to id).
func (c *Command) DisplayTitle() string {
	if c.title != "" {
		return c.title
	}
	return c.id
}

// IsSection reports whether this entry is a labeled separator.
func (c *Command) IsSection() bool { return c != nil && c.section }

// subtitle is the muted second line: Detail, or Group when Detail is empty.
func (c *Command) subtitle() string {
	if c.detail != "" {
		return c.detail
	}
	return c.group
}
