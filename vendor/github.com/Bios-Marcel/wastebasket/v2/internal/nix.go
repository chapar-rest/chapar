//go:build freebsd || openbsd || netbsd || linux

package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"net/url"
	"os"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

// The mounts could change anytime, but probably won't all too often. So we
// read them each time Mount() is called, but remember the count, so we can
// at least prepare a buffer of that size. This must only be accessed atomicly.
var lastMountCount int32

// Mounts reads /proc/mounts and returns all mounts found, excluding
// everything on the devices sysfs, rootfs, cgroup and /dev/ paths.
func Mounts() ([]string, error) {
	handle, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("error opening /proc/mounts: %w", err)
	}

	defer handle.Close()

	scanner := bufio.NewScanner(handle)
	// I just assume that entries won't be much longer than 200, since my
	// own /proc/mounts was max 157 characters. Worst case, we'll probably
	// expand to 400. Either way, this should handle most cases efficiently.
	buffer := make([]byte, 0, 200)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			if firstSpace := bytes.IndexByte(data, ' '); firstSpace >= 0 {
				// Some device won't contain a trash either way or might be
				// dangerous to interact with.
				device := string(data[:firstSpace])
				switch device {
				case "rootfs", "sysfs", "cgroup", "cgroup2":
					// Skip line
					return i + 1, nil, nil
				}

				if nextSpace := bytes.IndexByte(data[firstSpace+1:], ' '); nextSpace >= 0 {
					path := data[firstSpace+1 : firstSpace+1+nextSpace]
					// Devices don't usually contain files and /sys should
					// be off-limits anyway.
					if bytes.HasPrefix(path, []byte("/dev/")) || bytes.HasPrefix(path, []byte("/sys/")) {
						return i + 1, nil, nil
					}

					return i + 1, path, nil
				}
			}

			return i + 1, nil, nil
		}
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), data, nil
		}
		// Request more data.
		return 0, nil, nil
	})
	scanner.Buffer(buffer, math.MaxInt)
	mounts := make([]string, 0, atomic.LoadInt32(&lastMountCount))

	for scanner.Scan() {
		mounts = append(mounts, scanner.Text())
	}

	atomic.StoreInt32(&lastMountCount, int32(len(mounts)))

	return mounts, nil
}

// RemoveAllIfExists tries to remove the given path. The path can either be a
// directory or a file. This function catches certain errors, so you don't have
// to.
func RemoveAllIfExists(path string) error {
RETRY:
	if err := os.RemoveAll(path); err != nil {
		pathErr, ok := err.(*os.PathError)
		if ok {
			switch pathErr.Err {
			case unix.EINTR:
				// A non-error basically which tells you to try again.
				// Use goto in order to prevent growing stack.
				goto RETRY
			case unix.EACCES:
				// Missing permissions, so we shouldn't clear it.
				return nil
			case unix.ENOENT:
				// Does not exist.
				return nil
			case unix.ENOTDIR:
				// Not a directory or file. This happens on WSL when
				// attempting to delete in /mnt/wslg/versions.txt, which is
				// weird considering that os.Remove (used internally) can
				// delete files.
				return nil
			case unix.EROFS:
				// Occurs if the filesystem is read-only.
				return nil
			}
		}
		return fmt.Errorf("error removing file: %w", err)
	}

	return nil
}

// EscapeUrl escapes the path according to the FreeDesktop Trash specification.
// Which basically just refers to "RFC 2396, section 2".
func EscapeUrl(path string) string {
	u := &url.URL{Path: path}
	return u.EscapedPath()
}

// FileExists omits the parts to make this usable cross-platform and
// therefore saves a minimal amount of CPU cycles and some allocations.
func FileExists(path string) (bool, error) {
	var (
		stat unix.Stat_t
		err  error
	)

RETRY:
	for {
		err = unix.Stat(path, &stat)
		switch err {
		case nil:
			// No issue, exists
			return true, nil
		case unix.EINTR:
			// A non-error basically which tells you to try again.
			continue RETRY
		case unix.ENOENT:
			// Doesn't exist
			return false, nil
		default:
			// Unexpected error
			return false, err
		}
	}
}
