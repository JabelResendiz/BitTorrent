# Seguridad HMAC en Sincronización de Trackers

## 📋 Tabla de Contenidos
- [Introducción](#introducción)
- [¿Qué es HMAC?](#qué-es-hmac)
- [Arquitectura de la Solución](#arquitectura-de-la-solución)
- [Implementación Detallada](#implementación-detallada)
- [Flujo de Operación](#flujo-de-operación)
- [Protección contra Ataques](#protección-contra-ataques)
- [Archivos Modificados](#archivos-modificados)
- [Uso y Configuración](#uso-y-configuración)
- [Logs y Monitoreo](#logs-y-monitoreo)
- [Consideraciones de Seguridad](#consideraciones-de-seguridad)

---

## Introducción

Este documento describe la implementación de un sistema de autenticación e integridad para la sincronización entre trackers distribuidos usando **HMAC-SHA256** (Hash-based Message Authentication Code).

### Motivación

En un sistema de trackers distribuidos, los trackers se comunican entre sí para sincronizar el estado de los swarms mediante HTTP. Sin un mecanismo de seguridad, el sistema es vulnerable a:

- **Mensajes falsificados**: Un atacante podría hacerse pasar por un tracker legítimo
- **Manipulación de datos**: Los mensajes podrían ser interceptados y modificados
- **Inyección de datos maliciosos**: Peers falsos podrían ser insertados en el sistema

### Objetivos de Seguridad

✅ **Autenticación**: Verificar que los mensajes provienen de trackers legítimos
✅ **Integridad**: Garantizar que los mensajes no han sido modificados
✅ **Simplicidad**: Implementación transparente sin configuración adicional
✅ **Eficiencia**: Overhead mínimo en la sincronización

---

## ¿Qué es HMAC?

### Definición

**HMAC** (Hash-based Message Authentication Code) es un mecanismo de autenticación que utiliza:
- Una **función hash criptográfica** (SHA256 en nuestro caso)
- Una **clave secreta compartida** (conocida solo por los trackers legítimos)

### Fórmula Básica

```
HMAC(K, M) = H((K ⊕ opad) || H((K ⊕ ipad) || M))
```

Donde:
- `K` = clave secreta
- `M` = mensaje a autenticar
- `H` = función hash (SHA256)
- `⊕` = operación XOR
- `||` = concatenación
- `opad` = outer padding (0x5c repetido)
- `ipad` = inner padding (0x36 repetido)

### ¿Por qué HMAC?

- **Seguridad probada**: Estándar RFC 2104, ampliamente utilizado
- **Resistente a colisiones**: Prácticamente imposible generar dos mensajes con el mismo HMAC
- **Clave secreta**: Sin la clave, no se puede generar un HMAC válido
- **Eficiente**: Cálculo rápido, overhead mínimo

---

## Arquitectura de la Solución

### Componentes

```
┌─────────────────────────────────────────────────────────────┐
│                    Sistema de Trackers                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────┐                            ┌──────────┐       │
│  │ Tracker1 │                            │ Tracker2 │       │
│  │          │                            │          │       │
│  │ security.go ────────────────────────► security.go│       │
│  │   (Sign)   │   Mensaje + Firma HMAC │ (Validate)│       │
│  │            │                            │          │       │
│  └──────────┘                            └──────────┘       │
│       │                                         │            │
│       │    Ambos comparten el mismo secreto    │            │
│       └─────────────────────────────────────────┘            │
│              "bittorrent-tracker-sync-secret-2025"           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Flujo de Datos

```
ENVÍO (Tracker1 → Tracker2):
1. Crear SyncMessage con datos del swarm
2. Serializar a JSON (sin firma)
3. Calcular HMAC-SHA256 del JSON
4. Añadir firma al mensaje
5. Enviar JSON completo via HTTP POST

RECEPCIÓN (Tracker2 recibe de Tracker1):
1. Recibir JSON con firma
2. Extraer firma recibida
3. Reconstruir mensaje sin firma
4. Calcular HMAC-SHA256 esperado
5. Comparar firmas (tiempo constante)
6. ✅ Aceptar o ❌ Rechazar
```

---

## Implementación Detallada

### 1. Módulo de Seguridad (`security.go`)

#### Secreto Compartido

```go
const SHARED_SECRET = "bittorrent-tracker-sync-secret-2025"
```

**Diseño:**
- Embebido en el código fuente
- Compartido por todos los trackers en la misma imagen Docker
- Cambio requiere rebuild de la imagen

**Justificación (Proyecto Académico):**
- ✅ Cero configuración adicional
- ✅ Automatización completa
- ✅ Simplicidad de despliegue
- ⚠️ En producción: usar variables de entorno o vault

#### Función de Firma

```go
func SignMessage(message []byte) string {
    // 1. Crear objeto HMAC con SHA256 y secreto
    mac := hmac.New(sha256.New, []byte(SHARED_SECRET))
    
    // 2. Alimentar bytes del mensaje
    mac.Write(message)
    
    // 3. Calcular hash (32 bytes)
    hashBytes := mac.Sum(nil)
    
    // 4. Codificar en hexadecimal (64 caracteres)
    return hex.EncodeToString(hashBytes)
}
```

**Características:**
- Entrada: bytes del mensaje JSON
- Salida: string hexadecimal de 64 caracteres
- Determinista: mismo mensaje = misma firma
- Unidireccional: no se puede revertir

#### Función de Validación

```go
func ValidateSignature(message []byte, signature string) bool {
    expectedSignature := SignMessage(message)
    
    // Comparación en tiempo constante (previene timing attacks)
    return hmac.Equal([]byte(expectedSignature), []byte(signature))
}
```

**Seguridad:**
- Usa `hmac.Equal()` en lugar de `==`
- Comparación en **tiempo constante**
- Previene **timing attacks** (medir tiempo de comparación)

---

### 2. Estructura de Mensajes (`sync_messages.go`)

#### Mensaje de Sincronización

```go
type SyncMessage struct {
    FromNodeID string                      `json:"from_node_id"` // Emisor
    Timestamp  HLC                         `json:"timestamp"`    // Reloj lógico
    Swarms     map[string]map[string]*Peer `json:"swarms"`       // Datos
    Signature  string                      `json:"signature"`    // ← NUEVO
}
```

**Campo Signature:**
- String hexadecimal de 64 caracteres
- Calculado sobre el mensaje completo (sin el campo signature)
- Incluido en el JSON enviado

**Ejemplo de Mensaje:**

```json
{
  "from_node_id": "tracker1",
  "timestamp": {
    "wall_time": 1702382400,
    "logical": 15,
    "node_id": "tracker1"
  },
  "swarms": {
    "abc123def456": {
      "peer1": {
        "peer_id": "peer1",
        "ip": "192.168.1.10",
        "port": 6881,
        "completed": false,
        "deleted": false
      }
    }
  },
  "signature": "a7f3e9d1c4b8f2e5a9d7c3e1f8b4a2d6c5e7f9a3b1d8e2f4a6c8e0f2a4b6c8d0"
}
```

---

### 3. Proceso de Envío (`sync.go` - `pushToPeer`)

```go
func (sm *SyncManager) pushToPeer(remotePeer string, msg *SyncMessage) {
    url := fmt.Sprintf("http://%s/sync", remotePeer)

    // PASO 1: Serializar mensaje SIN firma
    msg.Signature = "" // Asegurar que está vacío
    data, err := json.Marshal(msg)
    if err != nil {
        log.Printf("[SYNC] Error marshaling: %v", err)
        return
    }

    // PASO 2: Calcular firma HMAC
    signature := SignMessage(data)
    
    // PASO 3: Añadir firma al mensaje
    msg.Signature = signature
    
    // PASO 4: Serializar mensaje completo con firma
    data, err = json.Marshal(msg)
    if err != nil {
        log.Printf("[SYNC] Error marshaling signed message: %v", err)
        return
    }

    // PASO 5: Log de seguridad
    log.Printf("[SYNC] Sending signed message to %s (signature: %s...)", 
               remotePeer, signature[:16])

    // PASO 6: Enviar via HTTP POST
    resp, err := http.Post(url, "application/json", bytes.NewReader(data))
    // ... manejo de respuesta ...
}
```

**Flujo:**
1. Limpiar campo signature
2. Convertir a JSON
3. Calcular HMAC del JSON
4. Añadir firma al struct
5. Reserializar con firma
6. Enviar

---

### 4. Proceso de Recepción (`sync.go` - `handleSync`)

```go
func (sl *SyncListener) handleSync(w http.ResponseWriter, r *http.Request) {
    // PASO 1: Leer cuerpo de la petición
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    // PASO 2: Deserializar JSON
    var msg SyncMessage
    if err := json.Unmarshal(body, &msg); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    // PASO 3: Extraer firma recibida
    receivedSignature := msg.Signature
    
    // PASO 4: Verificar que existe firma
    if receivedSignature == "" {
        log.Printf("[SECURITY] ❌ Rejected: missing signature from %s", msg.FromNodeID)
        http.Error(w, "Unauthorized: missing signature", http.StatusUnauthorized)
        return
    }

    // PASO 5: Reconstruir mensaje sin firma
    msg.Signature = ""
    messageBytes, err := json.Marshal(msg)
    if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // PASO 6: Validar firma
    if !ValidateSignature(messageBytes, receivedSignature) {
        log.Printf("[SECURITY] ❌ Invalid signature from %s (attack?)", msg.FromNodeID)
        log.Printf("[SECURITY] Remote IP: %s", r.RemoteAddr)
        http.Error(w, "Unauthorized: invalid signature", http.StatusUnauthorized)
        return
    }

    // PASO 7: Firma válida - procesar mensaje
    log.Printf("[SYNC] ✅ Valid signature from %s with %d swarms", 
               msg.FromNodeID, len(msg.Swarms))

    sl.tracker.MergeSwarms(&msg)
    
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
```

**Validaciones:**
1. ✅ Firma presente
2. ✅ Firma válida (HMAC coincide)
3. ✅ Mensaje íntegro (no modificado)
4. ✅ Origen auténtico (conoce el secreto)

---

## Flujo de Operación

### Caso Normal: Sincronización Exitosa

```
Tracker1                                    Tracker2
   |                                           |
   |  1. Crear SyncMessage                     |
   |     {from: "tracker1", swarms: {...}}     |
   |                                           |
   |  2. Calcular HMAC                         |
   |     sig = HMAC("message")                 |
   |     sig = "a7f3e9d1..."                   |
   |                                           |
   |  3. POST /sync                            |
   |─────────────────────────────────────────►│
   |     {message + signature}                 │
   |                                           │
   |                                4. Extraer firma
   |                                   "a7f3e9d1..."
   |                                           │
   |                                5. Recalcular HMAC
   |                                   expected = "a7f3e9d1..."
   |                                           │
   |                                6. Comparar
   |                                   ✅ Match!
   |                                           │
   |                                7. MergeSwarms()
   |                                           │
   |  8. HTTP 200 OK                           │
   |◄─────────────────────────────────────────│
   |                                           |
```

**Logs en Tracker1:**
```
[SYNC] Sending signed message to tracker2:9090 (signature: a7f3e9d1c4b8f2e5...)
[SYNC] Successfully pushed to tracker2:9090
```

**Logs en Tracker2:**
```
[SYNC] ✅ Valid signature from tracker1 with 3 swarms
[SYNC] Merging swarms from node tracker1
[SYNC] Added new peer peer1 to swarm abc123
```

---

### Caso de Ataque: Mensaje Falsificado

```
Atacante                                   Tracker2
   |                                           |
   |  1. Crear mensaje falso                   |
   |     {from: "tracker1", swarms: {FAKE}}    |
   |                                           |
   |  2. Firma inventada o ausente             |
   |     sig = "fake123..." o sin sig          |
   |                                           |
   |  3. POST /sync                            |
   |─────────────────────────────────────────►│
   |     {mensaje_falso + firma_falsa}         │
   |                                           │
   |                                4. Extraer firma
   |                                   "fake123..."
   |                                           │
   |                                5. Recalcular HMAC
   |                                   expected = "b8c2d4e6..."
   |                                           │
   |                                6. Comparar
   |                                   ❌ NO Match!
   |                                           │
   |                                7. Log ataque
   |                                   RECHAZAR
   |                                           │
   |  8. HTTP 401 Unauthorized                 │
   |◄─────────────────────────────────────────│
   |                                           |
```

**Logs en Tracker2:**
```
[SECURITY] ❌ Rejected sync from tracker1: invalid signature (potential attack)
[SECURITY] Remote IP: 172.18.0.50:45678
```

---

### Caso de Ataque: Mensaje Modificado (MITM)

```
Tracker1          Atacante          Tracker2
   |                 |                  |
   | Mensaje legítimo|                  |
   |────────────────►| Intercepta       |
   |                 | Modifica datos   |
   |                 | (firma ya no es válida)
   |                 |                  |
   |                 | Mensaje modificado|
   |                 |─────────────────►|
   |                 |                  |
   |                 |        Recalcula HMAC
   |                 |                  |
   |                 |        ❌ No coincide
   |                 |                  |
   |                 |   401 Unauthorized|
   |                 |◄─────────────────|
```

**¿Por qué falla?**

El atacante modifica el contenido pero no puede recalcular la firma válida porque **no conoce el secreto compartido**.

---

## Protección contra Ataques

### 1. **Suplantación de Identidad (Spoofing)**

**Ataque:** Un atacante intenta hacerse pasar por tracker1

```bash
curl -X POST http://tracker2:9090/sync -d '{
  "from_node_id": "tracker1",
  "swarms": {"malicious": {...}},
  "signature": "firma_inventada_123"
}'
```

**Protección:**
- ❌ Firma inválida (no conoce el secreto)
- ❌ Mensaje rechazado con 401
- 📋 Ataque registrado en logs

---

### 2. **Manipulación de Datos (Tampering)**

**Ataque:** Interceptar y modificar mensaje legítimo

```json
// Mensaje original de tracker1:
{"from_node_id": "tracker1", "swarms": {...}, "signature": "abc123..."}

// Atacante modifica:
{"from_node_id": "tracker1", "swarms": {...DATOS_MODIFICADOS...}, "signature": "abc123..."}
```

**Protección:**
- ❌ Firma calculada ≠ firma recibida
- ❌ Cambio en datos invalida la firma
- ❌ Mensaje rechazado

---

### 3. **Replay Attack**

**Ataque:** Reenviar mensajes antiguos capturados

**Estado Actual:**
- ⚠️ Parcialmente protegido por HLC (reloj lógico)
- ⚠️ Mensajes muy antiguos podrían aceptarse si la firma es válida

**Mejora Futura (Opcional):**
- Añadir campo `nonce` único por mensaje
- Mantener caché de nonces recientes
- Rechazar mensajes con nonce duplicado

```go
type SyncMessage struct {
    // ... campos existentes ...
    Nonce     string `json:"nonce"`     // UUID único
    Signature string `json:"signature"`
}
```

---

### 4. **Timing Attack**

**Ataque:** Medir tiempo de comparación de firmas para adivinar bytes

**Protección:**
```go
// ❌ VULNERABLE:
if expectedSignature == receivedSignature { ... }
// Comparación se detiene en el primer byte diferente

// ✅ SEGURO:
return hmac.Equal([]byte(expectedSignature), []byte(receivedSignature))
// Comparación en TIEMPO CONSTANTE
// Siempre compara todos los bytes
```

---

### 5. **Man-in-the-Middle (MITM)**

**Estado Actual:**
- ✅ Integridad garantizada (no pueden modificar)
- ✅ Autenticación garantizada (no pueden falsificar)
- ⚠️ Sin encriptación (contenido visible)

**Consideraciones:**
- Para red local confiable: HMAC es suficiente
- Para internet público: considerar HTTPS/TLS

---

## Archivos Modificados

### Nuevos Archivos

#### `src/tracker/security.go`
```
- Constante SHARED_SECRET
- Función SignMessage()
- Función ValidateSignature()
- Función LogSecurityStatus()
```

### Archivos Modificados

#### `src/tracker/sync_messages.go`
```diff
 type SyncMessage struct {
     FromNodeID string                      `json:"from_node_id"`
     Timestamp  HLC                         `json:"timestamp"`
     Swarms     map[string]map[string]*Peer `json:"swarms"`
+    Signature  string                      `json:"signature"`
 }
```

#### `src/tracker/sync.go`
```diff
 func (sm *SyncManager) pushToPeer(remotePeer string, msg *SyncMessage) {
+    // Serializar sin firma
+    msg.Signature = ""
+    data, _ := json.Marshal(msg)
+    
+    // Calcular y añadir firma
+    msg.Signature = SignMessage(data)
+    data, _ = json.Marshal(msg)
+    
+    log.Printf("[SYNC] Sending signed message...")
     http.Post(url, "application/json", bytes.NewReader(data))
 }

 func (sl *SyncListener) handleSync(w http.ResponseWriter, r *http.Request) {
     var msg SyncMessage
     json.Unmarshal(body, &msg)
     
+    // Validar firma
+    receivedSig := msg.Signature
+    msg.Signature = ""
+    messageBytes, _ := json.Marshal(msg)
+    
+    if !ValidateSignature(messageBytes, receivedSig) {
+        log.Printf("[SECURITY] ❌ Invalid signature")
+        http.Error(w, "Unauthorized", 401)
+        return
+    }
+    
+    log.Printf("[SYNC] ✅ Valid signature")
     sl.tracker.MergeSwarms(&msg)
 }
```

#### `src/tracker/cmd/main.go`
```diff
 if len(remotePeers) > 0 {
     log.Printf("Starting distributed sync...")
+    tracker.LogSecurityStatus()
     t.StartSyncListener(*syncListen)
     t.StartSyncManager(...)
 }
```

---

## Uso y Configuración

### Compilación

```bash
# Rebuild de la imagen del tracker con seguridad HMAC
docker build -t tracker_img -f src/tracker/Dockerfile .
```

### Despliegue

Los comandos de ejecución **NO CAMBIAN**:

```bash
# Tracker 1
docker run \
  --name tracker1 \
  --hostname tracker1 \
  --network net \
  --publish 8081:8080 \
  --publish 9091:9090 \
  tracker_img \
  -sync-peers "tracker2:9090,tracker3:9090"

# Tracker 2
docker run \
  --name tracker2 \
  --hostname tracker2 \
  --network net \
  --publish 8082:8080 \
  --publish 9092:9090 \
  tracker_img \
  -sync-peers "tracker1:9090,tracker3:9090"

# Tracker 3
docker run \
  --name tracker3 \
  --hostname tracker3 \
  --network net \
  --publish 8083:8080 \
  --publish 9093:9090 \
  tracker_img \
  -sync-peers "tracker1:9090,tracker2:9090"
```

### Verificación

```bash
# Ver logs de seguridad
docker logs tracker1 | grep SECURITY
docker logs tracker1 | grep "✅\|❌"

# Monitorear sincronización
docker logs -f tracker1 | grep SYNC
```

---

## Logs y Monitoreo

### Logs de Inicio

```
Tracker node-id: tracker1, data: /data/tracker1_data.json
Starting distributed sync with 2 peers
[SECURITY] HMAC authentication enabled for tracker synchronization
[SECURITY] Sync messages will be signed with HMAC-SHA256
[SECURITY] Secret fingerprint: bittorre...ret-2025
[SYNC] Sync listener started on [::]:9090
[SYNC] Starting sync manager with 2 remote peers, interval=15s
```

### Logs de Operación Normal

```
[SYNC] Pushing state to 2 peers (swarms=3)
[SYNC] Sending signed message to tracker2:9090 (signature: a7f3e9d1c4b8f2e5...)
[SYNC] Successfully pushed to tracker2:9090
[SYNC] Sending signed message to tracker3:9090 (signature: b8c2d4e6f8a0b2c4...)
[SYNC] Successfully pushed to tracker3:9090

[SYNC] ✅ Valid signature from tracker1 with 3 swarms
[SYNC] Merging swarms from node tracker1
[SYNC] Added new peer peer1 to swarm abc123de
```

### Logs de Seguridad (Ataque Detectado)

```
[SECURITY] ❌ Rejected sync from tracker1: missing signature
[SECURITY] ❌ Rejected sync from tracker2: invalid signature (potential attack)
[SECURITY] Remote IP: 172.18.0.50:45678
```

### Filtros Útiles

```bash
# Solo mensajes de seguridad
docker logs tracker1 | grep SECURITY

# Solo validaciones exitosas
docker logs tracker1 | grep "✅"

# Solo ataques detectados
docker logs tracker1 | grep "❌"

# Firmas enviadas
docker logs tracker1 | grep "Sending signed"

# Seguimiento en tiempo real
docker logs -f tracker1 --tail 50 | grep -E "SECURITY|✅|❌"
```

---

## Consideraciones de Seguridad

### Fortalezas

✅ **Autenticación robusta**: Solo trackers con el secreto pueden enviar mensajes válidos
✅ **Integridad garantizada**: Cualquier modificación invalida la firma
✅ **Sin overhead significativo**: SHA256 es muy rápido
✅ **Resistente a timing attacks**: Comparación en tiempo constante
✅ **Protocolo estándar**: HMAC-SHA256 es ampliamente utilizado y probado

### Limitaciones (Proyecto Académico)

⚠️ **Secreto embebido**: En producción debería estar en variables de entorno o vault
⚠️ **Sin encriptación**: El contenido del mensaje es visible (solo integridad, no confidencialidad)
⚠️ **Sin protección completa contra replay**: Mensajes antiguos con firma válida podrían repetirse
⚠️ **Gestión de secretos**: Cambiar el secreto requiere rebuild de todas las imágenes

### Mejoras Futuras (Opcional)

#### 1. Variable de Entorno

```bash
docker run -e SYNC_SECRET="secreto_personalizado" tracker_img
```

```go
func getSharedSecret() string {
    if secret := os.Getenv("SYNC_SECRET"); secret != "" {
        return secret
    }
    return SHARED_SECRET // fallback
}
```

#### 2. Protección contra Replay

```go
type SyncMessage struct {
    // ... campos existentes ...
    Nonce     string `json:"nonce"`
    Signature string `json:"signature"`
}

// En el tracker:
var recentNonces sync.Map // cache de nonces recientes

func validateNonce(nonce string) bool {
    if _, exists := recentNonces.LoadOrStore(nonce, true); exists {
        return false // nonce duplicado
    }
    return true
}
```

#### 3. HTTPS/TLS

Para entornos de producción, combinar con HTTPS:

```go
// En vez de http.Post:
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{...},
    },
}
```

#### 4. Rotación de Secretos

```go
const SHARED_SECRET_V1 = "secreto-viejo"
const SHARED_SECRET_V2 = "secreto-nuevo"

func ValidateSignature(msg []byte, sig string) bool {
    // Intentar con secreto actual
    if validateWith(msg, sig, SHARED_SECRET_V2) {
        return true
    }
    // Fallback a secreto anterior (período de transición)
    return validateWith(msg, sig, SHARED_SECRET_V1)
}
```

---

## Conclusión

La implementación de HMAC-SHA256 proporciona una **capa sólida de seguridad** para la sincronización entre trackers distribuidos:

- ✅ **Autenticación**: Solo trackers legítimos pueden comunicarse
- ✅ **Integridad**: Los mensajes no pueden ser modificados
- ✅ **Simplicidad**: Cero configuración adicional
- ✅ **Transparencia**: Logs claros de validaciones y ataques

El sistema es **apropiado para un entorno académico/demostración** donde se prioriza la simplicidad y claridad del código sobre la complejidad de gestión de secretos de nivel empresarial.

Para un despliegue en producción, se recomendaría:
1. Externalizar el secreto (variables de entorno)
2. Añadir protección contra replay attacks (nonces)
3. Considerar HTTPS para encriptación del canal
4. Implementar rotación de secretos

---

## Referencias

- **RFC 2104**: HMAC: Keyed-Hashing for Message Authentication
- **FIPS PUB 180-4**: Secure Hash Standard (SHA-256)
- **Go crypto/hmac**: https://pkg.go.dev/crypto/hmac
- **Go crypto/sha256**: https://pkg.go.dev/crypto/sha256

---

**Fecha de Implementación**: Diciembre 2025  
**Versión del Documento**: 1.0  
**Autor**: Sistema de Seguridad BitTorrent Tracker
