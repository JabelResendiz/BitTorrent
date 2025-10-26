# Changelog - Implementación Round-Robin

## [v2.1.0] - 2025-10-26

### 📊 Sistema de Logging Detallado

#### Logs de Descarga Paralela
- **Descripción**: Sistema completo de logging que muestra desde qué peer se obtiene cada bloque
- **Beneficio**: Visibilidad en tiempo real del comportamiento Round-Robin
- **Archivos**: `manager_broadcast.go`, `manager.go`

#### Nuevas Salidas de Log

1. **Solicitud de bloques (REQUEST)**:
   ```
   → Solicitando bloque 0 de pieza 0 a peer 10.0.1.5:6881
   ```

2. **Recepción de bloques (PIECE)**:
   ```
   ✓ Recibido bloque 0 de pieza 0 desde peer 10.0.1.5:6881 (offset 0, tamaño 16384 bytes)
   ```

3. **Resumen estadístico al completar pieza**:
   ```
   ═══════════════════════════════════════════════
   ✓ Pieza 0 completada (Round-Robin)
   ═══════════════════════════════════════════════
   Distribución de bloques por peer:
     • Peer 10.0.1.5:6881: 22 bloques
     • Peer 10.0.1.6:6881: 21 bloques
     • Peer 10.0.1.7:6881: 21 bloques
   Total: 64 bloques
   ═══════════════════════════════════════════════
   ```

#### Cambios Técnicos

**PieceDownload**:
- ➕ Campo `blocksReceived map[string]int`: Contador de bloques por peer

**DownloadPieceParallel()**:
- ➕ Log de cada REQUEST enviado con dirección del peer

**handleMessage() case MsgPiece**:
- ➕ Log de cada PIECE recibido con origen del peer
- ➕ Incremento de contador `blocksReceived[peerAddr]`
- ➕ Resumen estadístico al completar pieza

---

## [v2.0.0] - 2025-10-26

### 🚀 Características Nuevas

#### Algoritmo Round-Robin para Descarga Paralela
- **Descripción**: Implementado algoritmo que distribuye bloques de una pieza entre múltiples peers simultáneamente
- **Impacto**: Mejora throughput hasta 3x cuando hay múltiples peers disponibles
- **Archivos**: `manager_broadcast.go`, `manager.go`, `message.go`

### 🔧 Cambios Técnicos

#### `src/peerwire/manager_broadcast.go`
- ➕ **Nueva estructura** `PieceDownload`:
  - `pieceIndex`: índice de la pieza
  - `blocksPending`: map de bloques pendientes
  - `blocksInProgress`: map de bloques en descarga activa
  
- ➕ **Campos agregados a** `Manager`:
  - `pieceDownloads map[int]*PieceDownload`: tracking de descargas activas
  - `downloadsMu sync.Mutex`: protección thread-safe

- ➕ **Nuevo método** `calculateNumBlocks(pieceIndex int) int`:
  - Calcula número de bloques en una pieza
  - Considera última pieza que puede ser más corta

- ➕ **Nuevo método** `DownloadPieceParallel(pieceIndex int)`:
  - Filtra peers elegibles (RemoteHasPiece + !PeerChoking)
  - Crea PieceDownload para tracking
  - Distribuye bloques en Round-Robin entre peers
  - Envía REQUEST a cada peer con su bloque asignado

#### `src/peerwire/manager.go`
- 🔄 **Modificado** `case MsgUnchoke`:
  - **Antes**: `p.requestNextBlocks(piece)` (secuencial)
  - **Después**: `p.manager.DownloadPieceParallel(piece)` (paralelo)
  - **Impacto**: Primera pieza se descarga de múltiples peers

- 🔄 **Modificado** `case MsgPiece`:
  - **Antes**: Pide siguiente bloque automáticamente (secuencial)
  - **Después**: Tracking con `blocksPending` y sin petición automática
  - **Lógica nueva**:
    1. Guardar bloque en storage
    2. Marcar bloque como recibido en PieceDownload
    3. Si todos bloques recibidos → limpiar tracking
    4. Si pieza verificada → elegir siguiente pieza con DownloadPieceParallel()

#### `src/peerwire/message.go`
- ➕ **Nuevo método** `SendBlockRequest(index, begin, length uint32)`:
  - Construye payload de 12 bytes (index + begin + length)
  - Envía mensaje MsgRequest
  - Usado por DownloadPieceParallel para enviar requests

### 📊 Mejoras de Rendimiento

