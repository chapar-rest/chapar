package domain

import (
	"strconv"
	"strings"
)

// PythonExecutorRepo is the Docker Hub repository of the Python script runner.
const PythonExecutorRepo = "chapar/python-executor"

// PythonExecutorImage is the runner this build speaks the scripting API of.
const PythonExecutorImage = PythonExecutorRepo + ":0.3.1"

// UpgradeExecutorImage moves a config still on an older runner to
// PythonExecutorImage and reports whether it changed anything. Another image,
// or a newer version of this one, is the user's choice and is left alone.
// "latest" (the default before images were pinned) and no tag are replaced
// too: they follow whatever runner is released next, which may speak another
// scripting API than this build.
func (s *ScriptingConfig) UpgradeExecutorImage() bool {
	repo, tag, _ := strings.Cut(s.DockerImage, ":")
	if repo != PythonExecutorRepo && repo != "docker.io/"+PythonExecutorRepo {
		return false
	}
	if tag != "" && tag != "latest" {
		have, ok := parseVersion(tag)
		if !ok {
			return false
		}
		want, _ := parseVersion(strings.TrimPrefix(PythonExecutorImage, PythonExecutorRepo+":"))
		if !versionLess(have, want) {
			return false
		}
	}
	s.DockerImage = PythonExecutorImage
	return true
}

// parseVersion reads "X.Y.Z" (an optional leading v is fine).
func parseVersion(s string) ([3]int, bool) {
	var v [3]int
	parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

func versionLess(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}
