package loader

import (
	"os"
	"path"

	"github.com/mirzakhany/chapar/internal/domain"
)

func GetRequestsDir() (string, error) {
	dir, err := CreateConfigDir()
	if err != nil {
		return "", err
	}

	requestsDir := path.Join(dir, requestsDir)
	if _, err := os.Stat(requestsDir); os.IsNotExist(err) {
		if err := os.Mkdir(requestsDir, 0755); err != nil {
			return "", err
		}
	}

	return requestsDir, nil
}

func DeleteRequest(env *domain.Request) error {
	return os.Remove(env.FilePath)
}

func LoadRequest(filePath string) (*domain.Request, error) {
	req, err := LoadFromYaml[domain.Request](filePath)
	if err != nil {
		return nil, err
	}

	setRequestDefaultValues(req)

	req.FilePath = filePath
	return req, nil
}

func LoadRequests() ([]*domain.Request, error) {
	dir, err := GetRequestsDir()
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]*domain.Request, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := path.Join(dir, file.Name())
		req, err := LoadRequest(filePath)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}

	return out, nil
}

func setRequestDefaultValues(r *domain.Request) {
	if r.MetaData.Type == "" {
		r.MetaData.Type = domain.KindRequest
	}

	if r.Spec.HTTP.Method == "" {
		r.Spec.HTTP.Method = "GET"
	}

	if r.Spec.HTTP.URL == "" {
		r.Spec.HTTP.URL = "https://example.com"
	}

	if r.Spec.HTTP.Request.Auth == (domain.Auth{}) {
		r.Spec.HTTP.Request.Auth = domain.Auth{
			Type: "None",
		}
	}

	if r.Spec.HTTP.Request.PostRequest == (domain.PostRequest{}) {
		r.Spec.HTTP.Request.PostRequest = domain.PostRequest{
			Type: "None",
		}
	}

	if r.Spec.HTTP.Request.PreRequest == (domain.PreRequest{}) {
		r.Spec.HTTP.Request.PreRequest = domain.PreRequest{
			Type: "None",
		}
	}
}

func UpdateRequest(req *domain.Request) error {
	if req.FilePath == "" {
		dir, err := GetRequestsDir()
		if err != nil {
			return err
		}
		// this is a new request
		fileName, err := getNewFileName(dir, req.MetaData.Name)
		if err != nil {
			return err
		}

		req.FilePath = fileName
	}

	if err := SaveToYaml(req.FilePath, req); err != nil {
		return err
	}

	// rename the file to the new name
	if req.MetaData.Name != path.Base(req.FilePath) {
		newFilePath := path.Join(path.Dir(req.FilePath), req.MetaData.Name+".yaml")
		if err := os.Rename(req.FilePath, newFilePath); err != nil {
			return err
		}
		req.FilePath = newFilePath
	}
	return nil
}

func GetNewFilePath(name, collectionName string) (string, error) {
	var dir string
	var err error

	if collectionName != "" {
		dir, err = GetCollectionsDir()
		if err != nil {
			return "", err
		}
		// if collection dir not found, create it
		dir = path.Join(dir, collectionName)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.Mkdir(dir, 0755); err != nil {
				return "", err
			}
		}

	} else {
		dir, err = GetRequestsDir()
		if err != nil {
			return "", err
		}
	}

	fileName, err := getNewFileName(dir, name)
	if err != nil {
		return "", err
	}

	return fileName, nil
}
