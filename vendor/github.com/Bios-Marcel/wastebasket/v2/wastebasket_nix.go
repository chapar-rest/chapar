//go:build freebsd || openbsd || netbsd || linux

package wastebasket

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Bios-Marcel/wastebasket/v2/internal"
	"github.com/Bios-Marcel/wastebasket/v2/wastebasket_nix"
	"github.com/gobwas/glob"
)

// RFC3339 is the same as time.RFC3339 but without timezones.
const RFC3339 string = "2006-01-02T15:04:05"

// cachedInformation makes sure we don't constantly check for the
// directory and which drive it is on again.
var cachedInformation = &cache{
	init: &sync.Once{},
}

type cache struct {
	// path is the path to the trash dir, for example
	//   /home/marcel/.local/share/Trash.
	path string
	// topdir is the closest mountpoint of `path`.
	topdir string
	// dataHome FIXME explain
	dataHome string

	init *sync.Once
	err  error
}

func getCache() (*cache, error) {
	cachedInformation.init.Do(func() {
		// On some big distros, such as Ubuntu for example, this variable isn't
		// set. Instead, we will fallback to what Ubuntu does for now.
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				cachedInformation.err = err
				return
			}

			cachedInformation.dataHome = filepath.Join(homeDir, ".local", "share")
		} else {
			cachedInformation.dataHome = dataHome
		}

		cachedInformation.path = filepath.Join(cachedInformation.dataHome, "Trash")

		mounts, err := internal.Mounts()
		if err != nil {
			cachedInformation.err = err
		} else {
			homeTopdir, err := topdir(mounts, cachedInformation.path)
			if err != nil {
				cachedInformation.err = err
			} else {
				cachedInformation.err = nil
				cachedInformation.topdir = homeTopdir
			}
		}
	})

	return cachedInformation, cachedInformation.err
}

func topdir(potentialTopdirs []string, path string) (string, error) {
	var matchingDir string
	for _, dir := range potentialTopdirs {
		// Technically mounts can be nested, so we can have more than one
		// match, but want the deepest possible match.
		if strings.HasPrefix(path, dir) && len(dir) > len(matchingDir) {
			matchingDir = dir
		}
	}

	return matchingDir, nil
}

