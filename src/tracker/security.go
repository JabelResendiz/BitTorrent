package tracker

// tracker/security.go
// Sistema de autenticación e integridad para mensajes de sincronización entre trackers.
// Utiliza HMAC-SHA256 para garantizar que los mensajes provienen de trackers legítimos
// y no han sido modificados en tránsito.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
)

// SHARED_SECRET es el secreto compartido entre todos los trackers para autenticación.
//
// NOTA DE SEGURIDAD:
// En un entorno de producción, este secreto debería estar en:
// - Variables de entorno (SYNC_SECRET)
// - Sistema de gestión de secretos (HashiCorp Vault, AWS Secrets Manager)
// - Archivo de configuración externo con permisos restringidos
//
// Para este proyecto académico/demostración, está embebido en la imagen para:
// - Simplicidad: no requiere configuración adicional
// - Automatización: todos los trackers usan el mismo secreto automáticamente
// - Transparencia: facilita la comprensión del sistema
const SHARED_SECRET = "bittorrent-tracker-sync-secret-2025"

// SignMessage calcula la firma HMAC-SHA256 de un mensaje.
//
// Proceso:
// 1. Crea un objeto HMAC usando SHA256 y el secreto compartido
// 2. Alimenta los bytes del mensaje al HMAC
// 3. Calcula el hash resultante (32 bytes)
// 4. Codifica el hash en hexadecimal (64 caracteres)
//
// Parámetros:
//   - message: bytes del mensaje a firmar (típicamente JSON serializado)
//
// Retorna:
//   - string: firma hexadecimal de 64 caracteres
func SignMessage(message []byte) string {
	mac := hmac.New(sha256.New, []byte(SHARED_SECRET))
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateSignature verifica que la firma de un mensaje es válida.
//
// Proceso:
// 1. Recalcula la firma esperada usando SignMessage()
// 2. Compara la firma calculada con la firma recibida
// 3. Usa hmac.Equal() para prevenir timing attacks
//
// Parámetros:
//   - message: bytes del mensaje original (sin el campo signature)
//   - signature: firma hexadecimal a validar
//
// Retorna:
//   - bool: true si la firma es válida, false en caso contrario
func ValidateSignature(message []byte, signature string) bool {
	expectedSignature := SignMessage(message)

	// hmac.Equal() hace comparación en tiempo constante para prevenir
	// ataques de temporización donde un atacante mide el tiempo de
	// comparación para adivinar la firma byte a byte
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// LogSecurityStatus registra el estado de seguridad al iniciar el tracker.
// Informa que la autenticación HMAC está habilitada para sync de trackers.
func LogSecurityStatus() {
	log.Println("[SECURITY] HMAC authentication enabled for tracker synchronization")
	log.Println("[SECURITY] Sync messages will be signed with HMAC-SHA256")
	log.Printf("[SECURITY] Secret fingerprint: %s...%s",
		SHARED_SECRET[:8],
		SHARED_SECRET[len(SHARED_SECRET)-8:])
}
