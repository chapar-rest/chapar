package domain

import "testing"

func TestUpgradeExecutorImage(t *testing.T) {
	for _, tc := range []struct {
		image, want string
		changed     bool
	}{
		{"chapar/python-executor:0.3.0", PythonExecutorImage, true},
		{"chapar/python-executor:0.2.9", PythonExecutorImage, true},
		{"chapar/python-executor:latest", PythonExecutorImage, true},
		{"chapar/python-executor", PythonExecutorImage, true},
		{"docker.io/chapar/python-executor:v0.1.0", PythonExecutorImage, true},
		{PythonExecutorImage, PythonExecutorImage, false},
		{"chapar/python-executor:9.0.0", "chapar/python-executor:9.0.0", false},
		{"chapar/python-executor:dev", "chapar/python-executor:dev", false},
		{"me/my-runner:0.1.0", "me/my-runner:0.1.0", false},
		{"", "", false},
	} {
		s := ScriptingConfig{DockerImage: tc.image}
		if changed := s.UpgradeExecutorImage(); changed != tc.changed || s.DockerImage != tc.want {
			t.Errorf("%q: got %q changed=%v, want %q changed=%v", tc.image, s.DockerImage, changed, tc.want, tc.changed)
		}
	}
}
