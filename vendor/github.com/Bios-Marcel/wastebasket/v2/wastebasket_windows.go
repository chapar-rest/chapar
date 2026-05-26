package wastebasket

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/Bios-Marcel/wastebasket/v2/wastebasket_windows"
	"github.com/gobwas/glob"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/unicode"
)

var (
	shell32DLL         = windows.NewLazyDLL("shell32.dll")
	shFileOperationW   = shell32DLL.NewProc("SHFileOperationW")
	shEmptyRecycleBinW = shell32DLL.NewProc("SHEmptyRecycleBinW")
)

/*
#define FO_DELETE         0x0003

#define FOF_SILENT                 0x0004
#define FOF_NOCONFIRMATION         0x0010
#define FOF_ALLOWUNDO              0x0040
#define FOF_NOCONFIRMMKDIR         0x0200
#define FOF_NOERRORUI              0x0400
#define FOF_NO_UI                  (FOF_SILENT | FOF_NOCONFIRMATION | FOF_NOERRORUI | FOF_NOCONFIRMMKDIR)

typedef struct _SHFILEOPSTRUCTW {
  HWND         hwnd;
  UINT         wFunc;
  PCZZWSTR     pFrom;
  PCZZWSTR     pTo;
  FILEOP_FLAGS fFlags;
  BOOL         fAnyOperationsAborted;
  LPVOID       hNameMappings;
  PCWSTR       lpszProgressTitle;
} SHFILEOPSTRUCTW, *LPSHFILEOPSTRUCTW;
*/

const (
	FO_DELETE = 0x0003

	FOF_SILENT         = 0x0004
	FOF_NOCONFIRMATION = 0x0010
	FOF_ALLOWUNDO      = 0x0040
	FOF_NOCONFIRMMKDIR = 0x0200
	FOF_NOERRORUI      = 0x0400
	FOF_NO_UI          = FOF_SILENT | FOF_NOCONFIRMATION | FOF_NOERRORUI | FOF_NOCONFIRMMKDIR
)

type SHFileOpStructW struct {
	// Irrelevant, as its for UI related things
	hwnd windows.HWND

	wFunc uint32
	pFrom *uint16
	// This isn't necessary as it is only needed for copy / move actions.
	pTo    *uint16
	fFlags uint16

	// FIXME Why isn't this relevant?
	fAnyOperationsAborted int32
	// Irrelevant, as its for move operations
	hNameMappings uintptr
	// Irrelevant, as its for UI related things
	lpszProgressTitle *uint16
}

// Trash moves a file or folder including its content into the systems trashbin.
func Trash(paths ...string) error {
	existingPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		// The API will return error code "2 - Operation completed successfully"
		// when attempting to delete a non-existent file.
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}

		existingPaths = append(existingPaths, path)
	}

	filesParameter, err := makeDoubleNullTerminatedLpstr(existingPaths...)
	if err != nil {
		return fmt.Errorf("error creating utf16ptr for passed path: %w", err)
	}

	ret, _, err := shFileOperationW.Call(uintptr(unsafe.Pointer(&SHFileOpStructW{
		hwnd:                  windows.HWND(0),
		wFunc:                 FO_DELETE,
		pFrom:                 filesParameter,
		pTo:                   nil,
		fFlags:                FOF_NO_UI | FOF_ALLOWUNDO,
		fAnyOperationsAborted: 0,
		hNameMappings:         0,
		lpszProgressTitle:     nil,
	})))

	if ret != 0 {
		return fmt.Errorf("windows error: %w", err)
	}

	return nil
}

func makeDoubleNullTerminatedLpstr(items ...string) (*uint16, error) {
	chars := []uint16{}
	for _, s := range items {
		converted, err := windows.UTF16FromString(s)
		if err != nil {
			return nil, fmt.Errorf("error converting string to utf16: %w", err)
		}
		chars = append(chars, converted...)
	}
	chars = append(chars, 0)
	return &chars[0], nil
}

const (
	SHERB_NOCONFIRMATION = 1
	SHERB_NOPROGRESSUI   = 2
	SHERB_NOSOUND        = 4
)

// Empty clears the platforms trashbin.
func Empty() error {
	flags := SHERB_NOCONFIRMATION | SHERB_NOPROGRESSUI | SHERB_NOSOUND

	ret, _, err := shEmptyRecycleBinW.Call(uintptr(unsafe.Pointer(nil)), uintptr(unsafe.Pointer(nil)), uintptr(flags))
	if ret != 0 {
		// Weird edge case, where windows reports that it couldnt load the DLL
		// if the trash bin is empty.
		if err.(windows.Errno) == 126 {
			return nil
		}

		return fmt.Errorf("windows error: %w", err)
	}

	return nil
}

// The info files have the following structure:
// 8 Byte header
// 8 Byte for file size
// 8 Byte for deletion date
// 4 Byte for path length
// N Byte for path

// https://stackoverflow.com/questions/6693^9004/windows-recycle-bin-information-file-binary-format

