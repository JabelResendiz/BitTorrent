package main

import (
	"log"
	"net/http"

	"api/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Crear router con Gin
	router := gin.Default()

	// Configurar CORS para permitir peticiones desde el frontend (Next.js en puerto 3000)
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "BitTorrent API",
			"version": "1.0.0",
		})
	})

	// API Routes
	api := router.Group("/api")
	{
		// Containers endpoints
		api.GET("/containers", handlers.ListContainers)
		api.POST("/containers", handlers.CreateContainer)
		api.GET("/containers/:id", handlers.GetContainer)
		api.POST("/containers/:id/start", handlers.StartContainer)
		api.POST("/containers/:id/stop", handlers.StopContainer)
		api.POST("/containers/:id/restart", handlers.RestartContainer)
		api.DELETE("/containers/:id", handlers.DeleteContainer)
		api.GET("/containers/:id/logs", handlers.GetLogs)
		api.GET("/containers/:id/stats", handlers.GetStats)
		api.GET("/containers/:id/status", handlers.GetContainerStatus)
		api.POST("/containers/:id/pause", handlers.PauseContainer)
		api.POST("/containers/:id/resume", handlers.ResumeContainer)

		// Torrents endpoints
		api.GET("/torrents", handlers.ListTorrents)
		api.POST("/torrents/upload", handlers.UploadTorrent)
		api.DELETE("/torrents/:name", handlers.DeleteTorrent)

		// Network endpoints
		api.GET("/networks", handlers.ListNetworks)
		api.POST("/networks", handlers.CreateNetwork)
	}

	// WebSocket endpoint para logs en tiempo real
	router.GET("/ws/logs/:id", handlers.StreamLogs)

	// Iniciar servidor
	log.Printf("🚀 BitTorrent API Server starting on http://localhost%s", APIPort)
	log.Printf("📡 WebSocket available at ws://localhost%s/ws/logs/:id", APIPort)
	log.Printf("🌐 Accepting requests from http://localhost:3000")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err := router.Run(APIPort); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
