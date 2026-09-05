# Auditoría funcional: implementado frente a documentado

## 1. Criterio de clasificación

- **Implementado**: existe código activo y una ruta de ejecución razonablemente identificable.
- **Parcial**: existe una base funcional, pero la garantía o el alcance descrito es mayor que lo que cubre el código.
- **Documentado, no activo o no verificable**: aparece en la documentación, ejemplos o diagramas, pero no se encuentra una ruta activa equivalente en `src/`.
- **Pendiente valioso**: mejora concreta que elevaría la calidad técnica del proyecto.

## 2. Funcionalidades reales verificadas en `src`

| Área | Estado | Evidencia y alcance real |
| --- | --- | --- |
| Lectura de `.torrent` | Implementado | `client/config.go` decodifica bencode, calcula SHA-1 de `info`, extrae piezas y hashes. |
| Cliente BitTorrent | Implementado | `client/cmd/main.go` arranca listener, descubrimiento, peerwire, announces y shutdown. |
| Tracker HTTP | Implementado | `tracker/cmd/main.go` registra `/announce`, `/scrape` y `/nodes`. |
| Announce BitTorrent | Implementado | `tracker/announce.go` valida `info_hash`, `peer_id`, puerto y contadores; registra eventos y devuelve peers. |
| Formato compacto/no compacto | Implementado | El tracker usa compacto para IPv4 y lista con hostname cuando es necesario para Docker. |
| Scrape | Implementado | `tracker/scrape.go` devuelve estadísticas bencodeadas. |
| Peerwire | Implementado | `peerwire/handshake.go`, `message.go`, `conn.go` y `manager.go` cubren handshake y mensajes básicos. |
| Descarga por bloques | Implementado | `peerwire/manager_broadcast.go` reparte bloques de una pieza entre peers y procesa timeouts. |
| Round-Robin | Implementado | El bucle `peerIndex%len(availablePeers)` asigna bloques circularmente entre peers elegibles. |
| Reintento ante fallo | Implementado | `RemovePeer`, `watchBlockTimeout` y `retryPendingBlocks` liberan y reintentan bloques. |
| Verificación de piezas | Implementado | `DiskPieceStore` valida SHA-1 por pieza cuando recibe todos sus bytes. |
| Seeder desde archivo existente | Implementado | `ScanAndMarkComplete` compara el archivo con los hashes esperados. |
| Failover de trackers | Implementado | `SendAnnounceWithFailover` recorre trackers y cambia el índice activo ante errores. |
| Selección por latencia | Implementado | `SelectAndReorderTrackers` mide trackers y reordena `AnnounceURLs`; la medición es secuencial. |
| Persistencia del tracker | Implementado | `persist.go` carga JSON y guarda mediante temporal/reemplazo. |
| Expiración de peers | Implementado | `GC` marca inactivos con tombstone y elimina tombstones antiguos. |
| Sincronización multi-tracker | Implementado | `sync.go` hace push periódico HTTP de estado completo a peers remotos. |
| HMAC entre trackers | Implementado con limitaciones | `security.go` firma HMAC-SHA256 y `/sync` rechaza firma ausente/inválida. El secreto es fijo en código y no hay TLS. |
| Merge distribuido | Implementado con limitaciones | `sync_merge.go` usa HLC, LWW y tombstones; no es consenso ni consistencia fuerte. |
| Overlay de clientes | Implementado como prototipo | `overlay/gossip.go` soporta announce, lookup, consultas a peers y mensajes JSON TCP. Depende de bootstrap. |
| TTL del overlay | Implementado | El overlay crea un store con TTL; la frescura depende de announces/gossip. |
| API HTTP del cliente | Implementado | Estado, salud, pausa y reanudación están en `client/http_server.go`. |
| DNS auxiliar | Implementado como servicio separado | UDP DNS A, API de registros y gossip propio existen en `src/dns/`; el tracker no lo activa por defecto. |

