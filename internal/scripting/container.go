package scripting

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/logger"
)

func isContainerRunning(containerName string) (bool, error) {
	cmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=^%s$", containerName))
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check running containers: %v", err)
	}

	// If output has content, container is running
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func pullImage(imageName string) error {
	cmd := exec.Command("docker", "pull", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull image: %v, output: %s", err, string(output))
	}
	return nil
}

func runContainer(imageName, containerName string, ports, envs []string) error {
	args := []string{"run", "-d", "--name", containerName}

	// Add port mappings
	for _, port := range ports {
		args = append(args, "-p", port)
	}

	// Add environment variables
	for _, env := range envs {
		args = append(args, "-e", env)
	}

	args = append(args, imageName)

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to run container: %v, output: %s", err, string(output)))
		return fmt.Errorf("failed to run container: %v, output: %s", err, string(output))
	}

	return waitForPort("localhost", ports[0], 10*time.Second)
}

func waitForPort(host string, port string, timeout time.Duration) error {
	logger.Info(fmt.Sprintf("Waiting for executor port %s", port))
	start := time.Now()
	for time.Since(start) < timeout {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 1*time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	logger.Error(fmt.Sprintf("Timeout waiting for port %s:%s", host, port))
	return fmt.Errorf("timeout waiting for port %s:%s", host, port)
}

// Force remove container
func forceRemoveContainer(containerName string) error {
	logger.Info(fmt.Sprintf("Force removing container %s", containerName))
	cmd := exec.Command("docker", "rm", "-f", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to force remove container %s: %v, output: %s", containerName, err, string(output)))
		return fmt.Errorf("failed to force remove container %s: %v, output: %s", containerName, err, string(output))
	}

	logger.Info(fmt.Sprintf("Force removed container %s", containerName))
	return nil
}
