package loader

import (
	"path"

	"github.com/mirzakhany/chapar/internal/domain"
)

func ReadPreferencesData() (*domain.Preferences, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	filePath := path.Join(dir, "preferences.yaml")
	return LoadFromYaml[domain.Preferences](filePath)
}

func UpdatePreferences(pref *domain.Preferences) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	filePath := path.Join(dir, "preferences.yaml")
	return SaveToYaml[domain.Preferences](filePath, pref)
}
