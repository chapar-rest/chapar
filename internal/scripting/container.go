package scripting

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/chapar-rest/chapar/internal/logger"
)

// DockerClient wraps the Docker SDK client
type DockerClient struct {
	client *client.Client
}

// NewDockerClient creates a new Docker client
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %v", err)
	}

	return &DockerClient{client: cli}, nil
}

// Close closes the Docker client
func (dc *DockerClient) Close() error {
	return dc.client.Close()
}

func (dc *DockerClient) isContainerRunning(containerName string) (bool, error) {
	ctx := context.Background()

	// Create filter for container name
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", fmt.Sprintf("^%s$", containerName))

	containers, err := dc.client.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check running containers: %v", err)
	}

	// Check if any container matches and is running
	for _, cn := range containers {
		for _, name := range cn.Names {
			// Docker API returns names with leading slash
			cleanName := strings.TrimPrefix(name, "/")
			if cleanName == containerName {
				return cn.State == "running", nil
			}
		}
	}

	return false, nil
}

func (dc *DockerClient) isContainerExists(containerName string) (bool, error) {
	ctx := context.Background()

	// Create filter for container name (include stopped containers)
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", fmt.Sprintf("^%s$", containerName))

	containers, err := dc.client.ContainerList(ctx, container.ListOptions{
		All:     true, // Include stopped containers
		Filters: filterArgs,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check existing containers: %v", err)
	}

	// Check if any container matches
	for _, cn := range containers {
		for _, name := range cn.Names {
			// Docker API returns names with leading slash
			cleanName := strings.TrimPrefix(name, "/")
			if cleanName == containerName {
				return true, nil
			}
		}
	}

	return false, nil
}

func (dc *DockerClient) isImageExists(imageName string) (bool, error) {
	ctx := context.Background()
	images, err := dc.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to check existing images: %v", err)
	}

	// Check if image exists in the list
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				return true, nil
			}
		}
	}

	return false, nil
}

// getRemoteImageDigest gets the digest of the remote image without pulling it
func (dc *DockerClient) getRemoteImageDigest(imageName string) (string, error) {
	ctx := context.Background()

	// Use DistributionInspect to get remote image information without pulling
	distributionInspect, err := dc.client.DistributionInspect(ctx, imageName, "")
	if err != nil {
		return "", fmt.Errorf("failed to inspect remote image: %v", err)
	}

	return distributionInspect.Descriptor.Digest.String(), nil
}

// isImageUpToDate checks if the local image exists and is up to date with the remote
func (dc *DockerClient) isImageUpToDate(imageName string) (exists bool, upToDate bool, err error) {
	ctx := context.Background()

	// Get local images
	images, err := dc.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return false, false, fmt.Errorf("failed to check existing images: %v", err)
	}

	// Find local image
	var localImage *image.Summary
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				localImage = &img
				break
			}
		}
		if localImage != nil {
			break
		}
	}

	// If image doesn't exist locally, return false for both
	if localImage == nil {
		return false, false, nil
	}

	// Get remote image manifest to compare digests
	remoteDigest, err := dc.getRemoteImageDigest(imageName)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get remote image digest for %s: %v", imageName, err))
		// If we can't check remote, assume local image is valid to avoid unnecessary pulls
		return true, true, nil
	}

	// Compare digests - if they match, local image is up to date
	localDigest := localImage.ID
	if strings.HasPrefix(localDigest, "sha256:") {
		localDigest = strings.TrimPrefix(localDigest, "sha256:")
	}
	if strings.HasPrefix(remoteDigest, "sha256:") {
		remoteDigest = strings.TrimPrefix(remoteDigest, "sha256:")
	}

	isUpToDate := localDigest == remoteDigest
	if !isUpToDate {
		logger.Info(fmt.Sprintf("Image %s exists locally but is outdated (local: %s, remote: %s)",
			imageName, localDigest[:12], remoteDigest[:12]))
	} else {
		logger.Info(fmt.Sprintf("Image %s is up to date (digest: %s)", imageName, localDigest[:12]))
	}

	return true, isUpToDate, nil
}

