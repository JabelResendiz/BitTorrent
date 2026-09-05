# Arquitectura actual del sistema BitTorrent

## 1. Alcance y criterio de lectura

Este documento describe el comportamiento que puede verificarse en el código activo de `src/`. La documentación existente se utiliza como contexto, pero no se considera prueba de una funcionalidad si no existe una ruta ejecutable que la implemente.

El proyecto contiene dos modos de descubrimiento de peers:

- **Tracker**: el cliente consulta uno o varios trackers HTTP.
- **Overlay/Gossip**: los clientes mantienen un store local de providers y consultan nodos bootstrap mediante TCP/JSON.

En ambos casos, la transferencia del contenido se realiza mediante el protocolo peerwire implementado en `src/peerwire/`.

## 2. Componentes y responsabilidades

### Cliente BitTorrent

El entrypoint activo es `src/client/cmd/main.go`.

1. Lee flags y carga el `.torrent` mediante `LoadTorrentMetadata`.
2. Decodifica bencode, obtiene `info`, calcula el `info_hash` con SHA-1 y extrae tamaño, tamaño de pieza y hashes esperados.
3. Abre un listener TCP en un puerto efímero para peerwire.
4. Inicializa el modo de descubrimiento elegido.
5. Inicializa almacenamiento de disco y el `Manager` de descargas.
6. Expone HTTP local para estado, salud, pausa y reanudación.
7. Descubre peers, conecta con ellos, acepta conexiones entrantes y arranca announces periódicos.
8. Ante `completed`, `stopped` o una señal del sistema, notifica al tracker o overlay.

### Tracker HTTP

El entrypoint es `src/tracker/cmd/main.go`. Expone:

- `GET /announce`: registra o actualiza un peer y devuelve peers del swarm.
- `GET /scrape`: devuelve estadísticas del torrent.
- `GET /nodes`: muestra nombres de nodos activos observados.
- `POST /sync` en el listener de sincronización: recibe estado de otros trackers.

El estado se agrupa por `info_hash` y por `peer_id`. Cada peer conserva hostname/IP, puerto, estado completed, timestamp HLC y un tombstone lógico.

### Overlay de clientes

`src/overlay/` implementa un store local de providers y mensajes JSON sobre TCP. Un `Overlay`:

- escucha en el puerto configurado;
- anuncia localmente un provider y lo envía a sus peers configurados;
- hace `Lookup` local y consulta hasta tres peers;
- mantiene expiración de entradas mediante TTL en el store;
- arranca gossip periódico y health check desde `Start`.

El cliente realiza un `Discover` síncrono antes de su primer `Announce`, y después obtiene providers para conectarse.

### Peerwire

`src/peerwire/` implementa la conexión entre clientes:

- handshake BitTorrent con validación del protocolo y `info_hash`;
- mensajes de bitfield, have, request y piece;
- lectura concurrente de mensajes;
- selección de piezas disponibles;
- descarga paralela de bloques;
- liberación y reintento de bloques cuando un peer se desconecta o supera 30 segundos.

El almacenamiento es `DiskPieceStore`, respaldado por un archivo `.part`. Al completar una pieza, puede validar su SHA-1 contra los hashes del torrent y notifica al manager para emitir `HAVE`.

### DNS auxiliar

`src/dns/` es un servicio separado. Tiene una API HTTP de registro/listado, un servidor DNS UDP para registros A y gossip TCP propio en el puerto 5300. Su gossip intercambia registros completos y actualiza el store local. En el tracker, la integración DNS aparece comentada en el entrypoint, por lo que no forma parte del arranque normal del tracker.

## 3. Flujo real de una descarga

```text
.torrent
  -> bencode + SHA-1(info)
  -> configuración de piezas y hashes
  -> listener peerwire
  -> descubrimiento por tracker o overlay
  -> handshake con cada peer
  -> bitfield / HAVE
  -> selección de pieza
  -> REQUEST de bloques en paralelo
  -> escritura en archivo .part
  -> SHA-1 de pieza
  -> HAVE a peers
  -> renombrado o finalización del archivo según SetupStorage
```

En modo tracker, `SendAnnounceWithFailover` prueba la lista de trackers en orden. Antes de iniciar, el cliente puede reordenarlos por latencia. El tracker devuelve peers en formato compacto IPv4 o en formato no compacto cuando necesita transportar hostnames Docker.

En modo overlay, el cliente no hace announce HTTP. Descubre providers mediante el overlay, se anuncia después de la discovery inicial y usa las direcciones devueltas para iniciar peerwire.

## 4. Persistencia y estado

### Cliente

El archivo de descarga se abre mediante `DiskPieceStore`. La estructura de bitfield y los contadores de bloques viven en memoria durante la ejecución. Para un seeder, `ScanAndMarkComplete` compara cada pieza del archivo existente con el hash esperado y construye el bitfield.

El contenido de una pieza solo se considera completo después de recibir todos sus bytes contabilizados y, si se configuraron hashes, superar la validación SHA-1.

### Tracker

