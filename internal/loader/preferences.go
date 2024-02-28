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

	pref, err := LoadFromYaml[domain.Preferences](filePath)
	if err != nil {
		return nil, err
	}

	return pref, nil
}

func UpdatePreferences(pref *domain.Preferences) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	filePath := path.Join(dir, "preferences.yaml")
	return SaveToYaml[domain.Preferences](filePath, pref)
}
