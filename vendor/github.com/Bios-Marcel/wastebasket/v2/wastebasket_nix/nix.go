package wastebasket_nix

import (
	"fmt"
	"hash/fnv"
	"time"
)

type TrashedFileInfo struct {
	originalPath string
	deletionDate time.Time

	infoPath, currentPath string
	restoreFunc           func(force bool) error
	deleteFunc            func() error
}

func NewTrashedFileInfo(
	originalPath string, deletionDate time.Time,
	infoPath, currentPath string,
	restoreFunc func(force bool) error,
	deleteFunc func() error,
) *TrashedFileInfo {
	return &TrashedFileInfo{
		originalPath: originalPath,
		deletionDate: deletionDate,
		infoPath:     infoPath,
		currentPath:  currentPath,
		restoreFunc:  restoreFunc,
		deleteFunc:   deleteFunc,
	}
}

func (t TrashedFileInfo) UniqueIdentifier() string {
	hash := fnv.New64()
	hash.Write([]byte(t.infoPath))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// OriginalPath is the files path before it was deleted.
func (t TrashedFileInfo) OriginalPath() string {
	return t.originalPath
}

// InfoPath is the path (inside the trashbin), where information about the
// trashed file is stored.
func (t TrashedFileInfo) InfoPath() string {
	return t.infoPath
}

// CurrentPath is the path (inside the trashbin), where the file currently
// resides.
func (t TrashedFileInfo) CurrentPath() string {
	return t.currentPath
}

// DeletionDate is the deletion date in the computers local timezone.
func (t TrashedFileInfo) DeletionDate() time.Time {
	return t.deletionDate
}

// Restore will attempt restoring the file to its previous location.
func (t TrashedFileInfo) Restore(force bool) error {
	return t.restoreFunc(force)
}

func (t TrashedFileInfo) Delete() error {
	return t.deleteFunc()
}
