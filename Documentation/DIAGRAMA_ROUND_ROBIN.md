# Diagrama de Flujo: Algoritmo Round-Robin

## 🔄 Secuencia de Descarga Paralela

```
┌─────────────────────────────────────────────────────────────────────┐
│                    INICIO: Peer Unchokes Client                     │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │ Picker.NextPieceFor()              │
        │ Selecciona pieza faltante          │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │ Manager.DownloadPieceParallel()    │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │ PASO 1: Filtrar Peers Elegibles    │
        │ • RemoteHasPiece(pieceIndex)       │
        │ • !PeerChoking                     │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │ availablePeers = [P1, P2, P3]      │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │ PASO 2: Calcular # de bloques      │
        │ numBlocks = pieceSize / 16KB       │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │ PASO 3: Crear PieceDownload        │
        │ blocksPending = {0,1,2,...,63}     │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │ PASO 4: Round-Robin Loop           │
        │ for blockNum in 0..63:             │
        │   peer = peers[blockNum % 3]       │
        │   SendBlockRequest(piece, offset)  │
        └────────────┬───────────────────────┘
                     │
                     ▼
┌────────────────────┴────────────────────┬────────────────────┐
│                                         │                    │
▼                                         ▼                    ▼
┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│ Peer1 recibe:    │    │ Peer2 recibe:    │    │ Peer3 recibe:    │
│ REQUEST bloque 0 │    │ REQUEST bloque 1 │    │ REQUEST bloque 2 │
│ REQUEST bloque 3 │    │ REQUEST bloque 4 │    │ REQUEST bloque 5 │
│ REQUEST bloque 6 │    │ REQUEST bloque 7 │    │ REQUEST bloque 8 │
│ ...              │    │ ...              │    │ ...              │
└────────┬─────────┘    └────────┬─────────┘    └────────┬─────────┘
         │                       │                       │
         │ Envía PIECE           │ Envía PIECE          │ Envía PIECE
         │                       │                       │
         └───────────────────────┴───────────────────────┘
                                 │
                                 ▼
                    ┌────────────────────────────┐
                    │ Client recibe MsgPiece     │
                    │ • WriteBlock(index,offset) │
                    │ • delete blocksPending[i]  │
                    └────────────┬───────────────┘
                                 │
                                 ▼
                    ┌────────────────────────────┐
                    │ ¿len(blocksPending) == 0?  │
                    └────────────┬───────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │                         │
                    ▼ SÍ                      ▼ NO
        ┌───────────────────────┐    ┌───────────────────┐
        │ Pieza completa        │    │ Esperar más       │
        │ • Verificar SHA-1     │    │ bloques           │
        │ • Broadcast HAVE      │    └───────────────────┘
        │ • Elegir siguiente    │
        └───────────────────────┘
```

## 📊 Distribución de Bloques (Ejemplo Real)

### Pieza 0 = 1 MB = 64 bloques de 16KB

```
┌─────────┬─────────┬─────────┬─────────┬─────────┬─────────┬─────────┐
│ Bloque  │    0    │    1    │    2    │    3    │    4    │   ...   │
├─────────┼─────────┼─────────┼─────────┼─────────┼─────────┼─────────┤
│ Peer    │  Peer1  │  Peer2  │  Peer3  │  Peer1  │  Peer2  │   ...   │
├─────────┼─────────┼─────────┼─────────┼─────────┼─────────┼─────────┤
│ Offset  │    0    │  16384  │  32768  │  49152  │  65536  │   ...   │
└─────────┴─────────┴─────────┴─────────┴─────────┴─────────┴─────────┘

Resultado:
┌──────────┬──────────────────┐
│ Peer1    │ 22 bloques       │  (bloques 0, 3, 6, 9, ...)
│ Peer2    │ 21 bloques       │  (bloques 1, 4, 7, 10, ...)
│ Peer3    │ 21 bloques       │  (bloques 2, 5, 8, 11, ...)
└──────────┴──────────────────┘
```

## ⚡ Timeline de Descarga

```
Tiempo   Peer1                Peer2                Peer3
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
0ms      REQUEST blk 0        REQUEST blk 1        REQUEST blk 2
10ms     PIECE blk 0 ←        
15ms                          PIECE blk 1 ←
20ms                                               PIECE blk 2 ←
20ms     REQUEST blk 3        REQUEST blk 4        REQUEST blk 5
30ms     PIECE blk 3 ←
35ms                          PIECE blk 4 ←
40ms                                               PIECE blk 5 ←
...
220ms    [Pieza completa - todos los bloques recibidos]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Comparación:
  Secuencial (1 peer):  640ms
  Round-Robin (3 peers): 220ms  → 3x más rápido
```

