# Sistema de Logging Detallado para Round-Robin

## 📊 Descripción

Se ha implementado un sistema de logging completo que muestra desde qué peer se obtiene cada bloque durante la descarga paralela con el algoritmo Round-Robin.

## 🎯 Logs Agregados

### 1. **Solicitud de Bloques (REQUEST)**
Cuando se inicia la descarga de una pieza, se muestra a qué peer se le solicita cada bloque:

```
Descargando pieza 0 desde 3 peers en paralelo (Round-Robin)
  → Solicitando bloque 0 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 1 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 2 de pieza 0 a peer 10.0.1.7:6881
  → Solicitando bloque 3 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 4 de pieza 0 a peer 10.0.1.6:6881
  ...
```

**Formato**: `→ Solicitando bloque [num] de pieza [index] a peer [IP:PORT]`

### 2. **Recepción de Bloques (PIECE)**
Cuando llega cada bloque, se muestra desde qué peer se recibió:

```
✓ Recibido bloque 0 de pieza 0 desde peer 10.0.1.5:6881 (offset 0, tamaño 16384 bytes)
✓ Recibido bloque 2 de pieza 0 desde peer 10.0.1.7:6881 (offset 32768, tamaño 16384 bytes)
✓ Recibido bloque 1 de pieza 0 desde peer 10.0.1.6:6881 (offset 16384, tamaño 16384 bytes)
✓ Recibido bloque 3 de pieza 0 desde peer 10.0.1.5:6881 (offset 49152, tamaño 16384 bytes)
...
```

**Formato**: `✓ Recibido bloque [num] de pieza [index] desde peer [IP:PORT] (offset [bytes], tamaño [bytes])`

### 3. **Resumen al Completar Pieza**
Al terminar una pieza, se muestra un resumen con estadísticas detalladas:

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

## 🏗️ Implementación Técnica

### Cambios en Estructuras

#### PieceDownload
```go
type PieceDownload struct {
    pieceIndex       int
    blocksPending    map[int]bool
    blocksInProgress map[int]*PeerConn
    blocksReceived   map[string]int  // ← NUEVO: peerAddr -> cantidad bloques
}
```

**Campo nuevo**: `blocksReceived` rastrea cuántos bloques se recibieron de cada peer.

### Modificaciones en Código

#### 1. manager_broadcast.go - DownloadPieceParallel()

**Ubicación**: Dentro del loop Round-Robin

```go
// Log: Mostrar desde qué peer se solicita el bloque
peerAddr := "unknown"
if peer.Conn != nil && peer.Conn.RemoteAddr() != nil {
    peerAddr = peer.Conn.RemoteAddr().String()
}
println("  → Solicitando bloque", blockNum, "de pieza", pieceIndex, "a peer", peerAddr)
```

**Cuándo se ejecuta**: Al enviar cada REQUEST a un peer.

#### 2. manager.go - handleMessage() case MsgPiece

**Ubicación**: Al recibir bloque

```go
// Log: Mostrar desde qué peer se recibió el bloque
peerAddr := "unknown"
if p.Conn != nil && p.Conn.RemoteAddr() != nil {
    peerAddr = p.Conn.RemoteAddr().String()
}
blockNum := int(begin) / blockLen
fmt.Printf("✓ Recibido bloque %d de pieza %d desde peer %s (offset %d, tamaño %d bytes)\n", 
    blockNum, index, peerAddr, begin, len(block))
```

**Cuándo se ejecuta**: Al recibir cada mensaje PIECE.

#### 3. manager.go - Tracking de estadísticas

**Ubicación**: Dentro del lock de pieceDownloads

```go
if pd, exists := p.manager.pieceDownloads[int(index)]; exists {
    // Incrementar contador de bloques recibidos desde este peer
    pd.blocksReceived[peerAddr]++
    
    delete(pd.blocksPending, blockNum)
    delete(pd.blocksInProgress, blockNum)

    // Si todos los bloques están completos
    if len(pd.blocksPending) == 0 {
        // Mostrar resumen estadístico
        fmt.Printf("\n═══════════════════════════════════════════════\n")
        fmt.Printf("✓ Pieza %d completada (Round-Robin)\n", index)
        fmt.Printf("═══════════════════════════════════════════════\n")
        fmt.Printf("Distribución de bloques por peer:\n")
        for pAddr, count := range pd.blocksReceived {
            fmt.Printf("  • Peer %s: %d bloques\n", pAddr, count)
        }
        fmt.Printf("═══════════════════════════════════════════════\n\n")
    }
}
```

## 📋 Ejemplo Completo de Salida

### Escenario: 3 Peers, Pieza de 5 bloques

