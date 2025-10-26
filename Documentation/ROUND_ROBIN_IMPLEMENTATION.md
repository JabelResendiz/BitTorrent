# Implementación del Algoritmo Round-Robin para Descarga Paralela

## 📋 Descripción General

Se ha implementado el algoritmo **Round-Robin** para coordinar la descarga de bloques desde múltiples peers simultáneamente. Esto mejora significativamente el throughput al distribuir la carga entre todos los peers disponibles que tienen una pieza específica.

## 🔄 Algoritmo Round-Robin

### Concepto
En lugar de descargar TODOS los bloques de una pieza desde un solo peer (secuencial), el algoritmo distribuye los bloques de manera circular entre múltiples peers.

### Ejemplo Visual
```
Pieza 0 = 64 bloques (1MB / 16KB)
Peers disponibles: Peer1, Peer2, Peer3

Distribución Round-Robin:
- Bloque 0  → Peer1
- Bloque 1  → Peer2
- Bloque 2  → Peer3
- Bloque 3  → Peer1 (vuelta al inicio)
- Bloque 4  → Peer2
- Bloque 5  → Peer3
...
- Bloque 63 → Peer1
```

## 🏗️ Arquitectura de la Implementación

### 1. Estructura PieceDownload
```go
type PieceDownload struct {
    pieceIndex       int
    blocksPending    map[int]bool      // bloques que faltan descargar
    blocksInProgress map[int]*PeerConn // bloque -> peer descargando
}
```

**Propósito**: Rastrear el estado de descarga de cada pieza cuando múltiples peers están colaborando.

### 2. Manager Actualizado
```go
type Manager struct {
    mu              sync.RWMutex
    peers           map[*PeerConn]struct{}
    store           PieceStore
    pieceDownloads  map[int]*PieceDownload // pieceIndex -> estado
    downloadsMu     sync.Mutex              // protege pieceDownloads
}
```

**Nuevos campos**:
- `pieceDownloads`: Tracking de qué bloques están descargándose
- `downloadsMu`: Mutex para sincronizar acceso concurrente

### 3. Método Principal: DownloadPieceParallel()

#### Flujo del Algoritmo

```go
func (m *Manager) DownloadPieceParallel(pieceIndex int) {
    // PASO 1: Filtrar peers elegibles
    for peer := range m.peers {
        if peer.RemoteHasPiece(pieceIndex) && !peer.PeerChoking {
            availablePeers = append(availablePeers, peer)
        }
    }
    
    // PASO 2: Crear tracking de bloques
    pd := &PieceDownload{...}
    for i := 0; i < numBlocks; i++ {
        pd.blocksPending[i] = true
    }
    
    // PASO 3: Distribuir bloques en Round-Robin
    for blockNum := 0; blockNum < numBlocks; blockNum++ {
        peer := availablePeers[peerIndex % len(availablePeers)]
        peer.SendBlockRequest(pieceIndex, offset, size)
        peerIndex++
    }
}
```

#### Criterios de Filtrado
Un peer es elegible si:
1. ✅ **Tiene la pieza**: `peer.RemoteHasPiece(pieceIndex) == true`
2. ✅ **No está choke**: `peer.PeerChoking == false`
3. ✅ **Está conectado**: Existe en `manager.peers`

## 🔄 Flujo Completo de Descarga

### 1. Inicio: MsgUnchoke
```
Peer unchokea → 
  Picker selecciona pieza → 
    Manager.DownloadPieceParallel(pieceIndex) → 
      Distribuye bloques Round-Robin
```

### 2. Recepción: MsgPiece
```
Bloque recibido →
  WriteBlock(index, offset, data) →
    Marcar bloque como completo →
      ¿Todos bloques recibidos? →
        SÍ → Verificar SHA-1 → Elegir siguiente pieza
        NO → Esperar más bloques
```

### 3. Tracking de Bloques
```go
// Al recibir bloque
blockNum := int(begin) / blockLen
delete(pd.blocksPending, blockNum)      // Ya no pendiente
delete(pd.blocksInProgress, blockNum)   // Ya no en progreso

// ¿Pieza completa?
if len(pd.blocksPending) == 0 {
    delete(m.pieceDownloads, pieceIndex) // Limpiar tracking
}
```

## 📊 Comparación: Antes vs Después

### ❌ Implementación Anterior (Secuencial)
```
Peer1 → Descarga TODA la Pieza 0 (64 bloques secuenciales)
  Tiempo: 64 * latencia = 640ms @ 10ms latency
  Throughput: 1 MB / 640ms = 1.56 MB/s

Peer2 → Descarga TODA la Pieza 1 (64 bloques secuenciales)
Peer3 → Idle (no se usa hasta que se necesita otra pieza)
```

