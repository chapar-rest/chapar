package loader

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v2"
)

const (
	configDir = "chapar"

	envDir      = "envs"
	requestsDir = "requests"
)

func GetConfigDir() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}

	return path.Join(dir, configDir), nil
}

func userConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("AppData")
		if dir == "" {
			return "", errors.New("%AppData% is not defined")
		}

	case "plan9":
		dir = os.Getenv("home")
		if dir == "" {
			return "", errors.New("$home is not defined")
		}
		dir += "/lib"

	default: // Unix
		dir = os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
			}
			dir += "/.config"
		}
	}

	return dir, nil
}

func CreateConfigDir() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return "", err
		}
	}

	return dir, nil
}

func getNewFileName(dir, name string) (string, error) {
	fileName := path.Join(dir, name+".yaml")

	// if its already exists, add a number to the end of the name
	if _, err := os.Stat(fileName); err != nil {
		if os.IsNotExist(err) {
			return fileName, nil
		}

		i := 0
		for {
			fileName = path.Join(dir, name, fmt.Sprintf("%s-%d.yaml", name, i))
			if _, err := os.Stat(fileName); err != nil {
				if os.IsNotExist(err) {
					return fileName, nil
				}
			}
			i++
		}
	}

	return "", errors.New("file already exists")
}

func LoadFromYaml[T any](filename string) (*T, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	env := new(T)
	if err := yaml.Unmarshal(data, env); err != nil {
		return nil, err
	}
	return env, nil
}

func SaveToYaml[T any](filename string, data *T) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, out, 0644); err != nil {
		return err
	}
	return nil
}

// AddSuffixBeforeExt to add a suffix before the file extension
func AddSuffixBeforeExt(filePath, suffix string) string {
	dir, file := filepath.Split(filePath)
	extension := filepath.Ext(file)
	baseName := file[:len(file)-len(extension)]
	newBaseName := baseName + suffix + extension
	return filepath.Join(dir, newBaseName)
}
