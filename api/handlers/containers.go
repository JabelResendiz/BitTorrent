package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"api/docker"

	"github.com/gin-gonic/gin"
)

var dockerClient *docker.DockerClient

// init inicializa el cliente Docker al arrancar
func init() {
	var err error
	dockerClient, err = docker.NewDockerClient()
	if err != nil {
		log.Fatalf("❌ Failed to create Docker client: %v", err)
	}
	log.Println("✅ Docker client initialized successfully")
}

// ContainerRequest representa la petición para crear un contenedor
type ContainerRequest struct {
	ContainerName string `json:"containerName" binding:"required"`
	NetworkName   string `json:"networkName" binding:"required"`
	FolderPath    string `json:"folderPath" binding:"required"`
	ImageName     string `json:"imageName" binding:"required"`
	TorrentFile   string `json:"torrentFile" binding:"required"`
	DiscoveryMode string `json:"discoveryMode" binding:"required"`
	Port          string `json:"port"`
	Bootstrap     string `json:"bootstrap"`
}

// ListContainers devuelve todos los contenedores
// GET /api/containers
func ListContainers(c *gin.Context) {
	containers, err := dockerClient.ListContainers()
	if err != nil {
		log.Printf("❌ Error listing containers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to list containers: %v", err),
		})
		return
	}

	// Formatear respuesta
	var response []gin.H
	for _, container := range containers {
		response = append(response, gin.H{
			"id":      container.ID[:12],
			"name":    strings.TrimPrefix(container.Names[0], "/"),
			"image":   container.Image,
			"state":   container.State,
			"status":  container.Status,
			"created": container.Created,
			"ports":   container.Ports,
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetContainer obtiene información de un contenedor específico
// GET /api/containers/:id
func GetContainer(c *gin.Context) {
	containerID := c.Param("id")

	container, err := dockerClient.GetContainer(containerID)
	if err != nil {
		log.Printf("❌ Error getting container %s: %v", containerID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Container not found: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      container.ID[:12],
		"name":    strings.TrimPrefix(container.Name, "/"),
		"image":   container.Config.Image,
		"state":   container.State.Status,
		"created": container.Created,
		"config":  container.Config,
	})
}

// CreateContainer crea y arranca un nuevo contenedor
// POST /api/containers
func CreateContainer(c *gin.Context) {
	var req ContainerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Construir comando para el contenedor
	cmd := []string{
		"--torrent=/torrents/" + req.TorrentFile,
		"--archives=/data",
		"--hostname=" + req.ContainerName,
		"--discovery-mode=" + req.DiscoveryMode,
	}

	// Agregar parámetros según el modo de descubrimiento
	if req.DiscoveryMode == "overlay" {
		if req.Port != "" {
			cmd = append(cmd, "--overlay-port="+req.Port)
		}
		if req.Bootstrap != "" {
			cmd = append(cmd, "--bootstrap="+req.Bootstrap)
		}
	}

	// Agregar puerto HTTP para el servidor de métricas
	cmd = append(cmd, "--http-port=9091")

	// Expandir tilde (~) a ruta absoluta del home
	folderPath := req.FolderPath
	if strings.HasPrefix(folderPath, "~/") {
		homeDir := "/home/" + os.Getenv("USER")
		if homeDir == "/home/" {
			homeDir = os.Getenv("HOME")
		}
		folderPath = strings.Replace(folderPath, "~", homeDir, 1)
	}

	// Construir binds para volúmenes
	torrentsPath, _ := filepath.Abs("../archives/torrents")
	binds := []string{
		folderPath + ":/data",
		torrentsPath + ":/torrents:ro",
	}

	// Mapeo de puertos (exponer puerto HTTP 9091)
	portBindings := map[string]string{
		"9091": "0", // Docker asignará un puerto aleatorio del host
	}
	if req.DiscoveryMode == "overlay" && req.Port != "" {
		portBindings[req.Port] = req.Port
	}

	// Configuración del contenedor
	config := docker.CreateContainerConfig{
		Name:         req.ContainerName,
		Image:        req.ImageName,
		NetworkName:  req.NetworkName,
		Binds:        binds,
		Cmd:          cmd,
		PortBindings: portBindings,
	}

	// Crear contenedor
	containerID, err := dockerClient.CreateContainer(config)
	if err != nil {
		log.Printf("❌ Error creating container: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create container: %v", err),
		})
		return
	}

	// Iniciar contenedor
	if err := dockerClient.StartContainer(containerID); err != nil {
		log.Printf("❌ Error starting container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Container created but failed to start: %v", err),
		})
		return
	}

	log.Printf("✅ Container created and started: %s (%s)", req.ContainerName, containerID[:12])

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"containerId": containerID[:12],
		"name":        req.ContainerName,
		"message":     "Container created and started successfully",
	})
}

// StartContainer inicia un contenedor detenido
// POST /api/containers/:id/start
func StartContainer(c *gin.Context) {
	containerID := c.Param("id")

	if err := dockerClient.StartContainer(containerID); err != nil {
		log.Printf("❌ Error starting container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to start container: %v", err),
		})
		return
	}

	log.Printf("✅ Container started: %s", containerID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Container started successfully",
	})
}

// StopContainer detiene un contenedor en ejecución
// POST /api/containers/:id/stop
func StopContainer(c *gin.Context) {
	containerID := c.Param("id")

	if err := dockerClient.StopContainer(containerID); err != nil {
		log.Printf("❌ Error stopping container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to stop container: %v", err),
		})
		return
	}

	log.Printf("✅ Container stopped: %s", containerID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Container stopped successfully",
	})
}

