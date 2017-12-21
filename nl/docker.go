package nl

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// With a heavy heart and deep internal sadness, we support Docker
func runningDockerContainers() (containerIDs []string, err error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		containerIDs = append(containerIDs, container.ID)
	}
	return containerIDs, nil
}

func dockerNetworkNamespace(containerID string) (string, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}
	defer cli.Close()
	r, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return "", nil
	}

	if r.State.Pid == 0 {
		// Container has exited
		return "", fmt.Errorf("Container has exited %v", containerID)
	}
	return fmt.Sprintf("/proc/%v/ns/net", r.State.Pid), nil
}

// Retrieve all namespaces from all running docker containers
func dockerNetworkNamespaces(containerIDs []string) (namespaces []string) {
	for _, containerID := range containerIDs {
		cid, err := dockerNetworkNamespace(containerID)
		if err == nil && len(cid) > 0 {
			namespaces = append(namespaces, cid)
		}
	}
	return
}
