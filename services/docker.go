package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const BusyboxImage = "busybox:1.36.1-glibc"

// DockerClient wraps Docker SDK operations
type DockerClient struct {
	cli *client.Client
	ctx context.Context
}

// NewDockerClient creates a new Docker client
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerClient{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

// Close closes the Docker client
func (dc *DockerClient) Close() error {
	if dc.cli != nil {
		return dc.cli.Close()
	}
	return nil
}

// GetImageUser inspects a Docker image to determine its default user
func (dc *DockerClient) GetImageUser(imageName string) (string, error) {
	imageInspect, err := dc.cli.ImageInspect(dc.ctx, imageName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect image %s: %w", imageName, err)
	}

	return imageInspect.Config.User, nil
}

// PullImage pulls a Docker image
func (dc *DockerClient) PullImage(imageName string) error {
	reader, err := dc.cli.ImagePull(dc.ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			slog.Debug("Failed to close image pull reader", "error", closeErr)
		}
	}()

	// Must consume the reader completely for the pull operation to finish
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to complete image pull for %s: %w", imageName, err)
	}

	return nil
}

// RunVolumeChowningContainer creates and runs a helper container for permission fixing
func (dc *DockerClient) RunVolumeChowningContainer(containerName, command string, mounts []mount.Mount) error {
	// First, ensure busybox image is available
	err := dc.PullImage(BusyboxImage)
	if err != nil {
		return fmt.Errorf("failed to pull busybox image: %w", err)
	}
	// Create helper container
	resp, err := dc.cli.ContainerCreate(dc.ctx, &container.Config{
		Image: BusyboxImage,
		Cmd:   []string{"sh", "-c", command},
	}, &container.HostConfig{
		Mounts:     mounts,
		AutoRemove: true,
	}, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create helper container: %w", err)
	}

	// Start helper container
	err = dc.cli.ContainerStart(dc.ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start helper container: %w", err)
	}

	// Wait for completion
	statusCh, errCh := dc.cli.ContainerWait(dc.ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error waiting for helper container: %w", err)
		}
	case status := <-statusCh:
		slog.Debug("Helper container completed",
			"container_name", containerName,
			"exit_code", status.StatusCode)

		if status.StatusCode != 0 {
			return fmt.Errorf("helper container exited with status %d", status.StatusCode)
		}
	}

	return nil
}

// RunContainerWithOutput runs a container and returns its stdout output
func (dc *DockerClient) RunContainerWithOutput(containerName, imageName, command string) (string, error) {
	// Create container
	resp, err := dc.cli.ContainerCreate(dc.ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"sh", "-c", command},
	}, &container.HostConfig{
		// No AutoRemove so we can get logs after completion
	}, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	err = dc.cli.ContainerStart(dc.ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for completion
	statusCh, errCh := dc.cli.ContainerWait(dc.ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		// Get logs after container has finished
		logs, logErr := dc.cli.ContainerLogs(dc.ctx, resp.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		})

		var output string
		var stderrOutput string
		if logErr == nil {
			defer func() {
				if closeErr := logs.Close(); closeErr != nil {
					slog.Debug("Failed to close container logs reader", "error", closeErr)
				}
			}()

			// Demultiplex Docker logs to remove headers
			var stdout, stderr bytes.Buffer
			_, readErr := stdcopy.StdCopy(&stdout, &stderr, logs)
			if readErr == nil {
				output = strings.TrimSpace(stdout.String())
				stderrOutput = strings.TrimSpace(stderr.String())
			}
		}

		// Clean up container
		removeErr := dc.cli.ContainerRemove(dc.ctx, resp.ID, container.RemoveOptions{})
		if removeErr != nil {
			slog.Debug("Failed to remove container", "error", removeErr)
		}

		if status.StatusCode != 0 {
			return "", fmt.Errorf("container exited with status %d, stderr: %s", status.StatusCode, stderrOutput)
		}

		return output, nil
	}

	return "", fmt.Errorf("unexpected end of container wait")
}