## 🔍 Estado del Manager Durante Descarga

### Estructura pieceDownloads

```go
// Estado inicial (después de enviar requests)
manager.pieceDownloads = {
  0: &PieceDownload{
    pieceIndex: 0,
    blocksPending: {
      0: true, 1: true, 2: true, ... 63: true  // 64 bloques pendientes
    },
    blocksInProgress: {
      0: *Peer1, 1: *Peer2, 2: *Peer3,
      3: *Peer1, 4: *Peer2, 5: *Peer3, ...
    }
  }
}

// Después de recibir bloque 0 de Peer1
manager.pieceDownloads = {
  0: &PieceDownload{
    pieceIndex: 0,
    blocksPending: {
      1: true, 2: true, ... 63: true  // 63 bloques pendientes
    },
    blocksInProgress: {
      1: *Peer2, 2: *Peer3, 3: *Peer1, ...  // bloque 0 eliminado
    }
  }
}

// Después de recibir TODOS los bloques
manager.pieceDownloads = {}  // Limpiado, pieza completa
```

## 📈 Throughput Comparison

```
┌──────────────┬───────────────┬────────────────┬──────────────┐
│  Algoritmo   │ Peers Usados  │   Throughput   │  Latencia    │
├──────────────┼───────────────┼────────────────┼──────────────┤
│ Secuencial   │      1        │   10 MB/s      │   640ms      │
│ Round-Robin  │      3        │   30 MB/s      │   220ms      │
│ Mejora       │     +200%     │     +200%      │    -65%      │
└──────────────┴───────────────┴────────────────┴──────────────┘
```

## 🎯 Decisión de Peers Elegibles

```
                    ┌─────────────────────┐
                    │   Todos los Peers   │
                    │    en el Swarm      │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌───────────────────────┐
                    │ Filtro 1:             │
                    │ RemoteHasPiece(0)?    │
                    └──────────┬────────────┘
                               │
                YES ┌──────────┴──────────┐ NO
                    │                     │
                    ▼                     ▼
         ┌──────────────────┐      ┌──────────┐
         │ Filtro 2:        │      │ EXCLUIR  │
         │ !PeerChoking?    │      └──────────┘
         └──────┬───────────┘
                │
   YES ┌────────┴────────┐ NO
       │                 │
       ▼                 ▼
┌────────────┐    ┌──────────┐
│ ELEGIBLE   │    │ EXCLUIR  │
│ availablePeers│ └──────────┘
└────────────┘

Ejemplo:
  Swarm: 5 peers
  Pieza 0: [1,1,1,0,0] bitfields
  
  Peer1: bit=1, !choking ✅ ELEGIBLE
  Peer2: bit=1, !choking ✅ ELEGIBLE
  Peer3: bit=1, choking  ❌ EXCLUIDO
  Peer4: bit=0, !choking ❌ EXCLUIDO (no tiene pieza)
  Peer5: bit=0, !choking ❌ EXCLUIDO (no tiene pieza)
  
  availablePeers = [Peer1, Peer2]
```

## 🔧 Sincronización y Thread-Safety

```
┌─────────────────────────────────────────────────────────────┐
│                    Manager (Thread-Safe)                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  mu (RWMutex)           ← protege peers map                  │
│  ├── Lock()   cuando se agrega/elimina peer                  │
│  └── RLock()  cuando se lee lista de peers                   │
│                                                              │
│  downloadsMu (Mutex)    ← protege pieceDownloads             │
│  ├── Lock()   cuando se modifica blocksPending               │
│  └── Unlock() después de modificar estado                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘

Flujo de acceso concurrente:
  Goroutine 1 (Peer1.ReadLoop) recibe bloque 0
    → downloadsMu.Lock()
    → delete(blocksPending[0])
    → downloadsMu.Unlock()
  
  Goroutine 2 (Peer2.ReadLoop) recibe bloque 1 SIMULTÁNEAMENTE
    → downloadsMu.Lock()  [espera a que Goroutine 1 termine]
    → delete(blocksPending[1])
    → downloadsMu.Unlock()
```

---

**Creado**: 26 de Octubre, 2025  
**Algoritmo**: Round-Robin Simple para Descarga Paralela  
**Complejidad**: O(n) donde n = número de bloques  
**Espacio**: O(n) para tracking de blocksPending
