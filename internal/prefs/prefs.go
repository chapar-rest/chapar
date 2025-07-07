package prefs

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

var (
	instance *Manager
	once     sync.Once
)

type (
	GlobalConfigChangeListener func(old, updated domain.GlobalConfig)
	AppStateChangeListener     func(old, updated domain.AppState)
)

// Manager handles loading, saving, and migrating preferences and config
type Manager struct {
	globalConfig *domain.GlobalConfig
	appState     *domain.AppState

	globalConfigChangeListeners []GlobalConfigChangeListener
	appStateChangeListeners     []AppStateChangeListener
}

// GetInstance returns the singleton instance of the Manager
func GetInstance() *Manager {
	once.Do(func() {
		instance = &Manager{}
		err := instance.Load()
		if err != nil {
			// Initialize with defaults if loading fails
			instance.globalConfig = domain.GetDefaultGlobalConfig()
			instance.appState = domain.GetDefaultAppState()
		}
	})
	return instance
}

func AddGlobalConfigChangeListener(listener GlobalConfigChangeListener) {
	m := GetInstance()
	if m == nil {
		return // Instance not initialized yet
	}
	m.AddGlobalConfigChangeListener(listener)
}

func (m *Manager) AddGlobalConfigChangeListener(listener GlobalConfigChangeListener) {
	m.globalConfigChangeListeners = append(m.globalConfigChangeListeners, listener)
}

func (m *Manager) AddAppStateChangeListener(listener AppStateChangeListener) {
	m.appStateChangeListeners = append(m.appStateChangeListeners, listener)
}

func (m *Manager) notifyGlobalConfigChange(old, updated domain.GlobalConfig) {
	for _, listener := range m.globalConfigChangeListeners {
		listener(old, updated)
	}
}

func (m *Manager) notifyAppStateChange(old, updated domain.AppState) {
	for _, listener := range m.appStateChangeListeners {
		listener(old, updated)
	}
}

// Load loads both globalConfig and appState from disk,
// performing migration from old format if needed
func (m *Manager) Load() error {
	// Try to load the new format first
	configLoaded, configErr := m.loadGlobalConfig()
	if configErr != nil {
		return configErr
	}

	stateLoaded, stateErr := m.loadAppState()
	if stateErr != nil {
		return stateErr
	}

	// If both loaded successfully, we're done
	if configLoaded && stateLoaded {
		return nil
	}

	// If either failed to load, try migrating from old format
	if err := m.migrateFromOldFormat(); err != nil {
		return fmt.Errorf("failed to load or migrate preferences: %w", err)
	}

	// After migration, save in new format
	if err := m.Save(); err != nil {
		return fmt.Errorf("failed to save after migration: %w", err)
	}

	return nil
}

// Save persists both globalConfig and appState to disk
func (m *Manager) Save() error {
	if err := m.saveGlobalConfig(); err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	if err := m.saveAppState(); err != nil {
		return fmt.Errorf("failed to save app state: %w", err)
	}

	return nil
}

func GetGlobalConfig() domain.GlobalConfig {
	return GetInstance().GetGlobalConfig()
}

func GetWorkspacePath() string {
	return GetInstance().GetGlobalConfig().Spec.Data.WorkspacePath
}

// GetGlobalConfig returns a copy of the global config
func (m *Manager) GetGlobalConfig() domain.GlobalConfig {
	if m.globalConfig == nil {
		m.globalConfig = domain.GetDefaultGlobalConfig()
	}

	return *m.globalConfig
}

func UpdateGlobalConfig(config domain.GlobalConfig) error {
	return GetInstance().UpdateGlobalConfig(config)
}

// UpdateGlobalConfig updates the global config
func (m *Manager) UpdateGlobalConfig(config domain.GlobalConfig) error {
	old := *m.globalConfig
	m.globalConfig = &config
	if err := m.saveGlobalConfig(); err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	m.notifyGlobalConfigChange(old, config)
	return nil
}

func GetAppState() domain.AppState {
	return GetInstance().GetAppState()
}

