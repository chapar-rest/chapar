package cookies

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/chapar-rest/chapar/internal/domain"
)

// Codec transforms jar files on their way to and from disk. It lets the files
// be encrypted later with a key kept in the OS secret store.
type Codec interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

type plainCodec struct{}

func (plainCodec) Encode(b []byte) ([]byte, error) { return b, nil }
func (plainCodec) Decode(b []byte) ([]byte, error) { return b, nil }

const (
	// StateDir holds machine-local workspace state that is kept out of git.
	StateDir = ".state"
	noEnvID  = "_none"
)

// Dir returns the cookie directory inside a workspace directory.
func Dir(workspaceDir string) string {
	return filepath.Join(workspaceDir, StateDir, "cookies")
}

// File returns the jar file of an environment inside a workspace directory.
func File(workspaceDir, envID string) string {
	if envID == "" {
		envID = noEnvID
	}
	return filepath.Join(Dir(workspaceDir), envID+".json")
}

// Store loads and saves one jar per environment of the active workspace.
type Store struct {
	workspaceDir func() (string, error)
	codec        Codec

	mu   sync.Mutex
	jars map[string]*Jar // keyed by jar file path
}

// NewStore creates a store for the workspace directory workspaceDir returns.
// It is a func because the active workspace can be switched or renamed.
func NewStore(workspaceDir func() (string, error), codec Codec) *Store {
	if codec == nil {
		codec = plainCodec{}
	}
	return &Store{workspaceDir: workspaceDir, codec: codec, jars: map[string]*Jar{}}
}

// For returns the jar of an environment, loading it from disk the first time.
// An empty envID is the jar used when no environment is active.
func (s *Store) For(envID string) (*Jar, error) {
	file, err := s.file(envID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jars[file]; ok {
		return j, nil
	}

	var list []*domain.Cookie
	b, err := os.ReadFile(file)
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return nil, err
	default:
		if b, err = s.codec.Decode(b); err != nil {
			return nil, fmt.Errorf("decode cookie jar %s: %w", file, err)
		}
		if err := json.Unmarshal(b, &list); err != nil {
			return nil, fmt.Errorf("parse cookie jar %s: %w", file, err)
		}
	}

	j := NewJar(list)
	s.jars[file] = j
	return j, nil
}

// Save writes the jar of an environment if it changed since the last save.
func (s *Store) Save(envID string) error {
	j, err := s.For(envID)
	if err != nil {
		return err
	}
	list, changed := j.snapshot()
	if !changed {
		return nil
	}
	if err := s.write(envID, list); err != nil {
		j.markDirty()
		return err
	}
	return nil
}

func (s *Store) write(envID string, list []*domain.Cookie) error {
	wsDir, err := s.workspaceDir()
	if err != nil {
		return err
	}
	if err := ensureStateDir(wsDir); err != nil {
		return err
	}

	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	if b, err = s.codec.Encode(b); err != nil {
		return err
	}

	file := File(wsDir, envID)
	tmp, err := os.CreateTemp(filepath.Dir(file), ".cookies-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}

// Delete drops the jar of an environment from memory and disk.
func (s *Store) Delete(envID string) error {
	file, err := s.file(envID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.jars, file)
	s.mu.Unlock()
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) file(envID string) (string, error) {
	wsDir, err := s.workspaceDir()
	if err != nil {
		return "", err
	}
	return File(wsDir, envID), nil
}

// ensureStateDir creates the state directory with a .gitignore that keeps all
// of it out of version control. An existing .gitignore is left alone so users
// can opt in to tracking.
func ensureStateDir(workspaceDir string) error {
	if err := os.MkdirAll(Dir(workspaceDir), 0o700); err != nil {
		return err
	}
	ignore := filepath.Join(workspaceDir, StateDir, ".gitignore")
	if _, err := os.Stat(ignore); err == nil || !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(ignore, []byte("*\n"), 0o644)
}
