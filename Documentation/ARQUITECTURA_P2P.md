# 📚 Arquitectura de Comunicación P2P - BitTorrent

## 🎯 Visión General

Tu implementación BitTorrent sigue el protocolo estándar de peer-to-peer con una arquitectura modular que separa responsabilidades en componentes especializados.

---

## 🏗️ Arquitectura de Componentes

```
┌─────────────────────────────────────────────────────────────┐
│                    CLIENTE BITTORRENT                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐     ┌──────────────┐     ┌─────────────┐ │
│  │   Tracker    │────▶│   Manager    │────▶│  Storage    │ │
│  │ Communicate  │     │  (Coordina)  │     │ (Disk I/O)  │ │
│  └──────────────┘     └──────┬───────┘     └─────────────┘ │
│         │                    │                               │
│         │                    ▼                               │
│         │          ┌───────────────────┐                    │
│         │          │   PeerConn Pool   │                    │
│         │          │ (N conexiones)    │                    │
│         │          └─────────┬─────────┘                    │
│         │                    │                               │
│         ▼                    ▼                               │
│  ┌─────────────────────────────────────┐                   │
│  │     TCP Connections (Peer Wire)      │                   │
│  │  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐│                   │
│  │  │Peer1│  │Peer2│  │Peer3│  │PeerN││                   │
│  │  └─────┘  └─────┘  └─────┘  └─────┘│                   │
│  └─────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

---

## 📦 Módulos y Responsabilidades

### 1. **`conn.go` - Gestión de Conexiones TCP**

**Responsabilidad:** Crear y mantener conexiones TCP con peers

**Funciones clave:**
```go
NewPeerConn(addr, infoHash, peerId) *PeerConn
  ↓ Establece conexión TCP con timeout de 5 segundos
  ↓ Inicializa estado del peer (choking=true, interested=false)
  
NewPeerConnFromConn(conn, infoHash, peerId) *PeerConn
  ↓ Envuelve conexión entrante (accept from listener)
  ↓ Reutiliza lógica de PeerConn para ambas direcciones
```

**Estado del PeerConn:**
- `AmChoking`: Si estamos bloqueando al peer (no le enviamos piezas)
- `AmInterested`: Si nos interesan sus piezas
- `PeerChoking`: Si el peer nos bloquea
- `PeerInterested`: Si al peer le interesan nuestras piezas
- `remoteBF`: Bitfield del peer (qué piezas tiene)
- `curPiece`: Pieza actual en descarga (-1 si ninguna)
- `curOffset`: Offset dentro de la pieza actual

---

### 2. **`handshake.go` - Protocolo de Handshake**

**Responsabilidad:** Validar identidad y compatibilidad entre peers

**Formato del Handshake (68 bytes):**
```
┌────┬──────────────────────┬────────┬────────────┬────────────┐
│ 19 │ BitTorrent protocol  │ 8 null │  InfoHash  │  PeerId    │
│ 1B │        19B           │   8B   │    20B     │    20B     │
└────┴──────────────────────┴────────┴────────────┴────────────┘
```

**Flujo:**
```go
1. Handshake() - Conexión saliente:
   ├─ Enviar nuestro handshake
   ├─ Leer respuesta del peer
   ├─ Validar pstr = "BitTorrent protocol"
   └─ Validar que info_hash coincida (mismo torrent)

2. SendHandshakeOnly() - Conexión entrante:
   ├─ Ya leímos el handshake del peer (en main.go)
   ├─ Solo enviamos nuestro handshake de respuesta
   └─ El peer validará nuestro info_hash
```

**Validaciones:**
- ✅ `pstrlen == 19`
- ✅ `pstr == "BitTorrent protocol"`
- ✅ `info_hash == nuestro info_hash` (mismo torrent)

---

### 3. **`message.go` - Protocolo de Mensajes**

**Responsabilidad:** Enviar/recibir mensajes del protocolo BitTorrent

**Tipos de Mensajes:**
```go
MsgChoke         = 0  // "No te voy a enviar piezas"
MsgUnchoke       = 1  // "Ahora sí puedes pedirme piezas"
MsgInterested    = 2  // "Me interesan tus piezas"
MsgNotInterested = 3  // "Ya no necesito tus piezas"
MsgHave          = 4  // "Tengo la pieza X"
MsgBitfield      = 5  // "Estas son todas mis piezas"
MsgRequest       = 6  // "Dame el bloque X de la pieza Y"
MsgPiece         = 7  // "Aquí está el bloque que pediste"
MsgCancel        = 8  // "Cancela mi request anterior"
MsgPort          = 9  // "Mi puerto DHT" (no usado)
```

**Formato de Mensajes:**
```
┌──────────┬────┬─────────────────┐
│  Length  │ ID │    Payload      │
│   4B     │ 1B │   Variable      │
└──────────┴────┴─────────────────┘

