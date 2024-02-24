# component

[![Go Reference](https://pkg.go.dev/badge/gioui.org/x/component.svg)](https://pkg.go.dev/gioui.org/x/component)

This package provides various material design components for [gio](https://gioui.org).

## State

This package has no stable API, and should always be locked to a particular commit with
go modules.

The included components attempt to conform to the [material design specifications](https://material.io/components/)
whenever possible, but they may not support unusual style tweaks or especially exotic
configurations.

## Implemented Components

The list of currently-Implemented components follows:

### Navigation Drawer (static and modal)

The navigation drawer [specified here](https://material.io/components/navigation-drawer) is mostly implemented by the type
`NavDrawer`, and the modal variant can be created with a `ModalNavDrawer`. The modal variant looks like this:

![modal navigation drawer example screenshot](https://git.sr.ht/~whereswaldon/gio-x/blob/main/component/img/modal-nav.png)

Features:
- Animated drawer open/close.
- Navigation items respond to hovering.
- Navigation selection is animated.
- Navigation item icons are optional.
- Content can be anchored to the bottom of the drawer for pairing with a bottom app bar.

Modal features:
- Swipe or touch scrim to close the drawer.

Known issues:

- API targets a fairly static and simplistic menu. Sub-sections with dividers are not yet supported. An API-driven way to traverse the current menu options is also not yet supported. Contributions welcome!

### App Bar (Top and Bottom)

The App Bar [specified here](https://material.io/components/app-bars-top) is mostly implemented by the type
`AppBar`. It looks like this:

Normal state:

![modal navigation drawer example screenshot](https://git.sr.ht/~whereswaldon/gio-x/blob/main/component/img/app-bar-top.png)

Contextual state:

![modal navigation drawer example screenshot](https://git.sr.ht/~whereswaldon/gio-x/blob/main/component/img/app-bar-top-contextual.png)

Features:
 - Action buttons and overflow menu contents can be changed easily.
 - Overflow button disappears when no items overflow.
 - Overflow menu can be dismissed by touching the scrim outside of it.
 - Action items disapper into overflow when screen is too narrow to fit them. This is animated.
 - Navigation button icon is customizable, and the button is not drawn if no icon is provided.
 - Contextual app bar can be triggered and dismissed programatically.
 - Bar supports use as a top and bottom app bar (animates the overflow menu in the proper direction).

Known Issues:
 - Compact and prominent App Bars are not yet implemented.

### Side sheet (static and modal)

Side sheets ([specified here](https://material.io/components/sheets-side)) are implemented by the `Sheet` and `ModalSheet` types.

Features:
- Animated appear/disappear

Modal features:
- Swipe to close
- Touch scrim to close

Known Issues:
- Only sheets anchored on the left are currently supported (contributions welcome!)

### Text Fields

Text Fields ([specified here](https://material.io/components/text-fields)) are implemented by the `TextField` type.

Features:
- Animated label transition when selected
- Responds to hover events
- Exposes underlying gio editor

Known Issues:
- Icons, hint text, error text, prefix/suffix, and other features are not yet implemented.

### Dividers

The `Divider` type implements [material dividers](https://material.io/components/dividers). You can customize the insets
embedded in the type to change which kind of divider it is. Use the constructor
functions to create nice defaults.

### Surfaces

The `Surface` type is a rounded rectangle with a background color and a drop
shadow. This isn't a material component per se, but is a useful building block
nonetheless.

### Menu

The `Menu` type defines contextual menus as described [here](https://material.io/components/menus).

![first menu example screenshot](https://git.sr.ht/~whereswaldon/gio-x/blob/main/component/img/menu1.png)

Known issues:
- Does not support nested submenus (yet).

The `MenuItem` type provides widgets suitable for use within the Menu, though
any widget can be used. Here are some `MenuItem`s in action:

![second menu example screenshot](https://git.sr.ht/~whereswaldon/gio-x/blob/main/component/img/menu2.png)

### ContextArea

The `ContextArea` type is a helper type that defines an area that accepts
right-clicks. This area will display a widget when clicked (anchored at the
click position). The displayed widget is overlaid on other content, and is
therefore useful in displaying contextual menus.

Known issues:
- the heuristic that ContextArea uses to attempt to avoid off-screen drawing of
  its contextual content can fail or backfire. Suggestions for improving this
  are welcome.

### Tooltips

The `Tooltip`, `TipArea`, and `TipIconButtonStyle` types define a tooltip, a contextual area for displaying tooltips (on hover and long-press), and a wrapper around `material.IconButtonStyle` that provides a tooltip for the button.

![tooltip example screenshot](https://git.sr.ht/~whereswaldon/gio-x/blob/main/component/img/tooltip.png)