func Trash(paths ...string) error {
	// RFC3339 defined in the time package contains the timezone offset, which
	// isn't defined by the spec and causes issues in some trash tools, such
	// as trash-cli.
	deletionDate := time.Now().Format(RFC3339)
	cache, err := getCache()
	if err != nil {
		return fmt.Errorf("error determining user trash directory: %w", err)
	}

	mounts, err := internal.Mounts()
	if err != nil {
		return fmt.Errorf("error retrieving mounts: %w", err)
	}

	for _, absPath := range paths {
		var err error
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			return fmt.Errorf("error retrieving absolute filepath: %w", err)
		}

		pathTopdir, err := topdir(mounts, absPath)
		if err != nil {
			return fmt.Errorf("error determining topdir: %w", err)
		}

		// We only support absolute filenames in the home trash. For
		// topdirs, we use relative paths. This allows us to move a
		// mount, while still keeping trash files recoverable.
		var pathForTrashInfo string

		var trashDir, filesDir, infoDir string
		// Deleting accross partitions / mounts
		if cache.topdir != pathTopdir {
			// While getTopDir won't return an empty string with its current
			// impl, this can change in the future, so beteter be safe than
			// sorry.
			if pathTopdir != "" {
				var uid string
				if currentUser, err := user.Current(); err != nil {
					return fmt.Errorf("error getting current user: %w", err)
				} else {
					uid = currentUser.Uid
				}

				trashDir = filepath.Join(pathTopdir, ".Trash")

				var useFallbackTopdirTrash bool
				if trashDirStat, err := os.Stat(trashDir); err != nil {
					if !os.IsNotExist(err) {
						return fmt.Errorf("error checking for trash directory: %w", err)
					}
					useFallbackTopdirTrash = true
				} else {
					if trashDirStat.Mode()&fs.ModeSticky != 0 {
						// If the topdir trash directory contains a trash for all
						// users, it needs to have the sticky bit set. This is only
						// required for .Trash though, not for .Trash-$uid.
						useFallbackTopdirTrash = true
					} else if trashDirStat.Mode()&os.ModeSymlink != 0 {
						// Symlinks must not be used as per spec.
						useFallbackTopdirTrash = true
					}
				}

				pathForTrashInfo, err = filepath.Rel(pathTopdir, absPath)
				if err != nil {
					return fmt.Errorf("error retrieving relative path: %w", err)
				}

				if !useFallbackTopdirTrash {
					filesDir = filepath.Join(trashDir, uid, "files")
					infoDir = filepath.Join(trashDir, uid, "info")
				} else {
					// If .Trash doesn't exist, we need to check for .Trash-$uid
					// and create it if it doesn't exist. The spec however
					// doesn't indicate that we should do the same with .Trash.
					trashDir = filepath.Join(pathTopdir, ".Trash-"+uid)
					filesDir = filepath.Join(trashDir, "files")
					infoDir = filepath.Join(trashDir, "info")
				}

			}
		}

		if trashDir == "" {
			// Fallback to home trash.
			trashDir = cache.path
			filesDir = filepath.Join(trashDir, "files")
			infoDir = filepath.Join(trashDir, "info")

			// Hometrash supports both relative and absolute paths.
			if trashParent := filepath.Dir(trashDir); strings.HasPrefix(absPath, trashParent) {
				relPath, err := filepath.Rel(trashParent, absPath)
				if err != nil {
					return fmt.Errorf("error retrieving relative path: %w", err)
				}
				pathForTrashInfo = relPath
			} else {
				pathForTrashInfo = absPath
			}
		}

		if err := os.MkdirAll(filesDir, 0o700); err != nil && !os.IsExist(err) {
			return fmt.Errorf("error creating directory '%s': %w", filesDir, err)
		}
		if err := os.MkdirAll(infoDir, 0o700); err != nil && !os.IsExist(err) {
			return fmt.Errorf("error creating directory '%s': %w", infoDir, err)
		}

		baseName := filepath.Base(absPath)
		trashedFilePath := filepath.Join(filesDir, baseName)
		trashedFileInfoPath := filepath.Join(infoDir, baseName) + ".trashinfo"

		// We need to check whether the trash already contains a file with this
		// name, since deleted files from different directories often have the
		// same name. An example would be .gitignore files, they always have
		// the same basename and therefore always the same trash path.
		// We simply count up in this case. Since we've got the info file, we
		// can map back to the original name later on.

		var infoFileHandle *os.File
		if exists, err := internal.FileExists(trashedFilePath); err != nil {
			return err
		} else if !exists {
			// We save ourselves the FileExists check, as we can combine it
			// with the opening of the file handle. This is a performance
			// optimisation.
			infoFileHandle, err = os.OpenFile(trashedFileInfoPath, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				if !os.IsExist(err) {
					return fmt.Errorf("error creating info file: %w", err)
				}
			}

			// While we close manually later, we want to prevent a leak.
			defer infoFileHandle.Close()
		}

		// If there isn't a valid info file handle yet, it means that one
		// of the two file names were already in use, requiring us to find
		// two unique filenames eiter way.
		if infoFileHandle == nil {
			extension := filepath.Ext(baseName)
			baseNameNoExtension := strings.TrimSuffix(baseName, extension)
			for i := uint64(1); i != 0; i = i + 1 {
				newBaseName := fmt.Sprintf("%s.%d%s", baseNameNoExtension, i, extension)

				// The names of both files must always be the same, putting
				// aside the .trashinfo extension.
				trashedFilePath = filepath.Join(filesDir, newBaseName)
				if exists, err := internal.FileExists(trashedFilePath); err != nil || exists {
					continue
				}
				infoFileHandle, err = os.OpenFile(filepath.Join(infoDir, newBaseName+".trashinfo"), os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0o600)
				if err != nil {
					if os.IsExist(err) {
						continue
					}
					return fmt.Errorf("error creating info file: %w", err)
				}

				defer infoFileHandle.Close()

				// We found a valid name, where neither the file itself, nor
				// the trashinfo file exist.
				break
			}
		}

		if err := os.Rename(absPath, trashedFilePath); err != nil {
			// We save ourselvse the exists check at the start of the loop, as
			// deleting non existing files probably does not happen that often.
			if os.IsNotExist(err) {
				// Since we already create the info file, we will have to manually delete it again.
				name := infoFileHandle.Name()
				infoFileHandle.Close()
				// We ignore the error here, it isn't super important
				os.Remove(name)
				continue
			}

			// All special treatment failed, return original os.Rename error
			return fmt.Errorf("error moving file to trash: %w", err)
		}

		if _, err = infoFileHandle.WriteString(fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", internal.EscapeUrl(pathForTrashInfo), deletionDate)); err != nil {
			return fmt.Errorf("error writing to info file: %w", err)
		}
	}

	return nil
}

