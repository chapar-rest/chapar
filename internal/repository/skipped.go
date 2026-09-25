package repository

import (
	"errors"
	"fmt"
	"strings"
)

// SkippedFile is a data file a load left out because it could not be read,
// most often because its YAML is malformed.
type SkippedFile struct {
	Path string
	Err  error
}

// SkippedFilesError is returned by a load that read most of its files but had
// to leave some out. The entities that did load are returned alongside it, so
// callers can keep them and only report the skipped files.
type SkippedFilesError struct {
	Files []SkippedFile
}

func (e *SkippedFilesError) Error() string {
	parts := make([]string, len(e.Files))
	for i, f := range e.Files {
		parts[i] = fmt.Sprintf("%s: %v", f.Path, f.Err)
	}
	return "could not read " + strings.Join(parts, "; ")
}

// SkippedFiles returns the files err reports as skipped, and whether err is
// nothing more than skipped files.
func SkippedFiles(err error) ([]SkippedFile, bool) {
	var skipped *SkippedFilesError
	if !errors.As(err, &skipped) {
		return nil, false
	}
	return skipped.Files, true
}

type skippedFiles []SkippedFile

// absorb collects the files err reports as skipped. It returns false when err
// is some other failure that the caller must return itself.
func (s *skippedFiles) absorb(err error) bool {
	if err == nil {
		return true
	}
	files, ok := SkippedFiles(err)
	*s = append(*s, files...)
	return ok
}

func (s skippedFiles) err() error {
	if len(s) == 0 {
		return nil
	}
	return &SkippedFilesError{Files: s}
}
