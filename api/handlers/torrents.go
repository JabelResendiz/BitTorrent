package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// TorrentInfo representa información de un archivo torrent
type TorrentInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// ListTorrents lista todos los archivos .torrent disponibles
// GET /api/torrents
func ListTorrents(c *gin.Context) {
	torrentsDir := "../archives/torrents"

	// Crear directorio si no existe
	if err := os.MkdirAll(torrentsDir, 0755); err != nil {
		log.Printf("❌ Error creating torrents directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to access torrents directory: %v", err),
		})
		return
	}

	// Leer archivos del directorio
	files, err := os.ReadDir(torrentsDir)
	if err != nil {
		log.Printf("❌ Error reading torrents directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read torrents directory: %v", err),
		})
		return
	}

	var torrents []TorrentInfo

	// Filtrar solo archivos .torrent
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) == ".torrent" {
			info, err := file.Info()
			if err != nil {
				log.Printf("⚠️ Warning: could not get info for %s: %v", file.Name(), err)
				continue
			}

			torrents = append(torrents, TorrentInfo{
				Name: file.Name(),
				Path: filepath.Join(torrentsDir, file.Name()),
				Size: info.Size(),
			})
		}
	}

	log.Printf("📂 Found %d torrent file(s)", len(torrents))

	c.JSON(http.StatusOK, torrents)
}

// UploadTorrent sube un nuevo archivo .torrent
// POST /api/torrents/upload
func UploadTorrent(c *gin.Context) {
	torrentsDir := "../archives/torrents"

	// Crear directorio si no existe
	if err := os.MkdirAll(torrentsDir, 0755); err != nil {
		log.Printf("❌ Error creating torrents directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create torrents directory: %v", err),
		})
		return
	}

	// Obtener archivo del form-data
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("❌ No file in request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No file uploaded",
		})
		return
	}

	// Validar extensión
	if filepath.Ext(file.Filename) != ".torrent" {
		log.Printf("❌ Invalid file extension: %s", file.Filename)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File must have .torrent extension",
		})
		return
	}

	// Construir path de destino
	dst := filepath.Join(torrentsDir, file.Filename)

	// Guardar archivo
	if err := c.SaveUploadedFile(file, dst); err != nil {
		log.Printf("❌ Error saving file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to save file: %v", err),
		})
		return
	}

	log.Printf("✅ Torrent uploaded: %s (%d bytes)", file.Filename, file.Size)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"filename": file.Filename,
		"size":     file.Size,
		"path":     dst,
		"message":  "Torrent uploaded successfully",
	})
}

// DeleteTorrent elimina un archivo .torrent
// DELETE /api/torrents/:name
func DeleteTorrent(c *gin.Context) {
	torrentName := c.Param("name")
	torrentsDir := "../archives/torrents"

	// Construir path completo
	torrentPath := filepath.Join(torrentsDir, torrentName)

	// Verificar que el archivo existe
	if _, err := os.Stat(torrentPath); os.IsNotExist(err) {
		log.Printf("❌ Torrent not found: %s", torrentName)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Torrent file not found",
		})
		return
	}

	// Eliminar archivo
	if err := os.Remove(torrentPath); err != nil {
		log.Printf("❌ Error deleting torrent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to delete torrent: %v", err),
		})
		return
	}

	log.Printf("✅ Torrent deleted: %s", torrentName)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Torrent '%s' deleted successfully", torrentName),
	})
}