#### Throughput
```
Configuración: 3 peers, 10 MB/s cada uno, pieza de 1 MB

Antes (Secuencial):
  - 1 peer descarga todos los bloques
  - Throughput efectivo: 10 MB/s
  - Tiempo: 100ms para 1 MB

Después (Round-Robin):
  - 3 peers descargan simultáneamente
  - Throughput efectivo: ~25-30 MB/s
  - Tiempo: ~35ms para 1 MB
  
Mejora: 2.5-3x más rápido
```

#### Latencia
```
Bloques por pieza: 64 (1 MB / 16 KB)
Latencia de red: 10ms por request

Antes (Secuencial):
  - 64 requests * 10ms = 640ms
  
Después (Round-Robin con 3 peers):
  - 22 requests * 10ms = 220ms (Peer1)
  - 21 requests * 10ms = 210ms (Peer2)
  - 21 requests * 10ms = 210ms (Peer3)
  - Total: 220ms (máximo)

Reducción: 65% menos latencia
```

### 🔒 Thread-Safety

#### Sincronización Agregada
- `Manager.downloadsMu`: Protege acceso a `pieceDownloads` map
- Múltiples goroutines (ReadLoop de cada peer) acceden concurrentemente
- Lock/Unlock al modificar `blocksPending` y `blocksInProgress`

### 🧪 Testing

#### Escenarios Probados
1. ✅ **Compilación**: `go build ./...` exitoso
2. ✅ **Análisis estático**: Sin errores de lint
3. ⏳ **Prueba funcional**: Pendiente (requiere Docker Swarm activo)

#### Comandos de Prueba
```bash
# Rebuild cliente
./rebuild-client.sh

# Levantar tracker
docker service create --name tracker ...

# Seeder
docker run --name seeder --network net \
  -v ~/archives:/app/src/archives \
  client --torrent=vid.torrent --hostname=seeder

# Leechers (descargan en paralelo)
docker run --name leecher1 --network net \
  -v ~/leecher1:/app/src/archives \
  client --torrent=vid.torrent --hostname=leecher1
```

### 📝 Archivos de Documentación Creados

1. **ROUND_ROBIN_IMPLEMENTATION.md**
   - Descripción completa del algoritmo
   - Arquitectura de la implementación
   - Comparación antes/después
   - Ventajas y limitaciones

2. **DIAGRAMA_ROUND_ROBIN.md**
   - Diagrama de flujo completo
   - Timeline de descarga
   - Estado del Manager durante ejecución
   - Comparación de throughput

3. **rebuild-client.sh**
   - Script automatizado para rebuild
   - Compila código Go
   - Construye imagen Docker
   - Instrucciones de uso

### ⚠️ Limitaciones Conocidas

1. **No considera velocidad de peers**: Todos reciben igual cantidad de bloques
2. **Sin pipeline**: Envía todos los requests inmediatamente (puede saturar red)
3. **Sin retry logic**: Si peer falla después de asignar bloque, no se reasigna
4. **Asume peers homogéneos**: No mide ni adapta basado en throughput real

### 🚀 Próximas Mejoras Planificadas

1. **Weighted Round-Robin**: Asignar más bloques a peers rápidos
2. **Pipeline con ventana deslizante**: No enviar todos requests a la vez
3. **Retry y reasignación**: Si peer falla, reasignar bloque a otro peer
4. **Endgame mode**: Al final, pedir bloques pendientes a todos los peers
5. **Métricas de throughput**: Medir velocidad real de cada peer

### 🔗 Commits Relacionados

- `feat: Add PieceDownload structure for parallel tracking`
- `feat: Implement DownloadPieceParallel Round-Robin algorithm`
- `refactor: Change MsgUnchoke to use parallel download`
- `refactor: Update MsgPiece handler for Round-Robin tracking`
- `feat: Add SendBlockRequest method to PeerConn`
- `docs: Add comprehensive Round-Robin documentation`

---

## Versiones Anteriores

### [v1.0.0] - 2025-10-25

#### Características Base
- ✅ Cliente BitTorrent funcional
- ✅ Tracker HTTP
- ✅ Peer Wire Protocol
- ✅ Descarga secuencial (1 peer por pieza)
- ✅ Docker Swarm con DNS
- ✅ Handshake y verificación SHA-1

#### Bugs Corregidos
- 🐛 Fix: Hostname vacío en tracker (strconv.Unquote issue)
- 🐛 Fix: Bencode panic con []map[string]interface{}

---

**Última actualización**: 26 de Octubre, 2025  
**Versión actual**: v2.0.0  
**Siguiente versión**: v2.1.0 (Weighted Round-Robin)