## 3. Funcionalidades parcialmente implementadas

### 3.1 Seguridad

La sincronización de trackers tiene autenticación de mensaje mediante un secreto compartido, pero el proyecto no ofrece seguridad completa del sistema. No hay gestión de secretos, rotación de claves, TLS, autorización por endpoint ni identidad individual verificable. El overlay y peerwire no autentican al participante.

### 3.2 Consistencia distribuida

HLC, LWW y tombstones proporcionan una estrategia para ordenar y fusionar eventos, y el gossip permite convergencia eventual en condiciones favorables. No garantizan convergencia formal ante todos los conflictos, disponibilidad durante toda partición, ni una decisión única global. Tampoco hay quorum o consenso.

### 3.3 Descubrimiento overlay

La ruta está activa, pero no es una red overlay autónoma completa: los peers se configuran por `--bootstrap`, el gossip es push de mensajes completos y el lookup consulta como máximo tres peers. No hay membresía dinámica, difusión con TTL de mensajes, anti-entropy pull robusta, deduplicación por versión ni autenticación.

### 3.4 Métricas y control

La API de control existe y la métrica de descarga se calcula desde el archivo. El upload aparece expresamente pendiente en `client/http_server.go`; por ello velocidades, estado de seeding y ETA no deben venderse como telemetría completa.

### 3.5 Alta disponibilidad

El cliente tiene failover entre trackers y reintentos entre peers. Eso mejora la disponibilidad, pero no convierte automáticamente al sistema en tolerante a cualquier fallo de nivel 2: faltan pruebas de particiones, recuperación de procesos, reinicio coordinado, límites de fallos y una demostración reproducible multi-host.

## 4. Afirmaciones documentadas que deben matizarse

| Afirmación frecuente en documentación | Veredicto de auditoría |
| --- | --- |
| “Sistema completamente descentralizado” | Matizar: el modo overlay evita el tracker para discovery, pero necesita bootstrap configurado y la descarga depende de peers alcanzables. El modo tracker sigue siendo centralizado en discovery. |
| “Gossip periódico del overlay” | Activo en `Overlay.Start`, aunque parte del código antiguo de gossip está comentada y la implementación activa usa push/lookup más acotados. |
| “Trackers distribuidos con consistencia eventual” | Sustancialmente activo, limitado a push HTTP de estado completo y merge HLC/LWW. No implica consistencia fuerte. |
| “HMAC proporciona seguridad de comunicación” | Solo parcialmente: protege integridad y autenticidad compartida del endpoint `/sync`; no cifra transporte ni protege cliente, overlay o tracker público. |
| “Round-Robin mejora 3x o garantiza balance perfecto” | El algoritmo sí está implementado; las cifras de rendimiento y el balance dependen del número de peers, capacidad, choking, red y pruebas reproducibles. No se deben afirmar como garantía general sin benchmarks. |
| “Docker Swarm multi-PC funcionando” | La documentación describe despliegue y el código soporta hostnames, pero `src` no contiene una prueba de Swarm multi-host ni puede demostrar por sí mismo que el entorno está configurado. |
| “Frontend web completo” | Hay un `web/` en el repositorio principal, pero esta auditoría de implementación distribuida no valida que todas sus funcionalidades documentadas estén conectadas al backend. Debe auditarse por separado. |
| “Runtime modular activo” | No: `src/client/runtime/` contiene una implementación alternativa comentada. El camino real es `client/cmd/main.go`. |
| “DNS integrado con tracker” | No en el arranque actual: las líneas de registro DNS del tracker están comentadas. DNS existe como servicio separado. |
| “Failover completo para todas las operaciones” | Parcial: el cliente usa failover para announces del modo tracker, pero scrape inicial se hace contra el tracker actual sin una rutina equivalente de failover. |

## 5. Aspectos valiosos para un reclutador

### Fortalezas demostrables