El tracker guarda JSON en `/data/<hostname>_data.json`. `SaveOnChange` actualiza el estado y `SaveToFile` escribe primero en un temporal y luego reemplaza el archivo, reduciendo el riesgo de dejar un JSON truncado por una interrupción.

La recolección de basura marca peers inactivos como tombstones y elimina tombstones suficientemente antiguos. Las eliminaciones lógicas permiten propagar bajas durante la sincronización.

## 5. Coordinación y concurrencia

La coordinación es principalmente local y asíncrona:

- `sync.RWMutex` protege stores, peers y estado del tracker.
- Goroutines atienden conexiones, announces, gossip, timeouts y callbacks.
- El manager reserva una pieza antes de descargarla para evitar solicitudes duplicadas.
- Cada bloque en curso se asocia a un peer y tiene timestamp de solicitud.
- Al caer un peer, sus bloques vuelven a pendientes y se reintentan con otros peers.
- El tracker usa HLC para ordenar actualizaciones distribuidas y LWW por peer.

No se implementa un algoritmo de consenso, elección de líder, quorum, bloqueo distribuido ni transacciones distribuidas.

## 6. Consistencia, replicación y CAP

### Qué se garantiza realmente

- **Integridad del contenido**: el `info_hash` identifica el torrent y cada pieza puede validarse con SHA-1.
- **Consistencia local del tracker**: las operaciones de memoria están protegidas por mutex.
- **Consistencia de almacenamiento del tracker ante guardado**: el reemplazo mediante archivo temporal es atómico a nivel de estrategia de escritura.
- **Consistencia eventual entre trackers**: el estado se replica por push periódico; el merge usa HLC/LWW y tombstones.
- **Disponibilidad operativa ante fallo de tracker**: el cliente intenta otros trackers configurados.
- **Recuperación de descarga ante fallos de peers**: los bloques en timeout o asociados a conexiones cerradas se reintentan.

### Qué no se garantiza

- No hay consistencia fuerte ni linearizabilidad entre trackers.
- No se garantiza que todos los trackers estén sincronizados en un instante dado.
- No hay quorum para aceptar una actualización.
- No existe consenso para resolver conflictos; el criterio es LWW, dependiente de relojes HLC y del orden observado.
- El overlay puede perder anuncios durante fallos o particiones hasta que otro nodo los propague, y sus consultas son best-effort.
- El TTL y la recolección pueden eliminar información válida si un nodo no refresca a tiempo.

### Lectura CAP

La configuración distribuida de trackers y el overlay priorizan **disponibilidad y tolerancia a particiones (AP)** sobre consistencia fuerte. Durante una partición, cada tracker puede seguir aceptando announces y el overlay puede seguir descubriendo peers locales. Cuando la comunicación vuelve, se intenta converger mediante gossip y merge LWW.

Esto no significa que el sistema sea AP formal en el sentido de una especificación completa: no hay pruebas de disponibilidad bajo todas las fallas, ni límites formales de convergencia. La formulación rigurosa es: **diseño orientado a AP, con consistencia eventual y sin consenso**.

## 7. Seguridad actual

La sincronización de trackers firma el JSON sin el campo `signature` con HMAC-SHA256 y valida la firma con comparación en tiempo constante. Los mensajes sin firma o con firma inválida reciben `401`.

Las limitaciones importantes son:

- el secreto HMAC está hardcodeado en `src/tracker/security.go`;
- HMAC autentica a quien conoce el secreto, pero no establece identidad individual por tracker;
- la sincronización usa HTTP sin TLS;
- `/announce`, `/scrape`, `/nodes` y el HTTP de control del cliente no tienen autenticación;
- el overlay no firma ni autentica announces;
- el handshake peerwire valida protocolo e `info_hash`, pero no autentica criptográficamente al peer.

Por tanto, existe protección de integridad/autenticidad compartida para sync de trackers, pero no una arquitectura de seguridad de extremo a extremo.

## 8. Límites operativos visibles en el código

- El listener peerwire usa puerto efímero; el hostname y el puerto anunciado deben ser alcanzables desde los demás contenedores.
- El overlay depende de peers bootstrap configurados manualmente; no implementa descubrimiento automático de vecinos.
- El gossip de trackers envía el estado completo, no deltas.
- `SelectClosestTracker` mide secuencialmente y la función deja una variable de reordenamiento sin uso; la reordenación real está en `SelectAndReorderTrackers`.
- Las métricas de upload están marcadas como pendientes y actualmente no representan tráfico real.
- El runtime alternativo bajo `src/client/runtime/` está comentado y no es el camino de ejecución actual.

## 9. Resumen ejecutivo

El proyecto es un cliente BitTorrent funcional con dos mecanismos de descubrimiento, transferencia P2P basada en sockets, almacenamiento por piezas con verificación SHA-1, descarga paralela con reintentos, tracker HTTP persistente y sincronización multi-tracker de consistencia eventual. Su aportación de sistemas distribuidos está en la replicación asíncrona, HLC/LWW, tombstones, failover y overlay gossip; no en consenso ni consistencia fuerte.
