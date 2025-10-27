# 🔄 Diagramas de Secuencia - BitTorrent P2P

## Diagrama 1: Conexión Inicial y Handshake

```
Cliente A                Tracker              Cliente B
   │                        │                     │
   │  GET /announce         │                     │
   │  ?event=started        │                     │
   │  &hostname=client1     │                     │
   ├───────────────────────>│                     │
   │                        │                     │
   │  {"peers": [           │                     │
   │    {"ip":"client2",    │                     │
   │     "port":41801}]}    │                     │
   │<───────────────────────┤                     │
   │                        │                     │
   │         TCP SYN (client2:41801)              │
   ├─────────────────────────────────────────────>│
   │                        │          TCP SYN-ACK│
   │<─────────────────────────────────────────────┤
   │         TCP ACK        │                     │
   ├─────────────────────────────────────────────>│
   │                        │                     │
   │  Handshake Request     │                     │
   │  [19|BitTorrent...     │                     │
   │   |info_hash|peer_id]  │                     │
   ├─────────────────────────────────────────────>│
   │                        │                     │
   │                        │   Validar info_hash │
   │                        │   Validar protocol  │
   │                        │                     │
   │  Handshake Response    │                     │
   │<─────────────────────────────────────────────┤
   │                        │                     │
   │  Validar info_hash     │                     │
   │                        │                     │
   │  ✅ Conexión OK        │                     │
   │                        │                     │
```

---

## Diagrama 2: Intercambio de Bitfields y Descarga

```
Cliente A (Leecher)                      Cliente B (Seeder)
Has: [0,1,2]                             Has: [0,1,2,3,4,5,6,7]
Wants: [3,4,5,6,7]                       
   │                                           │
   │  MsgBitfield [00001111]                  │
   │  (tengo piezas 0,1,2,3)                  │
   ├──────────────────────────────────────────>│
   │                                           │
   │                     MsgBitfield [11111111]│
   │                     (tengo todas)         │
   │<──────────────────────────────────────────┤
   │                                           │
   │  MsgInterested                            │
   │  (me interesan tus piezas)                │
   ├──────────────────────────────────────────>│
   │                                           │
   │                                Evaluar    │
   │                                choking    │
   │                                           │
   │                          MsgUnchoke       │
   │                          (puedes pedir)   │
   │<──────────────────────────────────────────┤
   │                                           │
   │  Picker: NextPieceFor()                   │
   │  → piece = 3 (primera que no tengo)       │
   │                                           │
   │  MsgRequest                               │
   │  [piece=3, begin=0, len=16384]            │
   ├──────────────────────────────────────────>│
   │                                           │
   │                          ReadBlock(3,0)   │
   │                          desde disco      │
   │                                           │
   │                     MsgPiece              │
   │                     [3][0][16KB data]     │
   │<──────────────────────────────────────────┤
   │                                           │
   │  WriteBlock(3, 0, data)                   │
   │  → escribir a disco                       │
   │                                           │
   │  MsgRequest                               │
   │  [piece=3, begin=16384, len=16384]        │
   ├──────────────────────────────────────────>│
   │                                           │
   │                     MsgPiece              │
   │                     [3][16384][16KB data] │
   │<──────────────────────────────────────────┤
   │                                           │
   │  ... (repetir hasta completar pieza)      │
   │                                           │
   │  WriteBlock(3, N, last_data)              │
   │  → Pieza 3 completa                       │
   │  → Verificar SHA-1 ✅                     │
   │  → Marcar bitfield[3] = 1                 │
   │                                           │
```

---

## Diagrama 3: Broadcast HAVE a Múltiples Peers

