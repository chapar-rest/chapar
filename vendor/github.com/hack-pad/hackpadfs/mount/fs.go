// Package mount contains an implementation of hackpadfs.MountFS.
package mount

import (
	"io"
	"path"
	"strings"
	"sync"

	"github.com/hack-pad/hackpadfs"
)

var (
	_ interface {
		hackpadfs.FS
		hackpadfs.MountFS
		hackpadfs.RenameFS
	} = &FS{}
)

// FS is mesh of several file systems mounted at different paths.
// Mount a file system with AddMount().
//
// For ease of use, call the standard operations via hackpadfs.OpenFile(fs, ...), hackpadfs.Mkdir(fs, ...), etc.
type FS struct {
	rootFS  hackpadfs.FS
	mounts  sync.Map // map[string]hackpadfs.FS
	mountMu sync.Mutex
}

// NewFS returns a new FS.
func NewFS(rootFS hackpadfs.FS) (*FS, error) {
	return &FS{
		rootFS: rootFS,
	}, nil
}

// AddMount mounts 'mount' at 'path'. The mount point must already exist as a directory.
func (fs *FS) AddMount(path string, mount hackpadfs.FS) error {
	err := fs.addMount(path, mount)
	if err != nil {
		return &hackpadfs.PathError{Op: "mount", Path: path, Err: err}
	}
	return nil
}

func (fs *FS) addMount(p string, mountFS hackpadfs.FS) error {
	if !hackpadfs.ValidPath(p) || p == "." {
		return hackpadfs.ErrInvalid
	}
	_, loaded := fs.mounts.Load(p)
	if loaded {
		// cannot mount at same point as existing mount
		return hackpadfs.ErrExist
	}
	fs.mountMu.Lock()
	defer fs.mountMu.Unlock()

	dir, base := path.Split(p)
	parentFS, subPath := fs.Mount(dir) // get this mount point's parent mount, verify dir exists
	f, err := parentFS.Open(path.Join(subPath, base))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return hackpadfs.ErrNotDir
	}
	// TODO Handle data race when directory is removed or becomes a file between the Stat and the mount.

	_, loaded = fs.mounts.LoadOrStore(p, mountFS)
	if loaded {
		// cannot mount at same point as existing mount
		return hackpadfs.ErrExist
	}
	return nil
}

// Mount implements hackpadfs.MountFS
func (fs *FS) Mount(path string) (mount hackpadfs.FS, subPath string) {
	mount, mountPath, subPath := fs.mountPoint(path)
	if mountPath == "." {
		return mount, path
	}
	return mount, subPath
}

func (fs *FS) mountPoint(path string) (_ hackpadfs.FS, mountPoint, subPath string) {
	var resultPath string
	resultFS := fs.rootFS
	fs.mounts.Range(func(key, value interface{}) bool {
		mountPath, mountFS := key.(string), value.(hackpadfs.FS)
		switch {
		case strings.HasPrefix(path, mountPath+"/"):
			if len(mountPath) > len(resultPath) {
				resultPath, resultFS = mountPath, mountFS
			}
			return true
		case mountPath == path:
			// exact match
			resultPath, resultFS = mountPath, mountFS
			return false
		default:
			return true
		}
	})
	subPath = path
	subPath = strings.TrimPrefix(subPath, resultPath)
	subPath = strings.TrimPrefix(subPath, "/")

	if resultPath == "" {
		resultPath = "."
	}
	if subPath == "" {
		subPath = "."
	}
	return resultFS, resultPath, subPath
}

// Open implements hackpadfs.FS
func (fs *FS) Open(name string) (hackpadfs.File, error) {
	mountFS, subPath := fs.Mount(name)
	return mountFS.Open(subPath)
}

// Point represents a mount point, including any relevant metadata
type Point struct {
	Path string
}

// MountPoints returns a slice of mount points every mounted file system.
func (fs *FS) MountPoints() []Point {
	var points []Point
	fs.mounts.Range(func(key, _ interface{}) bool {
		path := key.(string)
		points = append(points, Point{path})
		return true
	})
	return points
}

// Rename implements hackpadfs.RenameFS
func (fs *FS) Rename(oldname, newname string) error {
	oldMount, oldPoint, oldSubPath := fs.mountPoint(oldname)
	newMount, newPoint, newSubPath := fs.mountPoint(newname)
	oldInfo, err := hackpadfs.Stat(oldMount, oldSubPath)
	if err != nil {
		return &hackpadfs.LinkError{Op: "rename", Old: oldname, New: newname, Err: err}
	}
	if oldname == newname {
		if !oldInfo.IsDir() {
			return nil
		}
		return &hackpadfs.LinkError{Op: "rename", Old: oldname, New: newname, Err: hackpadfs.ErrExist}
	}

	if oldPoint == newPoint {
		return hackpadfs.Rename(oldMount, oldSubPath, newSubPath)
	}
	if oldInfo.IsDir() {
		// TODO support renaming directories
		return &hackpadfs.LinkError{Op: "rename", Old: oldname, New: newname, Err: hackpadfs.ErrNotImplemented}
	}

	oldFile, err := oldMount.Open(oldSubPath)
	if err != nil {
		return err
	}
	defer func() { _ = oldFile.Close() }()
	newFile, err := hackpadfs.OpenFile(newMount, newSubPath, hackpadfs.FlagWriteOnly|hackpadfs.FlagCreate|hackpadfs.FlagTruncate, oldInfo.Mode())
	if err != nil {
		return err
	}
	newFileWriter, ok := newFile.(io.Writer)
	if !ok {
		return &hackpadfs.LinkError{Op: "rename", Old: oldname, New: newname, Err: hackpadfs.ErrPermission}
	}
	defer func() { _ = newFile.Close() }()
	_, err = io.Copy(newFileWriter, oldFile)
	if err != nil {
		_ = hackpadfs.Remove(newMount, newSubPath)
		return err
	}
	return hackpadfs.Remove(oldMount, oldSubPath)
}
