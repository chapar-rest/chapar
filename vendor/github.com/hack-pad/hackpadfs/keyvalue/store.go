package keyvalue

import "context"

// Store holds arbitrary file data at the given 'path' location. Can be wrapped as a file system with keyvalue.NewFS().
type Store interface {
	// Get retrieves a file record for the given 'path'.
	// Returns an error if the path was not found or could not be retrieved.
	// If the path was not found, the error must satisfy errors.Is(err, hackpadfs.ErrNotExist).
	Get(ctx context.Context, path string) (FileRecord, error)
	// Set assigns 'src' to the given 'path'. Returns an error if the data could not be set.
	Set(ctx context.Context, path string, src FileRecord) error
}
