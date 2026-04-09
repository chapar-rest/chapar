package repository

// S3Storage is a StorageBackend backed by Amazon S3.
// All methods currently return ErrNotImplemented.
//
// To implement: add an *s3.Client field and configure it via NewS3Storage.
type S3Storage struct {
	bucket string
	prefix string
	// client *s3.Client
}

// NewS3Storage returns an S3Storage targeting the given bucket and key prefix.
func NewS3Storage(bucket, prefix string) *S3Storage {
	return &S3Storage{bucket: bucket, prefix: prefix}
}

func (s *S3Storage) MkdirAll(_ string) error                          { return ErrNotImplemented }
func (s *S3Storage) ReadFile(_ string) ([]byte, error)                { return nil, ErrNotImplemented }
func (s *S3Storage) WriteFile(_ string, _ []byte) error               { return ErrNotImplemented }
func (s *S3Storage) Rename(_, _ string) error                         { return ErrNotImplemented }
func (s *S3Storage) Remove(_ string) error                            { return ErrNotImplemented }
func (s *S3Storage) RemoveAll(_ string) error                         { return ErrNotImplemented }
func (s *S3Storage) Stat(_ string) (bool, error)                      { return false, ErrNotImplemented }
func (s *S3Storage) ListEntries(_ string) ([]StorageEntry, error)     { return nil, ErrNotImplemented }
func (s *S3Storage) Glob(_ string) ([]string, error)                  { return nil, ErrNotImplemented }
