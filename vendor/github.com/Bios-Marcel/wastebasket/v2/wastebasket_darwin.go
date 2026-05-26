package wastebasket

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Trash moves a file or folder including its content into the systems trashbin.
func Trash(paths ...string) error {
	for _, path := range paths {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			return err
		}

		//Passing a relative path will lead to the Finder not being able to find the file at all.
		path, pathToAbsPathError := filepath.Abs(path)
		if pathToAbsPathError != nil {
			return pathToAbsPathError
		}

		path = strings.ReplaceAll(path, `"`, `\"`)
		osascriptCommand := fmt.Sprintf(`tell app "Finder" to delete POSIX file "%s"`, path)
		err = exec.Command("osascript", "-e", osascriptCommand).Run()
		if err != nil {
			return err
		}
	}

	return nil
}

// Empty clears the platforms trashbin. It uses the `Finder` app to empty the trashbin.
func Empty() error {
	return exec.Command("osascript", "-e", `tell app "Finder" to empty`).Run()
}

// Query is not supported.
func Query(options QueryOptions) (*QueryResult, error) {
	return nil, ErrPlatformNotSupported
}