### ✅ Implementación Actual (Round-Robin)
```
Peer1 → Bloques 0, 3, 6, 9...  (22 bloques)
Peer2 → Bloques 1, 4, 7, 10... (21 bloques)
Peer3 → Bloques 2, 5, 8, 11... (21 bloques)

Tiempo: 22 * latencia = 220ms @ 10ms latency
Throughput: 1 MB / 220ms = 4.54 MB/s (3x mejora)
```

## 🎯 Ventajas del Round-Robin

### ✅ Pros
1. **Simplicidad**: Solo un contador circular
2. **Sin estado complejo**: No necesita rastrear velocidades
3. **Balanceo automático**: Carga distribuida uniformemente
4. **Paralelización real**: Múltiples peers trabajando simultáneamente
5. **Tolerancia a fallos**: Si un peer falla, otros continúan

### ⚠️ Limitaciones
1. **No considera velocidad**: Peer lento puede retrasar pieza completa
2. **Sin priorización**: Todos los bloques tienen igual prioridad
3. **Asume peers homogéneos**: Tratamiento igual sin medir performance

## 🔧 Archivos Modificados

### 1. `manager_broadcast.go`
- ➕ Agregada estructura `PieceDownload`
- ➕ Campos `pieceDownloads` y `downloadsMu` en `Manager`
- ➕ Método `calculateNumBlocks()`
- ➕ Método `DownloadPieceParallel()`

### 2. `manager.go`
- 🔄 Modificado `case MsgUnchoke`: Usa `DownloadPieceParallel()` en lugar de `requestNextBlocks()`
- 🔄 Modificado `case MsgPiece`: Tracking de bloques Round-Robin en lugar de pedir siguiente bloque secuencial

### 3. `message.go`
- ➕ Agregado método `SendBlockRequest(index, begin, length)`

## 🧪 Prueba de la Implementación

### Escenario de Prueba
```bash
# Terminal 1: Levantar tracker
docker service create --name tracker ...

# Terminal 2: Seeder (tiene archivo completo)
docker run --name seeder --network net \
  -v ~/archives:/app/src/archives \
  client --torrent=vid.torrent --archives=/app/src/archives --hostname=seeder

# Terminal 3: Leecher1 (descarga desde seeder)
docker run --name leecher1 --network net \
  -v ~/leecher1:/app/src/archives \
  client --torrent=vid.torrent --archives=/app/src/archives --hostname=leecher1

# Terminal 4: Leecher2 (descarga desde seeder Y leecher1 en paralelo)
docker run --name leecher2 --network net \
  -v ~/leecher2:/app/src/archives \
  client --torrent=vid.torrent --archives=/app/src/archives --hostname=leecher2
```

### Salida Esperada
```
Peer te unchokeo. Buscando pieza a solicitar...
Descargando pieza 0 desde 2 peers en paralelo (Round-Robin)
Recibido block de pieza 0, offset 0, tamaño 16384 bytes
Recibido block de pieza 0, offset 16384, tamaño 16384 bytes
Recibido block de pieza 0, offset 32768, tamaño 16384 bytes
...
Pieza 0 completa (Round-Robin)
Broadcast HAVE a 2 peers. Pieza 0
```

## 📈 Métricas de Rendimiento

### Throughput Esperado
```
Configuración:
- 3 peers
- 10 MB/s por peer
- Latencia: 10ms
- Pieza: 1 MB (64 bloques)

Secuencial: 1 peer * 10 MB/s = 10 MB/s
Round-Robin: 3 peers * 10 MB/s = 30 MB/s (teórico)
              Real: ~25 MB/s (considerando overhead)
```

### Reducción de Latencia
```
Secuencial: 64 requests * 10ms = 640ms
Round-Robin: 22 requests * 10ms = 220ms (3x mejora)
```

## 🚀 Próximas Optimizaciones

### 1. Pipeline de Requests
Enviar múltiples REQUEST sin esperar respuesta (ventana deslizante)

### 2. Weighted Round-Robin
Considerar velocidad de peers: peers rápidos reciben más bloques

### 3. Rarest-First Picker
Priorizar piezas menos comunes en el swarm

### 4. Endgame Mode
Al final, pedir bloques pendientes a TODOS los peers

## 📝 Notas de Implementación

- **Thread-Safety**: `downloadsMu` protege acceso concurrente a `pieceDownloads`
- **Cleanup**: Tracking se elimina cuando `len(blocksPending) == 0`
- **Backward Compatible**: `requestNextBlocks()` sigue existiendo para casos edge
- **Verificación SHA-1**: Storage verifica hash automáticamente al completar pieza

---

**Implementado**: 26 de Octubre, 2025  
**Algoritmo**: Round-Robin Simple (Phase 1)  
**Próximo**: Weighted Round-Robin con medición de throughput