1. **Diseño con dos topologías**: modo centralizado para operación simple y modo overlay para explorar discovery distribuido.
2. **Separación de responsabilidades**: cliente, tracker, peerwire, overlay, DNS y almacenamiento tienen paquetes diferenciados.
3. **Protocolos de red reales**: HTTP, TCP JSON, UDP DNS y peerwire, con listeners y goroutines concurrentes.
4. **Tolerancia a fallos localizada**: failover de trackers, timeout de bloques, limpieza de conexiones y reintentos.
5. **Integridad de datos**: `info_hash`, hashes SHA-1 de piezas, bitfield y validación antes de completar.
6. **Estado distribuido explícito**: HLC, LWW, tombstones, persistencia y merge, que son conceptos defendibles en una entrevista.
7. **Persistencia razonable**: guardado temporal y reemplazo para reducir corrupción de estado.
8. **Operabilidad**: health endpoint, status endpoint, logs, pausa/reanudación y configuración por flags.
9. **Decisiones conscientes**: los comentarios y documentos reconocen límites del gossip full-push y del secreto embebido.

### Preguntas que un reclutador técnico probablemente haría

- ¿Qué sucede si dos trackers actualizan el mismo peer con timestamps concurrentes?
- ¿Qué propiedad exacta ofrece HLC frente a un reloj físico?
- ¿Por qué LWW es aceptable para presencia de peers y no para datos de archivo?
- ¿Cómo se detecta y recupera una partición de red?
- ¿Qué evita recibir dos veces un mismo bloque y qué ocurre si se reciben bytes duplicados?
- ¿Por qué el contador `received` puede ser delicado con bloques repetidos o solapados?
- ¿Cómo se protegerían `/announce`, `/sync` y el overlay en producción?
- ¿Qué benchmark demuestra el beneficio del Round-Robin?
- ¿Qué pruebas existen para reinicio, peer caído, tracker caído y divergencia de réplicas?

## 6. Funcionalidades recomendadas como siguiente inversión

### Prioridad alta

1. Añadir tests unitarios y de integración para bencode, announce, persistencia, HLC/LWW, HMAC, almacenamiento y peerwire.
2. Sustituir el secreto hardcodeado por variable de entorno o un proveedor de secretos y añadir TLS/mTLS para `/sync`.
3. Corregir el conteo de bytes recibidos para que sea por rangos únicos, tolerando retransmisiones y bloques duplicados.
4. Añadir límites de tamaño, timeouts de servidor y validación estricta a los endpoints HTTP y mensajes JSON.
5. Crear un escenario reproducible con Docker Compose/Swarm para probar caída de tracker, caída de peer y convergencia.

### Prioridad media

1. Implementar gossip por deltas o push/pull con versiones, en vez de enviar siempre el estado completo.
2. Añadir anti-entropy, membresía y eliminación explícita de vecinos en el overlay.
3. Hacer failover también para scrape y definir backoff/reintentos con límites.
4. Completar métricas reales de upload/download y exponerlas con un contrato documentado.
5. Añadir pruebas de rendimiento que comparen Round-Robin con asignación secuencial bajo distintas topologías.

### Prioridad de producto

1. Conectar y validar el frontend contra los endpoints reales.
2. Integrar DNS solo si aporta valor al despliegue; retirar o aislar claramente el código experimental si no se usa.
3. Documentar una matriz de garantías: qué ocurre ante caída de proceso, red, disco, tracker, peer y nodo de overlay.

## 7. Conclusión

El núcleo implementado es defendible como un sistema BitTorrent distribuido académico con discovery alternativo, transferencia P2P, persistencia, failover y replicación eventual. La presentación más rigurosa es evitar expresiones absolutas como “consistencia garantizada”, “seguridad completa”, “descentralización total” o “tolerancia a fallos de nivel 2” sin acompañarlas de las condiciones y pruebas que las respaldan.