Length = 1 + len(payload)
Si Length == 0 → Keep-alive (id=255 internamente)
```

**Implementación:**
```go
SendMessage(id byte, payload []byte):
  ├─ Construir: [length:4B][id:1B][payload:NB]
  └─ Enviar por TCP

ReadMessage() (id, payload, error):
  ├─ Leer length (4 bytes)
  ├─ Si length == 0 → Keep-alive (id=255)
  ├─ Leer data[length]
  └─ Retornar (data[0], data[1:], nil)
```

**Mensajes Especializados:**

1. **SendHave(index)** - Notificar nueva pieza:
   ```go
   Payload: [index:4B]
   Uso: Broadcast cuando completamos una pieza
   ```

2. **SendBitfield(bits)** - Enviar estado inicial:
   ```go
   Payload: [bits:ceil(numPieces/8) bytes]
   Uso: Justo después del handshake
   Optimización: No enviar si bitfield es todo ceros
   ```

3. **SendPiece(index, begin, data)** - Enviar bloque:
   ```go
   Formato: [length:4B][id=7:1B][index:4B][begin:4B][data:NB]
   Uso: Responder a MsgRequest del peer
   ```

---

### 4. **`manager.go` - Coordinador Central**

**Responsabilidad:** Coordinar múltiples peers y gestionar estado global

**Estructura:**
```go
type Manager struct {
    mu    sync.RWMutex
    peers []*PeerConn      // Lista de peers conectados
    store PieceStore       // Almacenamiento de piezas
}
```

**Funciones Principales:**

#### **A) Gestión de Peers**
```go
AddPeer(p *PeerConn):
  ↓ Añadir peer al pool
  ↓ El peer puede enviar/recibir mensajes
  
RemovePeer(p *PeerConn):
  ↓ Quitar del pool cuando se desconecta
```

#### **B) ReadLoop() - Loop Principal del Peer**
```go
func (p *PeerConn) ReadLoop():
    for {
        id, payload, err := p.ReadMessage()
        if err != nil {
            // Peer desconectado o error
            p.Close()
            return
        }
        p.handleMessage(id, payload)
    }
```

#### **C) handleMessage() - Estado del Protocolo**

```go
switch message_id {
case MsgChoke:
    p.PeerChoking = true
    ↓ Dejar de pedir bloques
    
case MsgUnchoke:
    p.PeerChoking = false
    ↓ Buscar pieza para descargar
    ↓ picker.NextPieceFor(p, store)
    ↓ requestNextBlocks(piece)
    
case MsgInterested:
    p.PeerInterested = true
    ↓ SendMessage(MsgUnchoke) // Permitir que nos pida
    
case MsgHave:
    index := binary.BigEndian.Uint32(payload)
    ↓ Actualizar remoteBF[index] = 1
    ↓ Si nos interesa → SendMessage(MsgInterested)
    
case MsgBitfield:
    ↓ UpdateRemoteBitfield(payload)
    ↓ Verificar si nos interesa alguna pieza
    ↓ Si sí → SendMessage(MsgInterested)
    
case MsgRequest:
    index, begin, length := parse(payload)
    ↓ Leer bloque del storage
    ↓ SendPiece(index, begin, data)
    
case MsgPiece:
    index, begin, block := parse(payload)
    ↓ store.WriteBlock(index, begin, block)
    ↓ Si pieza completa → Broadcast HAVE a todos los peers
    ↓ requestNextBlocks(index) // Pedir siguiente bloque
}
```

#### **D) requestNextBlocks() - Descargar Bloques**

**Parámetros:**
- Tamaño de bloque: `16 KB` (constante `blockLen`)
- Estrategia: Solicitar un bloque a la vez (pipeline de 1)

```go
func requestNextBlocks(piece int):
    ├─ Calcular tamaño de pieza (última puede ser menor)
    ├─ Si ya tenemos la pieza → return
    ├─ Establecer curPiece = piece, curOffset = begin
    ├─ Calcular: sz = min(blockLen, pieceSize - begin)
    ├─ Construir payload: [index:4B][begin:4B][length:4B]
    └─ SendMessage(MsgRequest, payload)