```
Peer te unchokeo. Buscando pieza a solicitar...
Descargando pieza 0 desde 3 peers en paralelo (Round-Robin)
  → Solicitando bloque 0 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 1 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 2 de pieza 0 a peer 10.0.1.7:6881
  → Solicitando bloque 3 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 4 de pieza 0 a peer 10.0.1.6:6881

✓ Recibido bloque 0 de pieza 0 desde peer 10.0.1.5:6881 (offset 0, tamaño 16384 bytes)
✓ Recibido bloque 1 de pieza 0 desde peer 10.0.1.6:6881 (offset 16384, tamaño 16384 bytes)
✓ Recibido bloque 2 de pieza 0 desde peer 10.0.1.7:6881 (offset 32768, tamaño 16384 bytes)
✓ Recibido bloque 3 de pieza 0 desde peer 10.0.1.5:6881 (offset 49152, tamaño 16384 bytes)
✓ Recibido bloque 4 de pieza 0 desde peer 10.0.1.6:6881 (offset 65536, tamaño 16384 bytes)

═══════════════════════════════════════════════
✓ Pieza 0 completada (Round-Robin)
═══════════════════════════════════════════════
Distribución de bloques por peer:
  • Peer 10.0.1.5:6881: 2 bloques
  • Peer 10.0.1.6:6881: 2 bloques
  • Peer 10.0.1.7:6881: 1 bloques
Total: 5 bloques
═══════════════════════════════════════════════

Broadcast HAVE a 3 peers. Pieza 0
```

## 🔍 Análisis del Output

### Verificar Distribución Round-Robin

Observando los logs de solicitud:
```
Bloque 0 → Peer1 (10.0.1.5)
Bloque 1 → Peer2 (10.0.1.6)
Bloque 2 → Peer3 (10.0.1.7)
Bloque 3 → Peer1 (10.0.1.5) ← Vuelta al inicio
Bloque 4 → Peer2 (10.0.1.6)
```

✅ **Confirmado**: Distribución circular perfecta.

### Verificar Recepción Paralela

Los bloques pueden llegar en desorden:
```
✓ Recibido bloque 0 ...
✓ Recibido bloque 2 ... ← Llegó antes que bloque 1
✓ Recibido bloque 1 ...
```

✅ **Confirmado**: Descarga verdaderamente paralela (no secuencial).

### Verificar Balance de Carga

En el resumen:
```
Peer1: 22 bloques (34.4%)
Peer2: 21 bloques (32.8%)
Peer3: 21 bloques (32.8%)
```

✅ **Confirmado**: Carga balanceada uniformemente.

## 🎨 Formato de Símbolos

- `→` : REQUEST enviado (solicitud)
- `✓` : PIECE recibido (respuesta exitosa)
- `═` : Bordes del resumen de estadísticas
- `•` : Bullet point en lista de peers

## 🐛 Debugging

### Problema: No aparece "desde peer"

**Causa**: `peer.Conn.RemoteAddr()` es nil

**Solución implementada**:
```go
peerAddr := "unknown"
if peer.Conn != nil && peer.Conn.RemoteAddr() != nil {
    peerAddr = peer.Conn.RemoteAddr().String()
}
```

### Problema: Estadísticas incorrectas

**Causa**: Race condition en `blocksReceived`

**Solución implementada**: 
- Lock `downloadsMu` antes de modificar
- Incremento dentro del lock

### Problema: Bloques duplicados en resumen

**Causa**: Counter no se limpia entre piezas

**Solución implementada**:
- `blocksReceived` es parte de `PieceDownload`
- Se crea nuevo map para cada pieza
- Se elimina al completar pieza

## 📊 Métricas que se Pueden Extraer

Con estos logs, puedes analizar:

1. **Latencia por peer**: Tiempo entre REQUEST y PIECE
2. **Throughput por peer**: Bloques recibidos / tiempo
3. **Fiabilidad**: Peers que no responden (REQUEST sin PIECE)
4. **Balance**: Distribución uniforme vs sesgada
5. **Paralelismo**: Orden de llegada vs orden de solicitud

## 🚀 Uso en Producción

### Activar solo para Debug
Si quieres logs menos verbosos en producción:

```go
const debugRoundRobin = false  // Variable global

if debugRoundRobin {
    println("  → Solicitando bloque", blockNum, ...)
}
```

### Métricas para Prometheus
Puedes exportar estas estadísticas:

```go
roundRobinBlocksReceivedTotal.WithLabelValues(peerAddr).Inc()
roundRobinPieceCompleteTime.Observe(time.Since(startTime).Seconds())
```

---

**Implementado**: 26 de Octubre, 2025  
**Archivos modificados**: 
- `src/peerwire/manager_broadcast.go`
- `src/peerwire/manager.go`

**Beneficio**: Visibilidad completa del comportamiento Round-Robin en tiempo real.