```
Cliente A                  Cliente B         Cliente C         Cliente D
bitfield: [111]            [110]             [100]             [101]
   │                          │                 │                 │
   │  Completa pieza 3        │                 │                 │
   │  SHA-1 OK ✅             │                 │                 │
   │                          │                 │                 │
   │  MsgHave(3)              │                 │                 │
   ├─────────────────────────>│                 │                 │
   ├────────────────────────────────────────────>│                 │
   ├──────────────────────────────────────────────────────────────>│
   │                          │                 │                 │
   │                          │  Actualizar     │  Actualizar     │
   │                          │  remoteBF[A]    │  remoteBF[A]    │
   │                          │  bit 3 = 1      │  bit 3 = 1      │
   │                          │                 │                 │
   │                          │  Evaluar:       │  Evaluar:       │
   │                          │  ¿Necesito 3?   │  ¿Necesito 3?   │
   │                          │  → Sí           │  → Sí           │
   │                          │                 │                 │
   │         MsgInterested    │                 │                 │
   │<─────────────────────────┤                 │                 │
   │                          │  MsgInterested  │                 │
   │<────────────────────────────────────────────┤                 │
   │                          │                 │                 │
   │  MsgUnchoke              │                 │                 │
   ├─────────────────────────>│                 │                 │
   │  MsgUnchoke              │                 │                 │
   ├────────────────────────────────────────────>│                 │
   │                          │                 │                 │
   │         MsgRequest       │  MsgRequest     │                 │
   │         [3][0][16KB]     │  [3][0][16KB]   │                 │
   │<─────────────────────────┤<────────────────┤                 │
   │                          │                 │                 │
   │  MsgPiece [3][0][...]    │                 │                 │
   ├─────────────────────────>│                 │                 │
   │  MsgPiece [3][0][...]    │                 │                 │
   ├────────────────────────────────────────────>│                 │
   │                          │                 │                 │
```

---

## Diagrama 4: Ciclo Completo con Tracker

```
Cliente                 Tracker                 Storage (Disco)
   │                       │                          │
   │  1. Announce          │                          │
   │  event=started        │                          │
   ├──────────────────────>│                          │
   │                       │                          │
   │  peers list           │                          │
   │<──────────────────────┤                          │
   │                       │                          │
   │  2. Conectar a peers  │                          │
   │  (handshake)          │                          │
   │                       │                          │
   │  3. Intercambio       │                          │
   │  de bitfields         │                          │
   │                       │                          │
   │  4. Request bloques   │                          │
   │  (MsgRequest)         │                          │
   │                       │                          │
   │  5. Recibir bloques   │                          │
   │  (MsgPiece)           │                          │
   │                       │                          │
   │                       │   WriteBlock(piece, offset, data)
   │                       │   ├──────────────────────>│
   │                       │   │  f.Seek(offset)       │
   │                       │   │  f.Write(data)        │
   │                       │   │<──────────────────────┤
   │                       │                          │
   │  6. Pieza completa    │                          │
   │                       │   ReadPiece(piece)       │
   │                       │   ├──────────────────────>│
   │                       │   │  leer del disco       │
   │                       │   │  sha1.Sum(data)       │
   │                       │   │<──────────────────────┤
   │                       │   SHA-1 Match ✅         │
   │                       │                          │
   │  7. Broadcast HAVE    │                          │
   │  a todos los peers    │                          │
   │                       │                          │
   │  ... (repetir 4-7)    │                          │
   │                       │                          │
   │  8. Todas completas   │                          │
   │                       │   Rename .part → final   │
   │                       │   ├──────────────────────>│
   │                       │   │<──────────────────────┤
   │                       │                          │
   │  9. Announce          │                          │
   │  event=completed      │                          │
   ├──────────────────────>│                          │
   │                       │  Update stats            │
   │                       │  (complete++)            │
   │                       │                          │
   │  10. Periodic         │                          │
   │  announces cada 30min │                          │
   ├──────────────────────>│                          │
   │                       │                          │
   │  11. Shutdown         │                          │
   │  event=stopped        │                          │
   ├──────────────────────>│                          │
   │                       │  RemovePeer()            │
   │                       │                          │
```

---

## Diagrama 5: Estado del Peer (State Machine)