```

**Flujo de Descarga de una Pieza:**
```
1. Recibir Unchoke
2. picker.NextPieceFor() → piece = 5
3. requestNextBlocks(5):
   ├─ Request bloque: [5][0][16384]
   ├─ Recibir Piece: [5][0][16KB data]
   ├─ store.WriteBlock(5, 0, data)
   ├─ Request bloque: [5][16384][16384]
   ├─ Recibir Piece: [5][16384][16KB data]
   ├─ ... (repetir hasta completar)
   └─ Última: [5][N][remaining bytes]
4. store marca pieza completa
5. Broadcast HAVE(5) a todos los peers
```

---

### 5. **`picker.go` - Estrategia de Selección**

**Responsabilidad:** Decidir qué pieza descargar a continuación

**Algoritmo Actual: First-Needed (Naive)**
```go
func NextPieceFor(peer, store) int:
    for i := 0; i < numPieces; i++:
        if !store.HasPiece(i) AND peer.RemoteHasPiece(i):
            return i
    return -1  // No hay piezas disponibles
```

**Características:**
- ✅ Simple y funcional
- ⚠️ No optimiza para piezas raras
- ⚠️ No balancea carga entre peers

**Algoritmos Avanzados (Futuro):**

1. **Rarest-First:**
   ```go
   ↓ Contar cuántos peers tienen cada pieza
   ↓ Priorizar piezas con menor disponibilidad
   ↓ Ventaja: Mejora distribución en el swarm
   ```

2. **End-Game Mode:**
   ```go
   ↓ Cuando quedan pocas piezas (< 5%)
   ↓ Pedir bloques faltantes a TODOS los peers
   ↓ Cancelar duplicados cuando llegue el primero
   ```

3. **Sequential (Streaming):**
   ```go
   ↓ Descargar en orden: 0, 1, 2, 3...
   ↓ Permite reproducir mientras se descarga
   ```

---

### 6. **`storage.go` - Persistencia en Disco**

**Responsabilidad:** Leer/escribir bloques de archivo con verificación SHA-1

**Estructura:**
```go
type DiskPieceStore struct {
    f           *os.File       // Archivo .part o final
    pieceLength int            // Tamaño de pieza (ej: 256KB)
    totalLength int64          // Tamaño total del torrent
    numPieces   int            // ceil(total/pieceLength)
    
    bitfield    []byte         // Bitmap de piezas completas
    completed   []bool         // Por pieza: true si completa
    received    []int64        // Bytes recibidos por pieza
    
    expected    [][20]byte     // SHA-1 esperado por pieza
    cbs         []func(int)    // Callbacks al completar
}
```

**Funciones Clave:**

#### **A) WriteBlock(piece, begin, data)**
```go
1. Calcular offset = piece * pieceLength + begin
2. Buscar posición en archivo: f.Seek(offset)
3. Escribir datos: f.Write(data)
4. Actualizar received[piece] += len(data)
5. Si received[piece] >= pieceSize(piece):
   ├─ Leer pieza completa del disco
   ├─ Calcular SHA-1
   ├─ Si coincide con expected[piece]:
   │  ├─ completed[piece] = true
   │  ├─ Actualizar bitfield
   │  └─ Llamar callbacks (broadcast HAVE)
   └─ Si NO coincide:
      └─ Log error (pieza corrupta)
```

#### **B) ReadBlock(piece, begin, length)**
```go
1. Calcular offset = piece * pieceLength + begin
2. Buscar: f.Seek(offset)
3. Leer length bytes: f.Read(buf)
4. Retornar buf
```

**Uso: Responder a MsgRequest de peers**

#### **C) Bitfield()**
```go
┌─────────────────────────────────────────────┐
│  Byte 0   │  Byte 1   │  Byte 2   │  ...   │
│ 76543210  │ 76543210  │ 76543210  │  ...   │
└─────────────────────────────────────────────┘
  ↑           ↑           ↑
Pieza 0     Pieza 8     Pieza 16

Si numPieces = 20 → bitfield = 3 bytes (ceil(20/8))

HasPiece(13):
  byteIdx = 13 / 8 = 1
  bit = 7 - (13 % 8) = 2
  return (bitfield[1] & (1 << 2)) != 0
```

#### **D) Modos de Apertura**

```go
NewDiskPieceStore(path, pieceLen, totalLen):
  ↓ Modo descarga: truncate=true
  ↓ Crea archivo.part truncado al tamaño total
  ↓ bitfield inicialmente todo ceros
  
NewDiskPieceStoreWithMode(path, ..., truncate=false):
  ↓ Modo seeding: truncate=false
  ↓ Abre archivo existente sin modificar
  ↓ ScanAndMarkComplete() lee y verifica piezas
