package repository

// GCSStorage is a StorageBackend backed by Google Cloud Storage.
// All methods currently return ErrNotImplemented.
//
// To implement: add a *storage.Client field and configure it via NewGCSStorage.
type GCSStorage struct {
	bucket string
	prefix string
	// client *storage.Client
}

// NewGCSStorage returns a GCSStorage targeting the given bucket and object prefix.
func NewGCSStorage(bucket, prefix string) *GCSStorage {
	return &GCSStorage{bucket: bucket, prefix: prefix}
}

func (g *GCSStorage) MkdirAll(_ string) error                          { return ErrNotImplemented }
func (g *GCSStorage) ReadFile(_ string) ([]byte, error)                { return nil, ErrNotImplemented }
func (g *GCSStorage) WriteFile(_ string, _ []byte) error               { return ErrNotImplemented }
func (g *GCSStorage) Rename(_, _ string) error                         { return ErrNotImplemented }
func (g *GCSStorage) Remove(_ string) error                            { return ErrNotImplemented }
func (g *GCSStorage) RemoveAll(_ string) error                         { return ErrNotImplemented }
func (g *GCSStorage) Stat(_ string) (bool, error)                      { return false, ErrNotImplemented }
func (g *GCSStorage) ListEntries(_ string) ([]StorageEntry, error)     { return nil, ErrNotImplemented }
func (g *GCSStorage) Glob(_ string) ([]string, error)                  { return nil, ErrNotImplemented }