func (dc *DockerClient) pullImage(imageName string) error {
	ctx := context.Background()

	reader, err := dc.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %v", err)
	}
	defer reader.Close()

	// Read the pull output (optional, for logging)
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %v", err)
	}

	return nil
}

func (dc *DockerClient) runContainer(imageName, containerName string, ports, envs []string) error {
	ctx := context.Background()

	// Parse port mappings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	for _, port := range ports {
		// Expect format like "8080:8080" or "8080:8080/tcp"
		parts := strings.Split(port, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid port format: %s", port)
		}

		hostPort := parts[0]
		containerPortStr := parts[1]

		// Handle protocol specification
		containerPort, err := nat.NewPort("tcp", containerPortStr)
		if err != nil {
			return fmt.Errorf("invalid container port: %s", containerPortStr)
		}

		exposedPorts[containerPort] = struct{}{}
		portBindings[containerPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			},
		}
	}

	// Create container configuration
	config := &container.Config{
		Image:        imageName,
		Env:          envs,
		ExposedPorts: exposedPorts,
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	// Create the container
	resp, err := dc.client.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Start the container
	if err := dc.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	return nil
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

func (dc *DockerClient) forceRemoveContainer(containerName string) error {
	ctx := context.Background()
	logger.Info(fmt.Sprintf("Force removing container %s", containerName))

	// Get container ID by name
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", fmt.Sprintf("^%s$", containerName))

	containers, err := dc.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	var containerID string
	for _, cn := range containers {
		for _, name := range cn.Names {
			cleanName := strings.TrimPrefix(name, "/")
			if cleanName == containerName {
				containerID = cn.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}

	if containerID == "" {
		logger.Info(fmt.Sprintf("Container %s not found", containerName))
		return nil // Container doesn't exist, consider it removed
	}

	// Remove the container with force option
	err = dc.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true,
	})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to force remove container %s: %v", containerName, err))
		return fmt.Errorf("failed to force remove container %s: %v", containerName, err)
	}

	logger.Info(fmt.Sprintf("Force removed container %s", containerName))
	return nil
}

func (dc *DockerClient) removeImage(imageName string) error {
	ctx := context.Background()
	// Remove the image
	_, err := dc.client.ImageRemove(ctx, imageName, image.RemoveOptions{
		Force: true, // Force removal even if the image is being used by stopped containers
	})
	if err != nil {
		return fmt.Errorf("failed to remove image %s: %v", imageName, err)
	}

	logger.Info(fmt.Sprintf("Removed image %s", imageName))
	return nil
}

// Helper functions that maintain the original API for backward compatibility
func isContainerRunning(containerName string) (bool, error) {
	dc, err := NewDockerClient()
	if err != nil {
		return false, err
	}
	defer dc.Close()

	return dc.isContainerRunning(containerName)
}

func isContainerExists(containerName string) (bool, error) {
	dc, err := NewDockerClient()
	if err != nil {
		return false, err
	}
	defer dc.Close()

	return dc.isContainerExists(containerName)
}

func isImageUpToDate(imageName string) (exists bool, upToDate bool, err error) {
	dc, err := NewDockerClient()
	if err != nil {
		return false, false, err
	}
	defer dc.Close()

	return dc.isImageUpToDate(imageName)
}

func isImageExists(imageName string) (bool, error) {
	dc, err := NewDockerClient()
	if err != nil {
		return false, err
	}
	defer dc.Close()

	return dc.isImageExists(imageName)
}

func removeImage(imageName string) error {
	dc, err := NewDockerClient()
	if err != nil {
		return err
	}
	defer dc.Close()

	return dc.removeImage(imageName)
}

func pullImage(imageName string) error {
	dc, err := NewDockerClient()
	if err != nil {
		return err
	}
	defer dc.Close()

	return dc.pullImage(imageName)
}

func runContainer(imageName, containerName string, ports, envs []string) error {
	dc, err := NewDockerClient()
	if err != nil {
		return err
	}
	defer dc.Close()

	return dc.runContainer(imageName, containerName, ports, envs)
}

func forceRemoveContainer(containerName string) error {
	dc, err := NewDockerClient()
	if err != nil {
		return err
	}
	defer dc.Close()

	return dc.forceRemoveContainer(containerName)
}
