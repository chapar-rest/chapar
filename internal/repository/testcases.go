package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
)

// LoadTestCases reads the test cases of the active workspace.
func (f *FilesystemV2) LoadTestCases() ([]*domain.TestCase, error) {
	path, err := f.EntityPath(domain.KindTestCase)
	if err != nil {
		return nil, err
	}

	cases, err := loadList[domain.TestCase](path, func(n *domain.TestCase) {
		f.entities.Set(n.ID(), n.GetName())
	})
	if files, ok := SkippedFiles(err); ok {
		for i := range files {
			if isLegacyTestCase(files[i].Path) {
				files[i].Err = ErrLegacyTestCase
			}
		}
	}
	return cases, err
}

// ErrLegacyTestCase explains a test case file an early preview of test cases
// wrote, whose format the app no longer reads.
var ErrLegacyTestCase = errors.New("this test case was made by an early preview of test cases, in a format that is no longer read; create it again in Tests")

// isLegacyTestCase reports whether path holds a test case in the preview
// format: steps with a request.collection, an assert block instead of a
// list, or before/after hooks.
func isLegacyTestCase(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var probe struct {
		Kind string `yaml:"kind"`
		Spec struct {
			Steps []map[string]any `yaml:"steps"`
		} `yaml:"spec"`
	}
	if yaml.Unmarshal(b, &probe) != nil || probe.Kind != domain.KindTestCase {
		return false
	}
	for _, step := range probe.Spec.Steps {
		if req, ok := step["request"].(map[any]any); ok {
			if _, ok := req["collection"]; ok {
				return true
			}
		}
		if _, ok := step["assert"].(map[any]any); ok {
			return true
		}
		for _, k := range []string{"before", "after", "runNaked"} {
			if _, ok := step[k]; ok {
				return true
			}
		}
	}
	return false
}

// CreateTestCase writes a new test case. One without an ID, or with the ID
// of an entity already loaded (an import of a file that is already here),
// gets a new ID.
func (f *FilesystemV2) CreateTestCase(testCase *domain.TestCase) error {
	if _, taken := f.entities.Get(testCase.ID()); testCase.ID() == "" || taken {
		testCase.MetaData.ID = uuid.NewString()
	}
	testCase.ApiVersion = domain.ApiVersion
	testCase.Kind = domain.KindTestCase

	path, err := f.EntityPath(domain.KindTestCase)
	if err != nil {
		return err
	}
	if err := f.writeFile(path, testCase, false); err != nil {
		return err
	}
	f.entities.Set(testCase.ID(), testCase.GetName())
	return nil
}

// UpdateTestCase writes a test case, renaming its file when its name changed.
func (f *FilesystemV2) UpdateTestCase(testCase *domain.TestCase) error {
	oldName, ok := f.entities.Get(testCase.ID())
	if !ok {
		return fmt.Errorf("test case with ID %s not found", testCase.ID())
	}

	path, err := f.EntityPath(domain.KindTestCase)
	if err != nil {
		return err
	}

	if oldName != testCase.GetName() {
		anotherFileExists, err := doesFileNameExistWithDifferentID(filepath.Join(path, testCase.GetName()+".yaml"), testCase.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			testCase.SetName(f.ensureUniqueName(path, testCase.GetName(), ".yaml"))
		}

		if err := f.renameEntity(path, oldName+".yaml", testCase.GetName()+".yaml"); err != nil {
			return fmt.Errorf("cannot rename test case with ID %s: %v", testCase.ID(), err)
		}
		f.entities.Set(testCase.ID(), testCase.GetName())
	}

	return f.writeFile(path, testCase, true)
}

func (f *FilesystemV2) DeleteTestCase(testCase *domain.TestCase) error {
	path, err := f.EntityPath(domain.KindTestCase)
	if err != nil {
		return err
	}
	if err := f.deleteEntity(path, testCase); err != nil {
		return err
	}
	f.entities.Delete(testCase.ID())
	return nil
}
