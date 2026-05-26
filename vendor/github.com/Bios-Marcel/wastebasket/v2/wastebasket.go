package wastebasket

import (
	"errors"
	"time"
)

// TrashedFileInfo represents a file that has been deleted and now resides
// in the trashbin.
type TrashedFileInfo interface {
	// OriginalPath is the files path before it was deleted.
	OriginalPath() string
	// DeletionDate is the deletion date in the computers local timezone.
	DeletionDate() time.Time
	// Restore will attempt restoring the file to its previous location.
	Restore(force bool) error
	// Delete will permanently deleting the underlying file. Note that we do not
	// zero the respective bytes on the disk.
	Delete() error
	// UniqueIdentifier can be used to uniquely identify a file if the path and
	// deletion date are exactly the same. This CAN be useful file restoration.
	UniqueIdentifier() string
}

type QueryResult struct {
	// Matches are the query results, mapped from the query input (globa / path)
	// to the respective trashed files. Note that the same file can be trashed
	// multiple times and a glob can match multiple files. Therefore, expect
	// multiple entries in both scenarios.
	Matches map[string][]TrashedFileInfo
}

// QueryOptions allows to configure the Query-Call. Options Globs and Paths
// aren't allowed to both be set.
type QueryOptions struct {
	Glob bool
	// Search can be relative or absolute.
	Search []string
}

var (
	// ErrPlatformNotSupported indicates that the current platform does not
	// suport trashing files or the API isn't fully implemented.
	ErrPlatformNotSupported = errors.New("platform not supported")
	ErrAlreadyExists        = errors.New("couldn't restore file, already exists, apply force")
	ErrOnlyOneGlobAllowed   = errors.New("only one glob is allowed")
)

func (options QueryOptions) validate() error {
	if options.Glob && len(options.Search) > 1 {
		return ErrOnlyOneGlobAllowed
	}
	return nil
}
