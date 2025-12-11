package main

// Configuración del servidor API
const (
	// Puerto del servidor HTTP
	APIPort = ":8090"

	// Orígenes permitidos para CORS (frontend)
	AllowedOrigins = "http://localhost:3000,http://localhost:3001"

	// Directorio de torrents (relativo a la raíz del proyecto)
	TorrentsDir = "../archives/torrents"
)