```

---

### 7. **`manager_broadcast.go` - Notificaciones Globales**

**Responsabilidad:** Broadcast de eventos a todos los peers

```go
func (m *Manager) BroadcastHave(piece int):
    m.mu.RLock()
    defer m.mu.RUnlock()
    for _, peer := range m.peers:
        peer.SendHave(uint32(piece))
```

**Uso:**
```
store.OnPieceComplete(func(piece) {
    manager.BroadcastHave(piece)
})
```

---

## 🔄 Flujo Completo: Descarga de un Torrent

### **Fase 1: Inicialización**

```
1. Cliente lee archivo.torrent
   ↓ Extrae: announce, info_hash, piece_length, pieces (SHA-1)
   
2. Genera peer_id aleatorio
   ↓ Formato: -JC0001-XXXXXXXXXXXX (20 bytes)
   
3. Calcula left = total_length - bytes_descargados
   
4. Envía announce al tracker:
   GET /announce?info_hash=...&peer_id=...&port=...&hostname=client1&event=started
   
5. Tracker responde (non-compact):
   {
     "interval": 1800,
     "peers": [
       {"ip": "client2", "port": 41801},
       {"ip": "client3", "port": 37443}
     ]
   }
```

### **Fase 2: Conexión con Peers**

```
Para cada peer en la lista:

1. NewPeerConn(addr, info_hash, peer_id)
   ↓ Establece TCP connection
   
2. Handshake()
   ↓ Enviar: [19|BitTorrent protocol|8null|info_hash|peer_id]
   ↓ Recibir y validar respuesta
   
3. SendBitfield(nuestro_bitfield)
   ↓ Informar qué piezas tenemos
   
4. SendMessage(MsgInterested, nil)
   ↓ Declarar interés
   
5. go ReadLoop()
   ↓ Escuchar mensajes del peer
```

### **Fase 3: Descarga de Piezas**

```
Peer State Machine:
┌──────────────────────────────────────────────────┐
│  Estado Inicial: Choking + Not Interested        │
└────────────────────┬─────────────────────────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │ Recibir Bitfield     │
          │ o HAVE messages      │
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │ Evaluar si nos       │
          │ interesa alguna      │
          │ pieza del peer       │
          └──────────┬───────────┘
                     │
                 ┌───┴───┐
                 │  Sí   │
                 └───┬───┘
                     ▼
          ┌──────────────────────┐
          │ SendMessage          │
          │ (MsgInterested)      │
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │ Esperar Unchoke      │
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │ picker.NextPieceFor()│
          │ → piece = 5          │
          └──────────┬───────────┘
                     │
                     ▼
     ┌───────────────────────────────────────┐
     │ Loop de Descarga (por bloque):        │
     │                                        │
     │ 1. requestNextBlocks(5)                │
     │    └─ Request [5][offset][16KB]       │
     │                                        │
     │ 2. Recibir MsgPiece                   │
     │    └─ Payload: [5][offset][data]      │
     │                                        │
     │ 3. store.WriteBlock(5, offset, data)  │
     │    └─ Escribir a disco                │
     │                                        │
     │ 4. offset += 16KB                     │
     │                                        │
     │ 5. Si offset < pieceSize:             │
     │    └─ Repetir desde paso 1            │
     │                                        │
     │ 6. Si pieza completa:                 │
     │    ├─ Verificar SHA-1                 │
     │    ├─ Marcar bitfield[5] = 1          │
     │    └─ BroadcastHave(5) a todos        │
     └────────────────────────────────────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │ Buscar siguiente     │
          │ pieza disponible     │
          └──────────┬───────────┘
                     │
                ┌────┴────┐
                │ Hay más │
                └────┬────┘
                     │
             ┌───────┴───────┐
             │      Sí       │  → Volver a requestNextBlocks
             └───────────────┘
                     │
                     │ No
                     ▼
          ┌──────────────────────┐
          │ Descarga completa    │
          │ SendMessage          │
          │ (MsgNotInterested)   │
          └──────────────────────┘
```

### **Fase 4: Seeding (Subir piezas)**

```
Cuando un peer nos envía MsgRequest:

1. handleMessage(MsgRequest, payload)
   ├─ index = payload[0:4]
   ├─ begin = payload[4:8]
   └─ length = payload[8:12]
   
2. data, err := store.ReadBlock(index, begin, length)
   ↓ Leer del disco
   
3. SendPiece(index, begin, data)
   ↓ Formato: [length:4B][7:1B][index:4B][begin:4B][data:NB]
   
4. Peer remoto recibe y persiste
```

### **Fase 5: Completación**

```
1. Todas las piezas completas (bitfield todo 1s)
   
2. Renombrar archivo.part → archivo final
   
3. Enviar announce al tracker:
   event=completed&left=0
   
