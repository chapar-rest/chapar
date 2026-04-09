package repository

// GitStorage is a StorageBackend that stores files inside a local Git repository
// and can commit changes via the Commit method.
//
// Read operations never commit. Write operations (WriteFile, Rename, Remove,
// RemoveAll) stage changes; call Commit to persist them as a Git commit.
// If Commit is never called, auto-commit per write is the recommended fallback.
//
// All StorageBackend methods currently return ErrNotImplemented.
// To implement: add a go-git *git.Repository field and configure via NewGitStorage.
type GitStorage struct {
	repoPath string
	// repo *git.Repository
}

// NewGitStorage returns a GitStorage rooted at the given repository path.
func NewGitStorage(repoPath string) *GitStorage {
	return &GitStorage{repoPath: repoPath}
}

// Commit creates a Git commit with the given message for all staged changes.
// This lifecycle method is specific to GitStorage and is not part of StorageBackend.
// Callers that care about Git commits type-assert: storage.(*GitStorage).Commit(msg).
func (g *GitStorage) Commit(_ string) error { return ErrNotImplemented }

func (g *GitStorage) MkdirAll(_ string) error                          { return ErrNotImplemented }
func (g *GitStorage) ReadFile(_ string) ([]byte, error)                { return nil, ErrNotImplemented }
func (g *GitStorage) WriteFile(_ string, _ []byte) error               { return ErrNotImplemented }
func (g *GitStorage) Rename(_, _ string) error                         { return ErrNotImplemented }
func (g *GitStorage) Remove(_ string) error                            { return ErrNotImplemented }
func (g *GitStorage) RemoveAll(_ string) error                         { return ErrNotImplemented }
func (g *GitStorage) Stat(_ string) (bool, error)                      { return false, ErrNotImplemented }
func (g *GitStorage) ListEntries(_ string) ([]StorageEntry, error)     { return nil, ErrNotImplemented }
func (g *GitStorage) Glob(_ string) ([]string, error)                  { return nil, ErrNotImplemented }