// RestartContainer reinicia un contenedor
// POST /api/containers/:id/restart
func RestartContainer(c *gin.Context) {
	containerID := c.Param("id")

	if err := dockerClient.RestartContainer(containerID); err != nil {
		log.Printf("❌ Error restarting container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to restart container: %v", err),
		})
		return
	}

	log.Printf("✅ Container restarted: %s", containerID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Container restarted successfully",
	})
}

// DeleteContainer elimina un contenedor
// DELETE /api/containers/:id
func DeleteContainer(c *gin.Context) {
	containerID := c.Param("id")
	force := c.Query("force") == "true"

	if err := dockerClient.RemoveContainer(containerID, force); err != nil {
		log.Printf("❌ Error removing container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to remove container: %v", err),
		})
		return
	}

	log.Printf("✅ Container removed: %s", containerID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Container removed successfully",
	})
}

// GetLogs obtiene los logs de un contenedor
// GET /api/containers/:id/logs
func GetLogs(c *gin.Context) {
	containerID := c.Param("id")
	tail := c.DefaultQuery("tail", "100")

	logs, err := dockerClient.GetLogs(containerID, tail)
	if err != nil {
		log.Printf("❌ Error getting logs for container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get logs: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}

// GetStats obtiene estadísticas de un contenedor
// GET /api/containers/:id/stats
func GetStats(c *gin.Context) {
	containerID := c.Param("id")

	stats, err := dockerClient.GetStats(containerID)
	if err != nil {
		log.Printf("❌ Error getting stats for container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get stats: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetContainerStatus obtiene el status completo del contenedor desde su HTTP server interno
// GET /api/containers/:id/status
func GetContainerStatus(c *gin.Context) {
	containerID := c.Param("id")

	// Obtener info del contenedor
	container, err := dockerClient.GetContainer(containerID)
	if err != nil {
		log.Printf("❌ Error getting container %s: %v", containerID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Container not found",
		})
		return
	}

	// Obtener puerto HTTP del contenedor (puerto 9091 mapeado)
	ports := container.NetworkSettings.Ports["9091/tcp"]
	if len(ports) == 0 {
		log.Printf("⚠️ Container %s does not expose HTTP port 9091", containerID)
		c.JSON(http.StatusOK, gin.H{
			"error":            "HTTP port not exposed",
			"progress_percent": 0,
			"state":            "unknown",
		})
		return
	}

	httpPort := ports[0].HostPort

	// Hacer request al HTTP server del contenedor
	url := fmt.Sprintf("http://localhost:%s/status", httpPort)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("⚠️ Failed to connect to container HTTP server: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"error":            "Container HTTP server not responding",
			"progress_percent": 0,
			"state":            "starting",
		})
		return
	}
	defer resp.Body.Close()

	// Parsear respuesta
	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		log.Printf("❌ Error parsing status response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse status",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// PauseContainer pausa la descarga del contenedor
// POST /api/containers/:id/pause
func PauseContainer(c *gin.Context) {
	containerID := c.Param("id")

	container, err := dockerClient.GetContainer(containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	ports := container.NetworkSettings.Ports["9091/tcp"]
	if len(ports) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HTTP port not exposed"})
		return
	}

	httpPort := ports[0].HostPort
	url := fmt.Sprintf("http://localhost:%s/pause", httpPort)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Printf("❌ Error pausing container: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pause"})
		return
	}
	defer resp.Body.Close()

	log.Printf("⏸️  Container paused: %s", containerID)
	c.JSON(http.StatusOK, gin.H{"success": true, "paused": true})
}

// ResumeContainer reanuda la descarga del contenedor
// POST /api/containers/:id/resume
func ResumeContainer(c *gin.Context) {
	containerID := c.Param("id")

	container, err := dockerClient.GetContainer(containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	ports := container.NetworkSettings.Ports["9091/tcp"]
	if len(ports) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HTTP port not exposed"})
		return
	}

	httpPort := ports[0].HostPort
	url := fmt.Sprintf("http://localhost:%s/resume", httpPort)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Printf("❌ Error resuming container: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resume"})
		return
	}
	defer resp.Body.Close()

	log.Printf("▶️  Container resumed: %s", containerID)
	c.JSON(http.StatusOK, gin.H{"success": true, "paused": false})
}

// ListNetworks lista todas las redes Docker
// GET /api/networks
func ListNetworks(c *gin.Context) {
	networks, err := dockerClient.ListNetworks()
	if err != nil {
		log.Printf("❌ Error listing networks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to list networks: %v", err),
		})
		return
	}

	var response []gin.H
	for _, net := range networks {
		response = append(response, gin.H{
			"id":      net.ID[:12],
			"name":    net.Name,
			"driver":  net.Driver,
			"scope":   net.Scope,
			"created": net.Created,
		})
	}

	c.JSON(http.StatusOK, response)
}

// CreateNetwork crea una nueva red Docker
// POST /api/networks
func CreateNetwork(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Driver string `json:"driver"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if req.Driver == "" {
		req.Driver = "bridge"
	}

	networkID, err := dockerClient.CreateNetwork(req.Name, req.Driver)
	if err != nil {
		log.Printf("❌ Error creating network: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create network: %v", err),
		})
		return
	}

	log.Printf("✅ Network created: %s (%s)", req.Name, networkID[:12])

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"networkId": networkID[:12],
		"name":      req.Name,
		"message":   "Network created successfully",
	})
}
