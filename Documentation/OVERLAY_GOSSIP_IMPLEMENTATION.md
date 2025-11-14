## Overlay Gossip — Implementación y guía

Fecha: 14 de noviembre de 2025

Este documento resume los cambios realizados para implementar un prototipo de descubrimiento distribuido basado en gossip (overlay) y su integración con el cliente BitTorrent existente en este repositorio. Explica los componentes nuevos, cómo probar localmente, limitaciones actuales y pasos siguientes recomendados.

## Objetivo

Proveer una alternativa al tracker centralizado: un overlay distribuido que permite `Announce` y `Lookup` mediante gossip (propagación epidémica). No se usa DHT tradicional; la solución replica información de proveedores (peers) por infohash entre nodos.

## Archivos nuevos y cambios principales

1. Nuevo paquete `src/overlay` (implementación propia):
   - `src/overlay/store.go`
     - `Store` en memoria que mantiene `infoHash -> providers`.
     - TTL para providers stale (por defecto 90 segundos).
     - Métodos: `NewStore`, `Announce`, `Merge`, `Lookup`, `ToJSON`.
     - Estructura `ProviderMeta{Addr, PeerId, Left, LastSeen}`.

   - `src/overlay/gossip.go`
     - `Overlay` controla el store, mantiene lista bootstrap y un TCP listener JSON.
     - Mensajes JSON por TCP: `gossip`, `announce`, `lookup`.
     - Periodic gossip (cada ~8s) que hace full-push de providers por infohash a peers bootstrap.
     - Métodos: `NewOverlay`, `Start`, `Stop`, `Announce`, `Lookup`.

2. Cambios en `src/client/cmd/main.go` (integración cliente):
   - Nuevos flags:
     - `--discovery-mode` (tracker|overlay) — por defecto `tracker`.
     - `--bootstrap` lista de bootstrap peers (comma-separated, `host:port`).
     - `--overlay-port` puerto TCP donde escucha overlay (default `6000`).
   - Si `--discovery-mode=overlay`:
     - Se inicia el `overlay` (listener TCP JSON y gossip periódico).
     - En vez de anunciar por HTTP al tracker, se llama `ov.Announce(infoHash, ProviderMeta)`.
     - En vez de parsear `peers` del tracker, se llama `ov.Lookup(infoHash, limit)` para obtener peers.
     - Los announces periódicos, `completed`, y `stopped` se envían al overlay si está activo.
   - Modo `tracker` mantiene comportamiento original.

3. Otros: no se modificaron los paquetes `peerwire` fundamentales excepto su uso por el cliente.

## Diseño del overlay (resumen técnico)

- Estructura de datos: por cada `infoHash` se guarda un map de `addr -> ProviderMeta`. Merge es LWW sobre `LastSeen`.
- Propagación: cada nodo envía periódicamente (full-push) la lista de providers por `infoHash` a sus peers bootstrap.
- Lookup: retorna providers locales ordenados por `LastSeen` y pregunta a hasta 3 peers bootstrap para completar la respuesta.
- TTL: providers con `LastSeen` anterior a `now - TTL` se ignoran en Lookup. TTL por defecto 90s.

Racional: diseño simple, resistente a churn moderado, evita DHT y permite políticas avanzadas luego (bloom filters, diffs, réplica adaptativa).

## Cómo probar (ejemplos)

Compilar (desde el directorio `src`):

```bash
cd /home/jabel/Documentos/GIthub_repo/BitTorrent/src
go build ./...
```

Ejecutar tres instancias locales (puedes usar `go run` para pruebas rápidas). Cambia rutas al `.torrent` y `--archives` según corresponda:

Terminal A (overlay puerto 6000):

```bash
go run ./client/cmd --torrent="../video.torrent" --archives="~/jabel/Documentos/GIthub_repo/BitTorrent/src" \
  --discovery-mode=overlay --overlay-port=6000 --bootstrap="127.0.0.1:6001,127.0.0.1:6002" --hostname=127.0.0.1
```

Terminal B (overlay puerto 6001):

```bash
go run ./client/cmd --torrent="../video.torrent" --archives="~/jabel/Documentos/GIthub_repo/BitTorrent/src" \
  --discovery-mode=overlay --overlay-port=6001 --bootstrap="127.0.0.1:6000,127.0.0.1:6002" --hostname=127.0.0.1
```

Terminal C (overlay puerto 6002):

