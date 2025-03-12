package scripting

import (
	"fmt"
	"sync"
)

// DefaultPluginManager provides a standard implementation of the PluginManager interface
type DefaultPluginManager struct {
	plugins map[string]ScriptPlugin
	mu      sync.RWMutex
}

// NewPluginManager creates a new DefaultPluginManager
func NewPluginManager() *DefaultPluginManager {
	return &DefaultPluginManager{
		plugins: make(map[string]ScriptPlugin),
	}
}

// RegisterPlugin registers a new plugin with the manager
func (pm *DefaultPluginManager) RegisterPlugin(name string, plugin ScriptPlugin, config map[string]interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if a plugin with this name already exists
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin with name '%s' already registered", name)
	}

	// Initialize the plugin
	if err := plugin.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize plugin '%s': %w", name, err)
	}

	// Store the plugin
	pm.plugins[name] = plugin
	return nil
}

// GetPlugin retrieves a plugin by name
func (pm *DefaultPluginManager) GetPlugin(name string) (ScriptPlugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins returns information about all registered plugins
func (pm *DefaultPluginManager) ListPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var infos []PluginInfo
	for _, plugin := range pm.plugins {
		infos = append(infos, plugin.GetInfo())
	}
	return infos
}

// ShutdownAll gracefully shuts down all plugins
func (pm *DefaultPluginManager) ShutdownAll() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var lastErr error
	for name, plugin := range pm.plugins {
		if err := plugin.Shutdown(); err != nil {
			lastErr = fmt.Errorf("failed to shutdown plugin '%s': %w", name, err)
		}
	}
	return lastErr
}
