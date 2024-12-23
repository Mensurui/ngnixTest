package main_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type inMemoryLogger struct {
	data []string
}

func (im *inMemoryLogger) Printf(format string, args ...interface{}) {
	im.data = append(im.data, fmt.Sprintf(format, args...))
}

type nginxContainer struct {
	testcontainers.Container
	URI string
}

func setupNginx(ctx context.Context, networkName string) (*nginxContainer, error) {
	dl := inMemoryLogger{}
	prints := []string{""}
	req := testcontainers.ContainerRequest{
		Image: "nginx:latest",
		LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
			{
				PreCreates: []testcontainers.ContainerRequestHook{
					func(ctx context.Context, req testcontainers.ContainerRequest) error {
						prints = append(prints, "pre-create hook 1")
						return nil
					},
					func(ctx context.Context, req testcontainers.ContainerRequest) error {
						prints = append(prints, "pre-create hook 2")
						return nil
					},
				},
			},
			testcontainers.DefaultLoggingHook(&dl),
		},
		ExposedPorts: []string{"80/tcp"},
		Networks:     []string{networkName},
		WaitingFor:   wait.ForHTTP("/"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "80")
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())
	return &nginxContainer{Container: container, URI: uri}, nil
}

func TestIntegrationNginxLatestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Create a custom network
	networkName := "foo-network"
	networkRequest := testcontainers.NetworkRequest{
		Name:   networkName,
		Driver: "bridge",
	}

	network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: networkRequest,
	})
	require.NoError(t, err)
	defer network.Remove(ctx) // Clean up the network

	// Set up the nginx container
	nginxC, err := setupNginx(ctx, networkName)
	require.NoError(t, err)
	defer nginxC.Terminate(ctx) // Clean up the container

	// Test the nginx container by making an HTTP request
	resp, err := http.Get(nginxC.URI)
	require.NoError(t, err)
	defer resp.Body.Close() // Ensure the response body is closed

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