```
                  ┌─────────────────────┐
                  │   INITIAL STATE     │
                  │  Choking=true       │
                  │  Interested=false   │
                  └──────────┬──────────┘
                             │
                             │ Handshake OK
                             ▼
                  ┌─────────────────────┐
                  │  CONNECTED          │
                  └──────────┬──────────┘
                             │
                             │ Recibir Bitfield
                             ▼
                  ┌─────────────────────┐
                  │  HAVE REMOTE        │
                  │  BITFIELD           │
                  └──────────┬──────────┘
                             │
                             │ Evaluar interés
                             ▼
                  ┌─────────────────────┐
           ┌─────>│  INTERESTED         │
           │      │  (AmInterested=true)│
           │      └──────────┬──────────┘
           │                 │
           │                 │ Recibir Unchoke
           │                 ▼
           │      ┌─────────────────────┐
           │      │  CAN DOWNLOAD       │
           │      │  (PeerChoking=false)│
           │      └──────────┬──────────┘
           │                 │
           │                 │ Picker: NextPieceFor()
           │                 ▼
           │      ┌─────────────────────┐
           │      │  DOWNLOADING        │
           │      │  (curPiece set)     │
           │      └──────────┬──────────┘
           │                 │
           │                 │ Loop: Request→Piece
           │                 │ WriteBlock()
           │                 │
           │                 │ Pieza completa
           │                 ▼
           │      ┌─────────────────────┐
           │      │  PIECE COMPLETE     │
           │      │  Broadcast HAVE     │
           │      └──────────┬──────────┘
           │                 │
           │                 │ Más piezas?
           │                 ├─── Sí ────┐
           │                 │           │
           │                 No          │
           │                 │           │
           │                 ▼           ▼
           │      ┌─────────────────────┐
           │      │  ALL COMPLETE       │
           │      │  SendNotInterested  │
           │      └─────────────────────┘
           │
           │      Si peer envía HAVE de pieza
           │      que necesitamos, volver a
           └──────INTERESTED

```

---

## Diagrama 6: Pipeline de Requests (Futuro)

```
Actual (Pipeline = 1):
───────────────────────────────────────────────────────>
  Request[0]  →  Wait  →  Piece[0]  →  Request[1]  →  Wait ...
  ────────────  ─────────  ──────────  ────────────  ──────
      RTT          RTT        RTT          RTT


Optimizado (Pipeline = 5):
───────────────────────────────────────────────────────>
  Req[0]  Req[1]  Req[2]  Req[3]  Req[4]
  ─────┐  ─────┐  ─────┐  ─────┐  ─────┐
       │       │       │       │       │
    Piece[0] Piece[1] Piece[2] ...
    ─────────────────────────────────────
         RTT (una sola vez)
         
Ventaja: 5x menos latencia total
```

---

## Diagrama 7: Docker Swarm DNS Resolution

```
Cliente A (client1)         Docker Swarm          Cliente B (client2)
Container IP: 10.0.1.8      DNS Service           Container IP: 10.0.1.9
   │                             │                        │
   │  Announce                   │                        │
   │  hostname=client1           │                        │
   ├────────────────────────────>│                        │
   │                             │  Store:                │
   │                             │  client1→10.0.1.8      │
   │                             │                        │
   │                             │  Announce              │
   │                             │  hostname=client2      │
   │                             │<───────────────────────┤
   │                             │  Store:                │
   │                             │  client2→10.0.1.9      │
   │                             │                        │
   │  GET /announce              │                        │
   ├────────────────────────────>│                        │
   │                             │  Return:               │
   │  {"peers": [                │  [{"ip":"client2",     │
   │    {"ip":"client2",         │     "port":41801}]     │
   │     "port":41801}]}         │                        │
   │<────────────────────────────┤                        │
   │                             │                        │
   │  Conectar a "client2:41801" │                        │
   │  DNS lookup: client2        │                        │
   ├────────────────────────────>│                        │
   │                             │  Resolver:             │
   │                             │  client2 → 10.0.1.9    │
   │  Resolved: 10.0.1.9:41801   │                        │
   │<────────────────────────────┤                        │
   │                             │                        │
   │  TCP connect to 10.0.1.9:41801                       │
   ├──────────────────────────────────────────────────────>│
   │                             │                        │
   │                      Handshake + Transfer            │
   │<─────────────────────────────────────────────────────>│
   │                             │                        │
```

---

## Resumen de Protocolos

### HTTP (Tracker Communication)
```
GET /announce?info_hash=XXX&peer_id=YYY&port=6881&hostname=client1&event=started
→ Response: bencode{interval, peers: [{ip, port}]}

GET /scrape?info_hash=XXX
→ Response: bencode{files: {hash: {complete, incomplete, downloaded}}}
```

### TCP (Peer Wire Protocol)
```
1. Handshake: [19|"BitTorrent protocol"|8null|info_hash|peer_id]
2. Messages: [length:4B][id:1B][payload:NB]
   - Choke/Unchoke (control de flujo)
   - Interested/NotInterested (declaración de interés)
   - Have (notificación de nueva pieza)
   - Bitfield (estado inicial)
   - Request (solicitud de bloque)
   - Piece (envío de bloque)
```

### Bencode (Serialization)
```
Integers:  i42e
Strings:   4:spam
Lists:     l4:spam4:eggse
Dicts:     d3:cow3:moo4:spam4:eggse
```

---