func isPermissionDenied(err error) bool {
	if err == os.ErrPermission {
		return true
	}

	if pe, ok := err.(*os.PathError); ok {
		if errno, ok := pe.Err.(syscall.Errno); ok {
			// EACCES (permission denied) and EPERM (operation not permitted) are checked
			return errno == syscall.EACCES || errno == syscall.EPERM
		}
	}

	return false
}

func Empty() error {
	cache, err := getCache()
	if err != nil {
		return fmt.Errorf("error accessing cache: %w", err)
	}

	if err := internal.RemoveAllIfExists(cache.path); err != nil {
		return fmt.Errorf("error clearing home trash '%s': %w", cache.path, err)
	}

	mounts, err := internal.Mounts()
	if err != nil {
		return fmt.Errorf("error retrieving mounts: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("error getting current user: %w", err)
	}

	for _, mount := range mounts {
		path := filepath.Join(mount, ".Trash", currentUser.Uid)
		if err := internal.RemoveAllIfExists(path); err != nil && !isPermissionDenied(err) {
			return fmt.Errorf("error clearing mount trash '%s': %w", path, err)
		}

		path = filepath.Join(mount, fmt.Sprintf(".Trash-%s", currentUser.Uid))
		if err := internal.RemoveAllIfExists(path); err != nil && !isPermissionDenied(err) {
			return fmt.Errorf("error clearing mount trash '%s': %w", path, err)
		}
	}

	return nil
}

func Query(options QueryOptions) (*QueryResult, error) {
	if err := options.validate(); err != nil {
		return nil, fmt.Errorf("error validating options: %w", err)
	}

	cached, err := getCache()
	if err != nil {
		return nil, fmt.Errorf("error accessing cache: %w", err)
	}

	result := &QueryResult{
		Matches: make(map[string][]TrashedFileInfo),
	}

	var matcher func(string) (string, bool)
	if options.Glob {
		globString := options.Search[0]
		compiled, err := glob.Compile(globString)
		if err != nil {
			return nil, fmt.Errorf("error compiling glob: %w", err)
		}
		matcher = func(s string) (string, bool) {
			if compiled.Match(s) {
				return globString, true
			}
			return "", false
		}
	} else {
		trashParent := filepath.Dir(cached.path)
		matcher, err = relativePathMatcher(trashParent, options.Search)
		if err != nil {
			return nil, fmt.Errorf("error creating matcher: %w", err)
		}
	}

	if err := queryTrashDir(result, matcher, cached.dataHome, cached.path); err != nil {
		return nil, fmt.Errorf("error querying home trash: %w", err)
	}

	mounts, err := internal.Mounts()
	if err != nil {
		return nil, fmt.Errorf("error retrieving mounts: %w", err)
	}

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("error retrieving user data: %w", err)
	}

	for _, mount := range mounts {
		// Previously generated relative paths are for the home
		// trash, therefore we need to regenerate them for the topdir
		// trash, but reuse the slice for less shitty performance.
		matcher, err = relativePathMatcher(mount, options.Search)
		if err != nil {
			return nil, fmt.Errorf("error creating matcher: %w", err)
		}
		if err := queryTrashDir(result, matcher, mount, filepath.Join(mount, ".Trash", u.Uid)); err != nil {
			return nil, fmt.Errorf("error querying mount trash: %w", err)
		}

		if err := queryTrashDir(result, matcher, mount, filepath.Join(mount, fmt.Sprintf(".Trash-%s", u.Uid))); err != nil {
			return nil, fmt.Errorf("error querying mount trash: %w", err)
		}
	}

	return result, nil
}

