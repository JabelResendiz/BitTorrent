# Sistema de Trackers Distribuidos

**Versión**: 3.0.0  
**Fecha**: Diciembre 2025

## Tabla de Contenidos

1. [Resumen Ejecutivo](#resumen-ejecutivo)
2. [Arquitectura del Sistema](#arquitectura-del-sistema)
3. [Componentes Principales](#componentes-principales)
4. [Sincronización y Consistencia](#sincronización-y-consistencia)
5. [Guía de Uso](#guía-de-uso)
6. [Ejemplos de Configuración](#ejemplos-de-configuración)
7. [Cambios Implementados](#cambios-implementados)

---

## Resumen Ejecutivo

El sistema de trackers distribuidos implementa una **arquitectura multi-líder asíncrona** que permite a múltiples trackers operar independientemente mientras mantienen consistencia eventual del estado de los swarms.

### Características principales:

- **Multi-líder**: Todos los trackers pueden aceptar announces de clientes
- **Asíncrono**: No hay coordinación síncrona entre trackers (sin bloqueos distribuidos)
- **Consistencia Eventual**: El estado converge usando gossip push periódico
- **Tolerancia a Particiones**: Los trackers pueden estar temporalmente desconectados
- **Sin Dependencia de NTP**: Usa Hybrid Logical Clocks (HLC) para ordenamiento causal
- **Tombstones con Resurrección**: Los peers eliminados se propagan correctamente

---

## Arquitectura del Sistema

### Diagrama de Alto Nivel

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│  Tracker 1  │◄───────►│  Tracker 2  │◄───────►│  Tracker 3  │
│  (node-1)   │  Sync   │  (node-2)   │  Sync   │  (node-3)   │
│  :8080      │  :9090  │  :8080      │  :9090  │  :8080      │
└──────▲──────┘         └──────▲──────┘         └──────▲──────┘
       │                       │                       │
       │ Announce              │ Announce              │ Announce
       │                       │                       │
   ┌───▼────┐              ┌───▼────┐              ┌───▼────┐
   │Client 1│              │Client 2│              │Client 3│
   └────────┘              └────────┘              └────────┘
```

### Flujo de Operaciones

1. **Announce del Cliente**: El cliente hace announce a cualquier tracker (ej: Tracker 1)
2. **Actualización Local**: Tracker 1 actualiza su estado local y su HLC
3. **Gossip Push Periódico**: Cada 15 segundos, Tracker 1 envía su estado completo a Tracker 2 y 3
4. **Merge Remoto**: Tracker 2 y 3 reciben el mensaje y hacen merge usando LWW (Last Write Wins)
5. **Convergencia**: Después de algunos ciclos de gossip, todos tienen el mismo estado

---

## Componentes Principales

### 1. Hybrid Logical Clock (HLC)

**Archivo**: `src/tracker/hlc.go`

HLC combina tiempo físico (del reloj del sistema) con un contador lógico para establecer orden causal sin relojes sincronizados.

#### Estructura

```go
type HLC struct {
    PhysicalTime int64  // milisegundos desde epoch
    LogicalTime  int64  // contador lógico
    NodeID       string // identificador del nodo
}
```

#### Funcionamiento

**Evento Local** (agregar un peer):
```
HLC anterior: {pt:1000, lt:2, node:"tracker-1"}
Evento local → {pt:1005, lt:3, node:"tracker-1"}
```

**Recibir Mensaje del "Futuro"**:
```
HLC local:  {pt:1000, lt:2, node:"tracker-1"}
Mensaje:    {pt:1010, lt:5, node:"tracker-2"}
Resultado:  {pt:1010, lt:6, node:"tracker-1"}  <- "salta" al futuro
```

**Recibir Mensaje del "Pasado"**:
```
HLC local:  {pt:1000, lt:2, node:"tracker-1"}
Mensaje:    {pt:990, lt:1, node:"tracker-2"}
Resultado:  {pt:1000, lt:3, node:"tracker-1"}  <- mantiene su tiempo
```

#### Comparación

Para saber qué evento ocurrió primero:

1. **Tiempo físico**: `pt1 > pt2` → más reciente
2. **Si iguales, contador lógico**: `lt1 > lt2` → más reciente  
3. **Si ambos iguales, node ID**: orden alfabético

### 2. Tombstones

**Archivos**: `src/tracker/tracker.go`, `src/tracker/sync_merge.go`

Los tombstones son marcadores que indican que un peer fue eliminado, permitiendo que la eliminación se propague en el sistema distribuido.

#### Estructura del Peer

```go
type Peer struct {
    PeerIDHex string
    IP        string
    Port      uint16
    LastSeen  HLC    // timestamp del último evento
    Completed bool
    HostName  string
    Deleted   bool   // tombstone flag
}
```

#### Ciclo de Vida de un Peer

```
1. Peer activo:      {Deleted: false, LastSeen: HLC{1000,1}}
2. Peer inactivo:    GC marca como tombstone → {Deleted: true, LastSeen: HLC{1020,5}}
3. Propagación:      Otros trackers reciben el tombstone
4. Resurrección:     Si llega announce más reciente → {Deleted: false, LastSeen: HLC{1030,2}}
5. Eliminación GC:   Tombstone viejo (>2×PeerTimeout) se elimina físicamente
```

#### Resurrección de Tombstones

Si un tracker tiene un peer eliminado (`Deleted=true`) pero recibe una actualización más reciente donde está activo:

```go
Local:  {Deleted: true,  LastSeen: HLC{1000,1}}
Remoto: {Deleted: false, LastSeen: HLC{1010,2}}
→ El peer se "resucita" porque la info remota es más nueva
```

### 3. Sistema de Sincronización

**Archivos**: 
- `src/tracker/sync.go` - Cliente y servidor de sincronización
- `src/tracker/sync_messages.go` - Mensajes de sincronización
- `src/tracker/sync_merge.go` - Lógica de merge

#### SyncManager (Cliente - Push)

Envía periódicamente el estado completo a todos los trackers remotos.

```go
type SyncManager struct {
    tracker      *Tracker
    remotePeers  []string      // direcciones de otros trackers
    syncInterval time.Duration // intervalo de sincronización (ej: 15s)
}
```

**Operación**:
1. Cada `syncInterval`, crea un `SyncMessage` con todo el estado
2. Envía POST a `http://tracker2:9090/sync`, `http://tracker3:9090/sync`, etc.
3. No espera respuesta (fire-and-forget)

#### SyncListener (Servidor - Receive)

Escucha en un puerto dedicado (ej: `:9090`) para recibir mensajes de otros trackers.

```go
type SyncListener struct {
    tracker  *Tracker
    listener net.Listener
}
```

**Operación**:
1. Recibe POST en `/sync`
2. Parsea el `SyncMessage`
3. Llama a `MergeSwarms()` para integrar el estado

#### Merge con LWW (Last Write Wins)

Cuando llega un peer remoto:

1. **Si no existe localmente**: agregar (incluso si es tombstone)
2. **Si existe localmente**: comparar `LastSeen` usando HLC
   - **Remoto más reciente**: actualizar local con datos remotos
   - **Local más reciente o igual**: ignorar remoto
3. **Resurrección**: Si local es tombstone pero remoto está activo y es más nuevo → resucitar

```go
if remotePeer.LastSeen.After(localPeer.LastSeen) {
    // Remoto es más reciente, actualizar
    if localPeer.Deleted && !remotePeer.Deleted {
        // ¡Resurrección!
        localPeer.Deleted = false
    }
    localPeer.LastSeen = remotePeer.LastSeen
    localPeer.IP = remotePeer.IP
    // ... copiar resto de campos
}
```

### 4. Garbage Collection

**Archivo**: `src/tracker/tracker.go`

El GC tiene dos fases:

**Fase 1: Marcar como Tombstone**
- Peers inactivos (>PeerTimeout) se marcan con `Deleted=true`
- Se actualiza `LastSeen` con el HLC actual
- El tombstone se propaga a otros trackers

**Fase 2: Eliminación Física**
- Tombstones muy antiguos (>2×PeerTimeout) se eliminan físicamente
- Esto evita que los tombstones crezcan indefinidamente

```go
func (t *Tracker) GC() int {
    t.hlc.Update(nil)
    thresholdInactive := t.hlc.SubtractDuration(t.PeerTimeout)
    thresholdTombstone := t.hlc.SubtractDuration(2 * t.PeerTimeout)
    
    for _, swarm := range t.Torrents {
        for id, peer := range swarm.Peers {
            if peer.Deleted {
                // Fase 2: eliminar tombstone antiguo
                if thresholdTombstone.After(peer.LastSeen) {
                    delete(swarm.Peers, id)
                }
            } else {
                // Fase 1: crear tombstone
                if thresholdInactive.After(peer.LastSeen) {
                    peer.Deleted = true
                    peer.LastSeen = t.hlc.Clone()
                }
            }
        }
    }
}
```

---

## Sincronización y Consistencia

### Propiedades del Sistema

#### Consistencia Eventual

El sistema NO es fuertemente consistente (no hay linearizabilidad). Características:

- **Eventual Consistency**: Después de que cesan las actualizaciones, todos los trackers convergen al mismo estado
- **Monotonic Reads**: Un tracker nunca "retrocede" en el tiempo (gracias a HLC)
- **Read Your Writes**: Un cliente que hace announce verá su peer inmediatamente en ese tracker

#### Resolución de Conflictos

El sistema usa **LWW (Last Write Wins)** basado en HLC:

```
Tracker 1 recibe: peer A con LastSeen={1000,1}
Tracker 2 recibe: peer A con LastSeen={1010,2}

Después del gossip:
→ Ambos tendrán peer A con LastSeen={1010,2} (el más reciente)
```

#### Casos Especiales

**Partición de Red**:
```
Tiempo 0: Tracker 1 y 2 desconectados
Tiempo 1: Cliente X hace announce a Tracker 1 → peer agregado localmente
Tiempo 2: Cliente Y hace announce a Tracker 2 → peer agregado localmente
Tiempo 3: Red se reconecta
Tiempo 4: Gossip → ambos trackers tienen ambos peers
```

**Eliminación Concurrente**:
```
Tracker 1: GC marca peer X como deleted → LastSeen={1000,5}
Tracker 2: Recibe announce de peer X → LastSeen={1010,1}

Gossip:
→ Tracker 2 gana (tiempo más reciente), peer X resucita
```

### Latencia de Convergencia

Con `sync-interval=15s` y 3 trackers:

- **Caso ideal**: Un evento se propaga en 15 segundos
- **Caso peor**: Con particiones, hasta que se reconecte + 15 segundos
- **Cascada**: Tracker 1 → 2 (15s) → 3 (si 3 solo habla con 2, 30s total)

---

## Guía de Uso

### Flags del Tracker

```bash
./tracker [flags]
```

| Flag | Default | Descripción |
|------|---------|-------------|
| `-listen` | `:8080` | Puerto HTTP para announces de clientes |
| `-interval` | `1800` | Intervalo de announce en segundos |
| `-data` | `tracker_data.json` | Archivo de persistencia |
| `-maxpeers` | `50` | Máximo de peers por respuesta |
| `-node-id` | `""` | **Requerido** si hay `-sync-peers`. ID único del tracker |
| `-sync-listen` | `:9090` | Puerto para sincronización entre trackers |
| `-sync-peers` | `""` | Lista de trackers separados por coma (ej: `tracker2:9090,tracker3:9090`) |
| `-sync-interval` | `15` | Intervalo de sincronización en segundos |

### Modos de Operación

#### Modo Standalone (Un Solo Tracker)

```bash
./tracker -listen :8080 -data tracker1.json
```

- No requiere `-node-id` ni `-sync-peers`
- Funciona como tracker centralizado tradicional

#### Modo Distribuido (Múltiples Trackers)

**Tracker 1**:
```bash
./tracker \
  -listen :8080 \
  -sync-listen :9090 \
  -node-id tracker-1 \
  -sync-peers "tracker2:9090,tracker3:9090" \
  -data tracker1.json
```

**Tracker 2**:
```bash
./tracker \
  -listen :8080 \
  -sync-listen :9090 \
  -node-id tracker-2 \
  -sync-peers "tracker1:9090,tracker3:9090" \
  -data tracker2.json
```

**Tracker 3**:
```bash
./tracker \
  -listen :8080 \
  -sync-listen :9090 \
  -node-id tracker-3 \
  -sync-peers "tracker1:9090,tracker2:9090" \
  -data tracker3.json
```

### Configuración de Clientes

Los clientes pueden usar **cualquier tracker** en el announce URL del .torrent:

**Opción 1: Un solo tracker** (otros se sincronizan)
```
announce=http://tracker1:8080/announce
```

**Opción 2: Lista de trackers** (más resiliencia)
```
announce-list=[
  ["http://tracker1:8080/announce"],
  ["http://tracker2:8080/announce"],
  ["http://tracker3:8080/announce"]
]
```

---

## Ejemplos de Configuración

### Docker Compose con 3 Trackers

```yaml
version: '3.8'

services:
  tracker1:
    build: ./src/tracker
    container_name: tracker1
    hostname: tracker1
    command: >
      /app/tracker
      -listen :8080
      -sync-listen :9090
      -node-id tracker-1
      -sync-peers "tracker2:9090,tracker3:9090"
      -sync-interval 15
      -data /data/tracker1.json
    ports:
      - "8081:8080"
      - "9091:9090"
    volumes:
      - ./data/tracker1:/data
    networks:
      - bittorrent

  tracker2:
    build: ./src/tracker
    container_name: tracker2
    hostname: tracker2
    command: >
      /app/tracker
      -listen :8080
      -sync-listen :9090
      -node-id tracker-2
      -sync-peers "tracker1:9090,tracker3:9090"
      -sync-interval 15
      -data /data/tracker2.json
    ports:
      - "8082:8080"
      - "9092:9090"
    volumes:
      - ./data/tracker2:/data
    networks:
      - bittorrent

  tracker3:
    build: ./src/tracker
    container_name: tracker3
    hostname: tracker3
    command: >
      /app/tracker
      -listen :8080
      -sync-listen :9090
      -node-id tracker-3
      -sync-peers "tracker1:9090,tracker2:9090"
      -sync-interval 15
      -data /data/tracker3.json
    ports:
      - "8083:8080"
      - "9093:9090"
    volumes:
      - ./data/tracker3:/data
    networks:
      - bittorrent

networks:
  bittorrent:
    driver: bridge
```

### Topología en Múltiples Hosts

**Host 1 (192.168.1.10)**:
```bash
./tracker \
  -listen :8080 \
  -sync-listen :9090 \
  -node-id tracker-host1 \
  -sync-peers "192.168.1.11:9090,192.168.1.12:9090"
```

**Host 2 (192.168.1.11)**:
```bash
./tracker \
  -listen :8080 \
  -sync-listen :9090 \
  -node-id tracker-host2 \
  -sync-peers "192.168.1.10:9090,192.168.1.12:9090"
```

**Host 3 (192.168.1.12)**:
```bash
./tracker \
  -listen :8080 \
  -sync-listen :9090 \
  -node-id tracker-host3 \
  -sync-peers "192.168.1.10:9090,192.168.1.11:9090"
```

---

## Cambios Implementados

### Archivos Nuevos

1. **`src/tracker/hlc.go`** (210 líneas)
   - Implementación completa de Hybrid Logical Clock
   - Métodos: `NewHLC()`, `Update()`, `After()`, `Before()`, `Clone()`, `SubtractDuration()`
   - Serialización JSON

2. **`src/tracker/sync.go`** (170 líneas)
   - `SyncManager`: cliente de sincronización periódica (push)
   - `SyncListener`: servidor de sincronización (receive)
   - Comunicación HTTP/JSON entre trackers

3. **`src/tracker/sync_messages.go`** (45 líneas)
   - Definición de `SyncMessage`
   - Serialización del estado completo del tracker

4. **`src/tracker/sync_merge.go`** (90 líneas)
   - Lógica de merge con LWW
   - Resurrección de tombstones
   - Logs detallados de operaciones de merge

### Archivos Modificados

1. **`src/tracker/tracker.go`**
   - **Estructura `Peer`**: 
     - `LastSeen`: `time.Time` → `HLC`
     - Nuevo campo: `Deleted bool` (tombstone)
   
   - **Estructura `Tracker`**:
     - Nuevos campos: `hlc`, `nodeID`, `remotePeers`, `syncListener`, `syncManager`
   
   - **Función `New()`**: 
     - Nuevos parámetros: `nodeID`, `remotePeers`
     - Inicializa HLC
   
   - **Función `AddPeer()`**:
     - Removido parámetro `now time.Time`
     - Usa `t.hlc.Update(nil)` internamente
     - Resucita peers eliminados
   
   - **Función `RemovePeer()`**:
     - Ya no elimina físicamente
     - Marca como tombstone (`Deleted=true`)
   
   - **Función `GC()`**:
     - Removido parámetro `now time.Time`
     - Usa umbrales basados en HLC
     - Fase 1: Crea tombstones
     - Fase 2: Elimina tombstones antiguos
   
   - **Función `GetPeers()`**:
     - Filtra peers con `Deleted=true`
   
   - **Función `CountPeers()`**:
     - Excluye peers eliminados
   
   - **Nuevas funciones**:
     - `StartSyncListener()`: Inicia servidor de sync
     - `StartSyncManager()`: Inicia cliente de sync
     - `StopSync()`: Detiene sincronización

2. **`src/tracker/announce.go`**
   - Removido import `time`
   - Removida variable `now := time.Now()`
   - Todas las llamadas a `AddPeer()` sin parámetro `now`

3. **`src/tracker/persist.go`**
   - Removido import `time`
   - `LoadFromFile()`: Usa HLC para filtrar peers antiguos
   - Cálculo de threshold con `t.hlc.SubtractDuration()`

4. **`src/tracker/cmd/main.go`**
   - **Nuevos flags**: 
     - `-node-id`
     - `-sync-listen`
     - `-sync-peers`
     - `-sync-interval`
   
   - **Parseo de peers remotos**: Split por comas
   
   - **Validación**: Require `node-id` si hay `sync-peers`
   
   - **Inicialización**:
     - Llama a `t.StartSyncListener()` si hay peers remotos
     - Llama a `t.StartSyncManager()` si hay peers remotos
   
   - **GC loop**: Removido argumento `time.Now()` de `t.GC()`

### Estadísticas de Cambios

- **Archivos nuevos**: 4
- **Archivos modificados**: 5
- **Líneas añadidas**: ~800
- **Líneas modificadas**: ~100

### Compatibilidad

**Breaking Changes**:
- La función `New()` ahora requiere 2 parámetros adicionales
- La función `AddPeer()` tiene una firma diferente
- La función `GC()` no recibe parámetro `now`
- El formato JSON de persistencia cambió (`LastSeen` es ahora un objeto HLC)

**Migración desde versión anterior**:
Los datos antiguos con `time.Time` NO son compatibles. Opciones:
1. Borrar `tracker_data.json` y empezar limpio
2. Script de migración (convertir `time.Time` a HLC)

---

## Debugging y Monitoring

### Logs de Sincronización

Los logs incluyen prefijo `[SYNC]` para operaciones de sincronización:

```
[SYNC] Starting sync manager with 2 remote peers, interval=15s
[SYNC] Sync listener started on :9090
[SYNC] Pushing state to 2 peers (swarms=5)
[SYNC] Successfully pushed to tracker2:9090
[SYNC] Received sync from node tracker-2 with 5 swarms
[SYNC] Merging swarms from node tracker-2
[SYNC] Added new peer abc123 to swarm 1a2b3c4d (deleted=false)
[SYNC] Updated peer def456 in swarm 5e6f7g8h
[SYNC] Resurrected peer xyz789 in swarm 9i0j1k2l
[SYNC] Ignored older update for peer mno345 in swarm 3m4n5o6p
```

### Verificar Estado

**Ver peers en un tracker**:
```bash
curl http://tracker1:8080/scrape?info_hash=<hash>
```

**Ver diferencias entre trackers**:
```bash
# Obtener scrape de ambos
diff <(curl -s http://tracker1:8080/scrape?info_hash=<hash>) \
     <(curl -s http://tracker2:8080/scrape?info_hash=<hash>)
```

**Monitorear sincronización**:
```bash
# Ver logs en tiempo real
docker logs -f tracker1 | grep SYNC
```

### Problemas Comunes

**Los trackers no se sincronizan**:
- Verificar conectividad de red entre puertos de sync (`:9090`)
- Verificar que los hostnames sean resolubles
- Revisar logs para errores de conexión

**Peers duplicados o inconsistentes**:
- Esperar algunos ciclos de sync (3-4 × `sync-interval`)
- Verificar que los `node-id` sean únicos
- Revisar que no haya particiones de red

**Relojes muy desincronizados**:
- HLC debería manejar diferencias de hasta minutos
- Si hay diferencias de horas, el sistema puede tener comportamiento extraño
- Considerar sincronizar relojes aproximadamente (±1 minuto es OK)

---

## Conclusión

El sistema de trackers distribuidos implementa un diseño robusto y escalable que permite:

✅ Múltiples trackers operando simultáneamente  
✅ Alta disponibilidad (si un tracker cae, otros continúan)  
✅ Consistencia eventual sin coordinación centralizada  
✅ Tolerancia a relojes desincronizados  
✅ Propagación correcta de eliminaciones  

El sistema es apropiado para despliegues donde:
- Se necesita redundancia de trackers
- Los clientes pueden estar en diferentes regiones
- La consistencia inmediata no es crítica
- Se prefiere disponibilidad sobre consistencia fuerte

---

**Documentación generada por el sistema de trackers distribuidos v3.0.0**