func Query(options QueryOptions) (*QueryResult, error) {
	if err := options.validate(); err != nil {
		return nil, fmt.Errorf("error validating options: %w", err)
	}

	user, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("error querying SID of windows user: %w", err)
	}

	// We map the paths per volume, assuming that each volume only contains
	// files from that volume. Additionally, we make all paths
	// absolute, defaulting to the volume of the current working directory.
	volumeMapping := make(map[string][][2]string)
	var globCompiled glob.Glob
	if options.Glob {
		globString := options.Search[0]
		globCompiled, err = glob.Compile(globString)
		if err != nil {
			return nil, fmt.Errorf("error compiling glob: %w", err)
		}

		// FIXME Figure out what exactly counts as a logical drive and whether
		// we need to potentially filter out network drives and such. Do network
		// drives even support trashing?
		volumes, err := windows.GetLogicalDriveStrings(0, nil)
		if err != nil {
			return nil, fmt.Errorf("error retrieving logical drive strings: %w", err)
		}

		a := make([]uint16, volumes)
		windows.GetLogicalDriveStrings(volumes, &a[0])
		s := string(utf16.Decode(a))
		for _, volume := range strings.Split(strings.TrimRight(s, "\x00"), "\x00") {
			volumeMapping[volume] = nil
		}
	} else {
		for _, path := range options.Search {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil, fmt.Errorf("error retrieving absolute filepath: %w", err)
			}

			volumeName := filepath.VolumeName(absPath)
			volumeMapping[volumeName] = append(volumeMapping[volumeName], [...]string{absPath, path})
		}
	}

	result := &QueryResult{
		Matches: make(map[string][]TrashedFileInfo),
	}
	pathDecoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	for volume, paths := range volumeMapping {
		rootTrash := fmt.Sprintf(`%s\$Recycle.Bin\%s`, volume, user.Uid)

		infoFiles, err := filepath.Glob(rootTrash + `\$I*`)
		if err != nil {
			return nil, fmt.Errorf("error looking up info files: %w", err)
		}

	INFO_LOOP:
		for _, infoFile := range infoFiles {
			bytes, err := os.ReadFile(infoFile)
			if err != nil {
				err := fmt.Errorf("error reading info file: %w", err)
				return nil, err
			}

			trashedFile := fmt.Sprintf("%s\\$R%s", rootTrash, strings.TrimPrefix(filepath.Base(infoFile), "$I"))
			// Windows seems to keep the metadata files on restoration
			// Until I've figured out why, i'll ignore these files.
			// If the error is non-nil, we will ignore it and continue. Since
			// the stat call to this file, is not directly important.
			if _, err := os.Stat(trashedFile); os.IsNotExist(err) {
				continue INFO_LOOP
			}

			// -2 to ignore nullbyte in the end
			var originalFilepath string
			if pathBytes, err := pathDecoder.Bytes(bytes[28 : len(bytes)-2]); err != nil {
				err := fmt.Errorf("error decoding path: %w", err)
				return nil, err
			} else {
				originalFilepath = string(pathBytes)
			}

			// Since globs and paths are mutually exclusive, at best one of those
			// loops will match and we don't need to run both.
			if globCompiled != nil {
				if globCompiled.Match(originalFilepath) {
					info := createTrashedFile(infoFile, trashedFile, originalFilepath, bytes)
					result.Matches[options.Search[0]] = append(result.Matches[options.Search[0]], info)
				}
				continue INFO_LOOP
			}
			for _, path := range paths {
				if path[0] == originalFilepath {
					info := createTrashedFile(infoFile, trashedFile, originalFilepath, bytes)
					result.Matches[path[1]] = append(result.Matches[path[1]], info)
					continue INFO_LOOP
				}
			}
		}
	}

	return result, nil
}

func createTrashedFile(infoFile, trashedFile, originalFilepath string, infoData []byte) *wastebasket_windows.TrashedFileInfo {
	// According to an SO article, the info file can contain
	// garbage bytes in the beginning, but seemingly, they seem
	// to be little endian notation BOM bytes. While I haven't
	// encountered these and can't confirm that they
	// exist, guarding against this shouldn't hurt.
	var byteOffset int
	for index, b := range infoData {
		// Start of header found
		if b == 0x02 {
			byteOffset = index
			break
		}
	}

	fileSize := binary.LittleEndian.Uint64(infoData[byteOffset+8 : byteOffset+16])
	deletionTime := syscall.Filetime{
		LowDateTime:  binary.LittleEndian.Uint32(infoData[byteOffset+16 : byteOffset+20]),
		HighDateTime: binary.LittleEndian.Uint32(infoData[byteOffset+20 : byteOffset+24]),
	}

	recoverFunc := createRecover(infoFile, trashedFile, originalFilepath)
	deleteFunc := createDelete(infoFile, trashedFile)
	return wastebasket_windows.NewTrashedFileInfo(
		fileSize,
		infoFile,
		originalFilepath,
		time.Unix(0, deletionTime.Nanoseconds()),
		recoverFunc,
		deleteFunc,
	)
}

func createDelete(infoFile, trashedFile string) func() error {
	return func() error {
		if err := os.Remove(infoFile); err != nil {
			return fmt.Errorf("error removing info file: %w", err)
		}

		if err := os.Remove(trashedFile); err != nil {
			return fmt.Errorf("error removing trashed file: %w", err)
		}

		return nil
	}
}

func createRecover(infoFile, trashedFile, originalFile string) func(force bool) error {
	return func(force bool) error {
		if !force {
			info, err := os.Stat(originalFile)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error checking whether file exists: %w", err)
			}
			if info != nil {
				return ErrAlreadyExists
			}
		}
		err := os.Rename(trashedFile, originalFile)
		if err != nil {
			return fmt.Errorf("error restoring file: %w", err)
		}

		if err := os.Remove(infoFile); err != nil {
			return fmt.Errorf("error removing info file: %w", err)
		}

		return nil
	}
}
