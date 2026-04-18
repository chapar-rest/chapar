package state

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/safemap"
)

type TestCaseChangeListener func(testCase *domain.TestCase, source Source, action Action)

type TestCases struct {
	testCaseChangeListeners []TestCaseChangeListener
	testCases               *safemap.Map[*domain.TestCase]

	repository repository.RepositoryV2
}

func NewTestCases(repository repository.RepositoryV2) *TestCases {
	return &TestCases{
		repository: repository,
		testCases:  safemap.New[*domain.TestCase](),
	}
}

func (m *TestCases) AddTestCaseChangeListener(listener TestCaseChangeListener) {
	m.testCaseChangeListeners = append(m.testCaseChangeListeners, listener)
}

func (m *TestCases) notifyTestCaseChange(testCase *domain.TestCase, source Source, action Action) {
	for _, listener := range m.testCaseChangeListeners {
		listener(testCase, source, action)
	}
}

func (m *TestCases) AddTestCase(testCase *domain.TestCase, source Source) {
	m.testCases.Set(testCase.MetaData.ID, testCase)
	m.notifyTestCaseChange(testCase, source, ActionAdd)
}

func (m *TestCases) RemoveTestCase(testCase *domain.TestCase, source Source, stateOnly bool) error {
	if _, ok := m.testCases.Get(testCase.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.DeleteTestCase(testCase); err != nil {
			return err
		}
	}

	m.testCases.Delete(testCase.MetaData.ID)
	m.notifyTestCaseChange(testCase, source, ActionDelete)
	return nil
}

func (m *TestCases) GetTestCase(id string) *domain.TestCase {
	testCase, _ := m.testCases.Get(id)
	return testCase
}

func (m *TestCases) UpdateTestCase(testCase *domain.TestCase, source Source, stateOnly bool) error {
	if _, ok := m.testCases.Get(testCase.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.UpdateTestCase(testCase); err != nil {
			return err
		}
	}

	m.testCases.Set(testCase.MetaData.ID, testCase)
	m.notifyTestCaseChange(testCase, source, ActionUpdate)

	return nil
}

func (m *TestCases) GetPersistedTestCase(id string) (*domain.TestCase, error) {
	testCases, err := m.repository.LoadTestCases()
	if err != nil {
		return nil, err
	}

	for _, tc := range testCases {
		if tc.MetaData.ID == id {
			return tc, nil
		}
	}

	return nil, ErrNotFound
}

func (m *TestCases) ReloadTestCase(id string, source Source) {
	_, ok := m.testCases.Get(id)
	if !ok {
		return
	}

	testCase, err := m.GetPersistedTestCase(id)
	if err != nil {
		return
	}

	m.testCases.Set(id, testCase)
	m.notifyTestCaseChange(testCase, source, ActionUpdate)
}

func (m *TestCases) GetTestCases() []*domain.TestCase {
	return m.testCases.Values()
}

func (m *TestCases) LoadTestCases() ([]*domain.TestCase, error) {
	testCases, err := m.repository.LoadTestCases()
	if err != nil {
		return nil, err
	}

	for _, tc := range testCases {
		m.testCases.Set(tc.MetaData.ID, tc)
	}

	return testCases, nil
}

func (m *TestCases) SaveTestCase(testCase *domain.TestCase, source Source) error {
	if _, ok := m.testCases.Get(testCase.MetaData.ID); ok {
		return m.UpdateTestCase(testCase, source, false)
	}

	if err := m.repository.CreateTestCase(testCase); err != nil {
		return err
	}

	m.AddTestCase(testCase, source)
	return nil
}