// GetAppState returns a copy of the app state
func (m *Manager) GetAppState() domain.AppState {
	return *m.appState
}

func UpdateAppState(config domain.AppState) error {
	return GetInstance().UpdateAppState(config)
}

// UpdateAppState updates the app state
func (m *Manager) UpdateAppState(state domain.AppState) error {
	old := *m.appState
	m.appState = &state
	if err := m.saveAppState(); err != nil {
		return fmt.Errorf("failed to save app state: %w", err)
	}

	m.notifyAppStateChange(old, state)
	return nil
}

// Private helper methods
func (m *Manager) loadGlobalConfig() (bool, error) {
	path, err := getFilePath("global-config.yaml")
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "js" {
		// TODO: Implement browser storage loading
		// For WASM you'd use browser localStorage here
		return false, fmt.Errorf("browser storage not implemented")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File doesn't exist yet, not an error
		}
		return false, err
	}

	config := &domain.GlobalConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return false, err
	}

	m.globalConfig = config
	return true, nil
}

func (m *Manager) loadAppState() (bool, error) {
	path, err := getFilePath("app-state.yaml")
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "js" {
		// TODO: Implement browser storage loading
		return false, fmt.Errorf("browser storage not implemented")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File doesn't exist yet, not an error
		}
		return false, err
	}

	state := &domain.AppState{}
	if err := yaml.Unmarshal(data, state); err != nil {
		return false, err
	}

	m.appState = state
	return true, nil
}

func (m *Manager) saveGlobalConfig() error {
	if m.globalConfig == nil {
		return fmt.Errorf("global config is nil")
	}

	path, err := getFilePath("global-config.yaml")
	if err != nil {
		return err
	}

	if runtime.GOOS == "js" {
		// TODO: Implement browser storage saving
		return fmt.Errorf("browser storage not implemented")
	}

	data, err := yaml.Marshal(m.globalConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (m *Manager) saveAppState() error {
	if m.appState == nil {
		return fmt.Errorf("app state is nil")
	}

	path, err := getFilePath("app-state.yaml")
	if err != nil {
		return err
	}

	if runtime.GOOS == "js" {
		// TODO: Implement browser storage saving
		return fmt.Errorf("browser storage not implemented")
	}

	data, err := yaml.Marshal(m.appState)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// migrateFromOldFormat loads data from the old format and converts it to the new format
func (m *Manager) migrateFromOldFormat() error {
	legacyDataDir, err := domain.LegacyConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get legacy config dir: %w", err)
	}

	// Create storage client to access old data format
	storageClient, err := repository.NewFilesystem(legacyDataDir, domain.AppStateSpec{})
	if err != nil {
		return err
	}

	// Try to load old config
	oldConfig, err := storageClient.GetConfig()

	// Create default new formats
	m.globalConfig = domain.GetDefaultGlobalConfig()
	m.appState = domain.GetDefaultAppState()

	if err == nil && oldConfig != nil {
		// Migrate data from old config to new structures

		// Move workspace data to AppState
		if oldConfig.Spec.ActiveWorkspace != nil {
			m.appState.Spec.ActiveWorkspace = oldConfig.Spec.ActiveWorkspace
		}

		// Move environment data to AppState
		if oldConfig.Spec.SelectedEnvironment != nil {
			m.appState.Spec.SelectedEnvironment = oldConfig.Spec.SelectedEnvironment
		}
	}

	// Try to load old preferences
	oldPrefs, err := storageClient.ReadPreferences()
	if err == nil && oldPrefs != nil {
		// Migrate data from old preferences to new structures

		// If we already have dark mode from config, don't overwrite
		m.appState.Spec.DarkMode = oldPrefs.Spec.DarkMode

		// Move selected environment if not already set
		if m.appState.Spec.SelectedEnvironment == nil || m.appState.Spec.SelectedEnvironment.ID == "" {
			m.appState.Spec.SelectedEnvironment = &domain.SelectedEnvironment{
				ID:   oldPrefs.Spec.SelectedEnvironment.ID,
				Name: oldPrefs.Spec.SelectedEnvironment.Name,
			}
		}
	}

	return nil
}
