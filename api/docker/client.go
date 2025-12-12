package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// DockerClient envuelve el cliente de Docker SDK
type DockerClient struct {
	cli *client.Client
}

// NewDockerClient crea una nueva instancia del cliente Docker
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	return &DockerClient{cli: cli}, nil
}

// ListContainers devuelve solo los contenedores de client_img
func (dc *DockerClient) ListContainers() ([]types.Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Filtrar solo contenedores de la imagen client_img
	filterArgs := filters.NewArgs()
	filterArgs.Add("ancestor", "client_img")

	return dc.cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true, // Incluir contenedores detenidos
		Filters: filterArgs,
	})
}

// GetContainer obtiene información detallada de un contenedor
func (dc *DockerClient) GetContainer(containerID string) (types.ContainerJSON, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return dc.cli.ContainerInspect(ctx, containerID)
}

// CreateContainerConfig representa la configuración para crear un contenedor
type CreateContainerConfig struct {
	Name         string
	Image        string
	NetworkName  string
	Binds        []string
	PortBindings map[string]string
	Cmd          []string
	Env          []string
}

// CreateContainer crea un nuevo contenedor con la configuración especificada
func (dc *DockerClient) CreateContainer(config CreateContainerConfig) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Construir port bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	for containerPort, hostPort := range config.PortBindings {
		port, _ := nat.NewPort("tcp", containerPort)
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{
			{HostPort: hostPort},
		}
	}

	// Configuración del contenedor
	containerConfig := &container.Config{
		Image:        config.Image,
		Cmd:          config.Cmd,
		Env:          config.Env,
		Tty:          true,
		ExposedPorts: exposedPorts,
	}

	// Configuración del host
	hostConfig := &container.HostConfig{
		Binds:        config.Binds,
		NetworkMode:  container.NetworkMode(config.NetworkName),
		PortBindings: portBindings,
	}

	// Configuración de red
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			config.NetworkName: {},
		},
	}

	// Crear contenedor
	resp, err := dc.cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		config.Name,
	)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// StartContainer inicia un contenedor existente
func (dc *DockerClient) StartContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return dc.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
}

// StopContainer detiene un contenedor
func (dc *DockerClient) StopContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	timeout := 10
	return dc.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
}

// RestartContainer reinicia un contenedor
func (dc *DockerClient) RestartContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	timeout := 10
	return dc.cli.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
}

// RemoveContainer elimina un contenedor (debe estar detenido)
func (dc *DockerClient) RemoveContainer(containerID string, force bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return dc.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	})
}

// GetLogs obtiene los logs de un contenedor
func (dc *DockerClient) GetLogs(containerID string, tail string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	}

	reader, err := dc.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// StreamLogs devuelve un reader para streaming de logs en tiempo real
func (dc *DockerClient) StreamLogs(containerID string) (io.ReadCloser, error) {
	ctx := context.Background()

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
		Tail:       "100",
	}

	return dc.cli.ContainerLogs(ctx, containerID, options)
}

// GetStats obtiene estadísticas del contenedor (CPU, memoria, red)
func (dc *DockerClient) GetStats(containerID string) (types.StatsJSON, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := dc.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return types.StatsJSON{}, err
	}
	defer stats.Body.Close()

	var statsJSON types.StatsJSON
	data, err := io.ReadAll(stats.Body)
	if err != nil {
		return types.StatsJSON{}, err
	}

	if err := json.Unmarshal(data, &statsJSON); err != nil {
		return types.StatsJSON{}, err
	}

	return statsJSON, nil
}

// ListNetworks devuelve todas las redes Docker
func (dc *DockerClient) ListNetworks() ([]types.NetworkResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return dc.cli.NetworkList(ctx, types.NetworkListOptions{})
}

// CreateNetwork crea una nueva red Docker
func (dc *DockerClient) CreateNetwork(name string, driver string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verificar si la red ya existe
	networks, err := dc.cli.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil {
		return "", err
	}

	if len(networks) > 0 {
		return networks[0].ID, nil
	}

	// Crear nueva red
	resp, err := dc.cli.NetworkCreate(ctx, name, types.NetworkCreate{
		Driver:     driver,
		Attachable: true,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// Close cierra la conexión con Docker
func (dc *DockerClient) Close() error {
	return dc.cli.Close()
}
