package loader

import (
	"os"
	"path"

	"github.com/mirzakhany/chapar/internal/domain"
)

func GetEnvDir() (string, error) {
	dir, err := CreateConfigDir()
	if err != nil {
		return "", err
	}

	envDir := path.Join(dir, envDir)
	if _, err := os.Stat(envDir); os.IsNotExist(err) {
		if err := os.Mkdir(envDir, 0755); err != nil {
			return "", err
		}
	}

	return envDir, nil
}

func DeleteEnvironment(env *domain.Environment) error {
	return os.Remove(env.FilePath)
}

func ReadEnvironmentsData() ([]*domain.Environment, error) {
	dir, err := GetEnvDir()
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]*domain.Environment, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := path.Join(dir, file.Name())

		env, err := LoadFromYaml[domain.Environment](filePath)
		if err != nil {
			return nil, err
		}
		env.FilePath = filePath
		out = append(out, env)
	}

	return out, nil
}

func UpdateEnvironment(env *domain.Environment) error {
	if env.FilePath == "" {
		// this is a new environment
		dir, err := GetEnvDir()
		if err != nil {
			return err
		}

		fileName, err := getNewFileName(dir, env.MetaData.Name)
		if err != nil {
			return err
		}

		env.FilePath = fileName
	}

	if err := SaveToYaml(env.FilePath, env); err != nil {
		return err
	}

	// rename the file to the new name
	if env.MetaData.Name != path.Base(env.FilePath) {
		newFilePath := path.Join(path.Dir(env.FilePath), env.MetaData.Name+".yaml")
		if err := os.Rename(env.FilePath, newFilePath); err != nil {
			return err
		}
		env.FilePath = newFilePath
	}

	return nil
}