```bash
go run ./client/cmd --torrent="../video.torrent" --archives="~/jabel/Documentos/GIthub_repo/BitTorrent/src" \
  --discovery-mode=overlay --overlay-port=6002 --bootstrap="127.0.0.1:6000,127.0.0.1:6001" --hostname=127.0.0.1
```

- Observa la salida: cada nodo imprimirá "Overlay gossip iniciado" y "Announced to overlay, left=...".
- Tras algunos ciclos (~8s cada uno) deberías observar que otros nodos obtienen providers para ese `infoHash` mediante `Lookup` y podrán conectarse.

Si quieres probar en Docker Swarm (requisito de entrega), configura `--bootstrap` con las direcciones de servicio de las instancias y expón el puerto overlay.

## Limitaciones actuales

- Gossip es full-push: envía la lista completa de providers por infoHash a cada peer bootstrap. No es escalable a escala masiva, pero es claro y suficiente para la 2ª entrega (prototipo).
- No hay autenticación ni firma en announces. Un nodo malicioso podría anunciar providers falsos. Recomendado: validación al intentar conectar (ya se verifica por SHA-1 en `peerwire`) y uso de HMAC/pubkeys si se desea seguridad.
- No hay NAT traversal: para que peers se conecten, `--hostname` debe ser una dirección alcanzable desde los otros nodos (o usar mapeo/port-forwarding en Docker).
- Sin delete explícito: eliminamos entries por TTL.

## Cómo encaja con `instructions.md` (requisitos de la 2ª entrega)

- Organización y roles: cada nodo ejecuta cliente + overlay. No existe punto central: la información se replica en todos los nodos que formen la red.
- Procesos: el sistema tiene procesos de cliente BitTorrent y un proceso overlay (listener TCP + periodic gossip + handlers). Ambos pueden correr en la misma instancia/container.
- Comunicación: gossip sobre TCP JSON para control/descubrimiento; peerwire sigue usando conexiones peer-to-peer para transferencia de piezas.
- Coordinación: decisiones distribuidas por réplica y anti-entropy (merge LWW). Para tareas que requieran exclusividad puede añadirse una elección (leader election ligera) más adelante.
- Nombrado/Localización: recurso identificado por `infoHash`. Localización mediante providers replicados en el store. Lookup realiza consultas a vecinos si la información local es insuficiente.
- Consistencia y replicación: eventual consistency mediante gossip; Merge LWW; TTL y anti-entropy permiten convergencia.
- Tolerancia a fallas: sin tracker central, la caída de nodos no detiene la localización si hay réplicas; el sistema tolera nodos caídos siempre que exista réplica suficiente.
- Seguridad: pendiente (ver Limitaciones). Para la entrega, justificarás validación en transferencia (SHA-1) y políticas de reputación como mitigación.

## Tests y verificación

- Compilación: `cd src && go build ./...` (ya verificado localmente durante la implementación).
- Test unitarios pendientes: crear tests para `Store.Merge`, TTL eviction y `Lookup`.
- Integration tests: probar con 3 instancias en CI/local con goroutines o contenedores.

## Próximos pasos recomendados (prioridad)

1. Añadir tests unitarios para `src/overlay` (merge, announce, lookup, TTL).  
2. Implementar `leave` o tombstones y/o un mecanismo explícito para remover providers.  
3. Optimizar gossip (push/pull diffs, bloom filters) para reducir ancho de banda.  
4. Añadir validación/handshake adicional en overlay y opcional HMAC para mitigación de falsos announces.  
5. Documentar despliegue en Docker Swarm (variables `--bootstrap` y cómo exponer `--overlay-port` entre hosts).

## Resumen de los cambios (lista rápida)

- Añadido: `src/overlay/store.go` (Store, ProviderMeta).
- Añadido: `src/overlay/gossip.go` (Overlay, TCP JSON gossip, Announce/Lookup).
- Modificado: `src/client/cmd/main.go` — integré `--discovery-mode=overlay` y flags `--bootstrap`, `--overlay-port`, y encaminé announces/lookups al overlay cuando está activo.

Si quieres, puedo:
- Añadir tests unitarios e integrados ahora.
- Implementar optimizaciones de gossip (diffs/bloom) y/o `leave`/tombstones.
- Generar el informe final formateado para la 2ª entrega rellenando las secciones de `instructions.md` con el diseño y decisiones.

---
Archivo generado automáticamente que documenta la implementación del overlay gossip del repositorio.
