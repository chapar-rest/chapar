// SPDX-License-Identifier: Unlicense OR MIT

package explorer

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"golang.org/x/sys/windows"
)

var (
	// https://docs.microsoft.com/en-us/windows/win32/api/commdlg/
	_Dialog32 = windows.NewLazySystemDLL("comdlg32.dll")

	_GetSaveFileName = _Dialog32.NewProc("GetSaveFileNameW")
	_GetOpenFileName = _Dialog32.NewProc("GetOpenFileNameW")

	// https://docs.microsoft.com/en-us/windows/win32/api/commdlg/ns-commdlg-openfilenamew
	_FlagFileMustExist    = uint32(0x00001000)
	_FlagForceShowHidden  = uint32(0x10000000)
	_FlagOverwritePrompt  = uint32(0x00000002)
	_FlagDisableLinks     = uint32(0x00100000)
	_FlagAllowMultiSelect = uint32(0x00000200)
	_FlagExplorer         = uint32(0x00080000)

	_FilePathLength       = uint32(65535)
	_OpenFileStructLength = uint32(unsafe.Sizeof(_OpenFileName{}))
)

type (
	// _OpenFileName is defined at https://docs.microsoft.com/pt-br/windows/win32/api/commdlg/ns-commdlg-openfilenamew
	_OpenFileName struct {
		StructSize      uint32
		Owner           uintptr
		Instance        uintptr
		Filter          *uint16
		CustomFilter    *uint16
		MaxCustomFilter uint32
		FilterIndex     uint32
		File            *uint16
		MaxFile         uint32
		FileTitle       *uint16
		MaxFileTitle    uint32
		InitialDir      *uint16
		Title           *uint16
		Flags           uint32
		FileOffset      uint16
		FileExtension   uint16
		DefExt          *uint16
		CustData        uintptr
		FnHook          uintptr
		TemplateName    *uint16
		PvReserved      uintptr
		DwReserved      uint32
		FlagsEx         uint32
	}
)

type explorer struct{}

func newExplorer(_ *app.Window) *explorer {
	return &explorer{}
}

func (e *Explorer) listenEvents(evt event.Event) {
	// NO-OP
}

func (e *Explorer) exportFile(name string) (io.WriteCloser, error) {
	pathUTF16 := make([]uint16, _FilePathLength)
	copy(pathUTF16, windows.StringToUTF16(name))

	open := _OpenFileName{
		File:          &pathUTF16[0],
		MaxFile:       _FilePathLength,
		Filter:        buildFilter([]string{filepath.Ext(name)}),
		FileExtension: uint16(strings.Index(name, filepath.Ext(name))),
		Flags:         _FlagOverwritePrompt,
		StructSize:    _OpenFileStructLength,
	}

	if r, _, _ := _GetSaveFileName.Call(uintptr(unsafe.Pointer(&open))); r == 0 {
		return nil, ErrUserDecline
	}

	path := windows.UTF16ToString(pathUTF16)
	if len(path) == 0 {
		return nil, ErrUserDecline
	}

	return os.Create(path)
}

func (e *Explorer) importFile(extensions ...string) (io.ReadCloser, error) {
	pathUTF16 := make([]uint16, _FilePathLength)

	open := _OpenFileName{
		File:       &pathUTF16[0],
		MaxFile:    _FilePathLength,
		Filter:     buildFilter(extensions),
		Flags:      _FlagFileMustExist | _FlagForceShowHidden | _FlagDisableLinks,
		StructSize: _OpenFileStructLength,
	}

	if r, _, _ := _GetOpenFileName.Call(uintptr(unsafe.Pointer(&open))); r == 0 {
		return nil, ErrUserDecline
	}

	path := windows.UTF16ToString(pathUTF16)
	if len(path) == 0 {
		return nil, ErrUserDecline
	}

	return os.Open(path)
}

func (e *Explorer) importFiles(extensions ...string) ([]io.ReadCloser, error) {
	pathUTF16 := make([]uint16, _FilePathLength)

	open := _OpenFileName{
		File:       &pathUTF16[0],
		MaxFile:    _FilePathLength,
		Filter:     buildFilter(extensions),
		Flags:      _FlagFileMustExist | _FlagForceShowHidden | _FlagDisableLinks | _FlagAllowMultiSelect | _FlagExplorer,
		StructSize: _OpenFileStructLength,
	}

	if r, _, _ := _GetOpenFileName.Call(uintptr(unsafe.Pointer(&open))); r == 0 {
		return nil, ErrUserDecline
	}

	// Split the pathUTF16 by null characters
	paths := make([]string, 0)
	currentPath := make([]uint16, 0)
	for _, char := range pathUTF16 {
		if char == 0 {
			if len(currentPath) > 0 {
				paths = append(paths, windows.UTF16ToString(currentPath))
				currentPath = currentPath[:0]
			}
		} else {
			currentPath = append(currentPath, char)
		}
	}

	// The first element is the directory, append it to each filename
	dir := paths[0]
	filePaths := make([]string, len(paths)-1)
	for i, file := range paths[1:] {
		filePaths[i] = filepath.Join(dir, file)
	}

	if len(filePaths) == 0 {
		return nil, ErrUserDecline
	}

	files := make([]io.ReadCloser, len(filePaths))
	for i, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			for _, file := range files {
				if file != nil {
					file.Close()
				}
			}
			return nil, err
		}
		files[i] = file
	}

	return files, nil
}

func buildFilter(extensions []string) *uint16 {
	if len(extensions) <= 0 {
		return nil
	}

	for k, v := range extensions {
		// Extension must have `*` wildcard, so `.jpg` must be `*.jpg`.
		if !strings.HasPrefix(v, "*") {
			extensions[k] = "*" + v
		}
	}
	e := strings.ToUpper(strings.Join(extensions, ";"))

	// That is a "string-pair", Windows have a Title and the Filter, for instance it could be:
	// Images\0*.JPG;*.PNG\0\0
	// Where `\0` means NULL
	f := windows.StringToUTF16(e + " " + e) // Use the filter as title so it appear `*.JPG;*.PNG` for the user.
	f[len(e)] = 0                           // Replace the " " (space) with NULL.
	f = append(f, uint16(0))                // Adding another NULL, because we need two.
	return &f[0]
}
