package widgets

import (
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

func MaterialIcons(name string, theme *chapartheme.Theme) material.LabelStyle {
	l := material.Label(theme.Material(), unit.Sp(24), "")
	l.Font.Typeface = "MaterialIcons"
	l.Text = name
	return l
}

var DeleteIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionDelete)
	return icon
}()

var CircleIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ImageLens)
	return icon
}()

var SaveIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentSave)
	return icon
}()

var MenuIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationMenu)
	return icon
}()

var CopyIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentContentCopy)
	return icon
}()

var SearchIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionSearch)
	return icon
}()

var HomeIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionHome)
	return icon
}()

var SwapHoriz *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionSwapHoriz)
	return icon
}()

var SettingsIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionSettings)
	return icon
}()

var OtherIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionHelp)
	return icon
}()

var HeartIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionFavorite)
	return icon
}()

var PlusIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentAdd)
	return icon
}()

var EditIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentCreate)
	return icon
}()

var VisibilityIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionVisibility)
	return icon
}()

var CloseIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationClose)
	return icon
}()

var ArrowDropDownIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationArrowDropDown)
	return icon
}()

var ForwardIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationChevronRight)
	return icon
}()

var ExpandIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationExpandMore)
	return icon
}()

var FileFolderIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.FileFolder)
	return icon
}()

var UploadIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.FileFileUpload)
	return icon
}()

var MoreVertIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationMoreVert)
	return icon
}()

var LogsIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionSubject)
	return icon
}()

var DarkIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionSubject)
	return icon
}()

var ConsoleIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.HardwareDesktopMac)
	return icon
}()

var TunnelIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionSwapVerticalCircle)
	return icon
}()

var SendIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentSend)
	return icon
}()

var WorkspacesIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationApps)
	return icon
}()

var RefreshIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationRefresh)
	return icon
}()

var CleanIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.EditorFormatColorText)
	return icon
}()