func relativePathMatcher(base string, search []string) (func(string) (string, bool), error) {
	absPaths := make([]string, len(search))
	relPaths := make([]string, len(search))
	for index, path := range search {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		absPaths[index] = absPath

		relPaths[index], err = filepath.Rel(base, absPath)
		if err != nil {
			return nil, fmt.Errorf("error retrieving relative path: %w", err)
		}
	}

	return func(s string) (string, bool) {
		for i, path := range search {
			if absPaths[i] == s || relPaths[i] == s {
				return path, true
			}
		}
		return "", false
	}, nil
}

func queryTrashDir(result *QueryResult, matcher func(string) (string, bool), baseDir, trashDir string) error {
	infoDirectoryPath := filepath.Join(trashDir, "info")
	err := filepath.WalkDir(infoDirectoryPath, func(infoPath string, dirEntry fs.DirEntry, err error) error {
		if infoDirectoryPath == infoPath {
			// No home trash means no files
			if os.IsNotExist(err) || errors.Is(err, fs.ErrNotExist) {
				return fs.ErrNotExist
			}

			return nil
		}

		// Info dir shouldn't contain any dir, therefore we ignore this,
		// as it should also not cause any further issues.
		if dirEntry.IsDir() {
			return nil
		}

		bytes, err := os.ReadFile(infoPath)
		if err != nil {
			return fmt.Errorf("error reading .trashinfo file: %w", err)
		}

		// FIXME Is Fscanf more efficient?
		// FIXME Are there generally more efficient stdlib ways to do this or should I manually parse?
		var originalPath, deletionDateStr string
		_, err = fmt.Sscanf(string(bytes), "[Trash Info]\nPath=%s\nDeletionDate=%s\n", &originalPath, &deletionDateStr)
		if err != nil {
			return fmt.Errorf("error parsing .trashinfo file: %w", err)
		}

		deletionDate, err := time.Parse(RFC3339, deletionDateStr)
		if err != nil {
			return fmt.Errorf("error parsing deletion date: %w", err)
		}

		// If we saved a relative path, we need to join it together first, as
		// our workdirectory might not match the directory the file resided in.
		if !strings.HasPrefix(originalPath, "/") {
			originalPath = filepath.Join(baseDir, originalPath)
		}

		// Hometrash supports both absolute paths and relative paths.
		if input, matches := matcher(originalPath); matches {
			trashedFile := filepath.Join(trashDir, "files", strings.TrimSuffix(filepath.Base(infoPath), ".trashinfo"))
			trashInfo := wastebasket_nix.NewTrashedFileInfo(
				originalPath,
				deletionDate,
				infoPath,
				trashedFile,
				func(force bool) error {
					return restore(infoPath, trashedFile, originalPath, force)
				},
				func() error {
					if err := os.Remove(infoPath); err != nil {
						return fmt.Errorf("error removing .trashinfo at '%s': %w", infoPath, err)
					}

					if err := os.Remove(trashedFile); err != nil {
						return fmt.Errorf("error removing trashed file at '%s': %w", trashedFile, err)
					}

					return nil
				},
			)
			result.Matches[input] = append(result.Matches[input], trashInfo)
		}

		return nil
	})

	// If there is no hometrash, that is fine.
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}

// It's probably preferable not to have a public Restore(...) function, as you
// mostly will have to query first in order to delete anyways. Even then, a
// restore with multiple files versions to restore would complicate the API.
func restore(infoPath, trahedFilePath, originalPath string, force bool) error {
	if !force {
		info, err := os.Stat(originalPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error checking whether file exists: %w", err)
		}
		if info != nil {
			return ErrAlreadyExists
		}
	}
	if err := os.Rename(trahedFilePath, originalPath); err != nil {
		// FIXME Use root error type that is public API
		return fmt.Errorf("error restoring file '%s' to '%s'; .trashinfo path: '%s'", trahedFilePath, originalPath, infoPath)
	}

	if err := os.Remove(infoPath); err != nil {
		return fmt.Errorf("error removing .trashinfo at '%s'; the file has been successfully restored though: %w", infoPath, err)
	}

	return nil
}