4. Continuar como seeder:
   ↓ Responder a MsgRequest de otros peers
   ↓ Announces periódicos cada 30 min
```

---

## 🎲 Algoritmos y Estrategias

### 1. **Choking Algorithm (BitTorrent Standard)**

**No implementado explícitamente en tu código, pero el flujo:**

```
Estado por defecto: AmChoking = true

Cuando peer envía MsgInterested:
  └─ SendMessage(MsgUnchoke)  // Permitir descarga
  
Estrategia avanzada (futuro):
  ├─ Unchoke a los 4 peers con mejor upload rate
  ├─ Cambiar cada 10 segundos
  └─ "Optimistic unchoke" cada 30s a un peer random
```

### 2. **Piece Selection (Picker)**

**Actual: First-Needed**
```python
for piece in range(numPieces):
    if not have(piece) and peer_has(piece):
        return piece
```

**Futuro: Rarest-First**
```python
# Contar disponibilidad de cada pieza
availability = [0] * numPieces
for peer in all_peers:
    for piece in peer.bitfield:
        availability[piece] += 1

# Ordenar por rareza (menor count primero)
sorted_pieces = sorted(range(numPieces), key=lambda p: availability[p])

# Devolver primera que no tenemos
for piece in sorted_pieces:
    if not have(piece) and peer_has(piece):
        return piece
```

### 3. **Block Request Strategy**

**Parámetros:**
- Block size: `16 KB`
- Pipeline: 1 (espera respuesta antes de pedir siguiente)

**Optimización futura:**
```
Pipeline de 5-10 requests:
  ├─ Request bloques [0], [1], [2], [3], [4]
  ├─ Mientras llegan respuestas, pedir [5], [6]...
  └─ Ventaja: Menor latencia, mejor uso de bandwidth
```

### 4. **Endgame Mode**

**No implementado. Lógica:**
```
if piezas_faltantes < 5% AND tiempo > threshold:
    for piece in missing_pieces:
        for peer in all_peers:
            if peer.has(piece):
                request_all_blocks(piece, peer)
    
    # Cuando llega primer bloque completo:
    send_cancel() a todos los demás peers
```

---

## 📊 Métricas y Estadísticas

### **Información Rastreable:**

```go
// Por peer
type PeerStats struct {
    Downloaded   int64   // Bytes recibidos de este peer
    Uploaded     int64   // Bytes enviados a este peer
    DownloadRate float64 // KB/s actual
    UploadRate   float64 // KB/s actual
}

// Global
type TorrentStats struct {
    TotalDownloaded int64
    TotalUploaded   int64
    ShareRatio      float64  // uploaded/downloaded
    ETA             time.Duration
    NumPeers        int
    NumSeeders      int
}
```

### **Cálculo de ETA:**
```go
bytesLeft := totalLength - bytesDownloaded
avgRate := totalDownloaded / elapsed.Seconds()
eta := time.Duration(bytesLeft / avgRate) * time.Second
```

---

## 🔒 Validaciones y Seguridad

### **1. Validación de Piezas (SHA-1)**

```go
// En WriteBlock():
if piezaCompleta {
    data := leerPiezaCompleta(piece)
    hash := sha1.Sum(data)
    
    if hash == expected[piece] {
        ✅ Pieza válida
    } else {
        ❌ Pieza corrupta
        ↓ Descartar y reintentar
    }
}
```

### **2. Validación de Handshake**
- ✅ Info_hash coincide (evita torrents incorrectos)
- ✅ Protocolo correcto

### **3. Validación de Mensajes**
- ✅ Length válido
- ✅ Payload tamaño esperado
- ✅ Índices de pieza dentro de rango

---

## 🚀 Optimizaciones Implementadas

1. ✅ **Bitfield comprimido**: 1 bit por pieza (no bool[])
2. ✅ **Broadcast eficiente**: HAVE solo cuando completamos
3. ✅ **Validación SHA-1**: Solo al completar pieza
4. ✅ **Reutilización de conexiones**: Misma struct para entrantes/salientes
5. ✅ **Mutex granular**: RWMutex en storage para concurrencia

---

## 🎯 Próximos Pasos (Mejoras Futuras)

1. **Rarest-First Picker** → Mejor distribución
2. **Pipeline de Requests** → Menor latencia
3. **Endgame Mode** → Completar más rápido
4. **Choking Inteligente** → Mejor uso de bandwidth
5. **Métricas y Stats** → Dashboard de progreso
6. **DHT Support** → No depender del tracker
7. **uTP (μTP)** → Protocolo de transporte optimizado
8. **Encryption** → Evadir throttling de ISPs

---

