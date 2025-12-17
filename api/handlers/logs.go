package handlers

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Configuración del upgrader para WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Permitir conexiones desde cualquier origen (para desarrollo)
		// En producción, deberías verificar el origen
		return true
	},
}

// StreamLogs transmite logs de un contenedor en tiempo real mediante WebSocket
// WS /ws/logs/:id
func StreamLogs(c *gin.Context) {
	containerID := c.Param("id")

	// Upgrade HTTP connection a WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("📡 WebSocket connection established for container %s", containerID)

	// Obtener stream de logs desde Docker
	logReader, err := dockerClient.StreamLogs(containerID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get logs stream: %v", err)
		log.Printf("❌ %s", errMsg)
		conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
		return
	}
	defer logReader.Close()

	// Channel para señal de cierre
	done := make(chan struct{})

	// Goroutine para leer mensajes del cliente (para detectar desconexiones)
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				log.Printf("📡 WebSocket client disconnected: %v", err)
				return
			}
		}
	}()

	// Scanner para leer logs línea por línea
	scanner := bufio.NewScanner(logReader)
	scanner.Split(bufio.ScanLines)

	// Ticker para heartbeat (mantener conexión viva)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Loop principal: enviar logs al cliente
	for {
		select {
		case <-done:
			// Cliente desconectado
			log.Printf("📡 Stopping log stream for container %s", containerID)
			return

		case <-ticker.C:
			// Enviar ping para mantener conexión viva
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("❌ Failed to send ping: %v", err)
				return
			}

		default:
			// Leer siguiente línea de log
			if scanner.Scan() {
				logLine := scanner.Text()

				// Enviar log al cliente WebSocket
				if err := conn.WriteMessage(websocket.TextMessage, []byte(logLine)); err != nil {
					log.Printf("❌ Failed to send log message: %v", err)
					return
				}
			} else {
				// Scanner terminó (contenedor detenido o error)
				if err := scanner.Err(); err != nil {
					errMsg := fmt.Sprintf("Error reading logs: %v", err)
					log.Printf("❌ %s", errMsg)
					conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
				} else {
					// EOF - contenedor detenido
					msg := fmt.Sprintf("Container %s stopped or reached end of logs", containerID)
					log.Printf("ℹ️ %s", msg)
					conn.WriteMessage(websocket.TextMessage, []byte(msg))
				}
				return
			}

			// Pequeña pausa para no saturar
			time.Sleep(10 * time.Millisecond)
		}
	}
}
