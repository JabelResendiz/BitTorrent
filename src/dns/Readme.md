
# BitTorrent + DNS Distribuido con Gossip

Este proyecto implementa un cliente BitTorrent que incluye soporte para un **DNS distribuido** y un **overlay network basado en gossip** para sincronizar información entre nodos. Además, se explica cómo Docker Swarm maneja redes overlay y la propagación de información sin broadcast.

---

## Estructura de Archivos

### `src/dns`
Contiene la implementación del servidor DNS distribuido.

- **`main.go`**: 
  - Inicializa el servidor DNS (`UDP`) en el puerto `8053`.
  - Inicializa el servidor HTTP API para agregar, eliminar y listar registros en `6969`.
  - Inicia el servicio de gossip para sincronizar registros entre peers.
  - Ejecución típica:
    ```bash
    cd src/dns
    go run cmd/main.go
    ```

- **`internal/store.go`**: 
  - Mantiene los registros DNS locales en un **map thread-safe**.
  - Métodos principales: `Add`, `Delete`, `Get`, `List`.

- **`internal/gossip.go`**: 
  - Implementa un **protocolo gossip** simple.
  - Cada nodo:
    - Escucha conexiones TCP entrantes (`:5300`).
    - Envía periódicamente sus registros a todos los peers conocidos.
  - Los mensajes pueden ser de tipo `update` para sincronizar registros.

- **`internal/dns.go`**:
  - Implementa el **resolver UDP**.
  - Maneja la estructura de mensajes DNS: header, question, answer.
  - Construye respuestas A para IPv4 y controla TTL dinámico.
  - Devuelve `NXDOMAIN` si el registro no existe o expiró.

- **`internal/models.go`**:
  - Define estructuras clave:
    - `Record`: registro DNS (nombre → IP), TTL y timestamp.
    - `GossipMessage`: mensaje de gossip para sincronizar registros.

- **`internal/logger.go`**:
  - Logger modular con colores para distinguir niveles de log (`INFO`, `WARN`, `ERROR`, `DEBUG`).

- **`dns/http_client.go` y `dns/register.go`**:
  - Helpers para registrar un nombre/IP en el DNS a través del API HTTP.
  - Cliente HTTP personalizado para resolver usando el DNS local.

- **`add_records.sh`**:
  - Script para agregar registros de ejemplo vía API.

- **Ejemplo de uso**:
  ```bash
  curl -s -X POST "http://localhost:6969/add" \
      -H 'Content-Type: application/json' \
      -d '{"name":"free.local","ips":["10.1.0.15"],"ttl":360}'
  dig @127.0.0.1 -p 8053 free.local
  ````
  
---

### `src/overlay`

Contiene el overlay network para el cliente BitTorrent.

* **`store.go`**:

  * Mantiene un mapeo `infoHash → providers`.
  * TTL configurable para limpiar peers obsoletos.
  * Métodos:

    * `Announce`: agrega o actualiza un provider.
    * `Merge`: fusiona providers de mensajes gossip.
    * `Lookup`: devuelve los peers más recientes.
    * `ToJSON`: serializa la lista de providers a JSON.

* **`gossip.go`**:

  * Implementa gossip unicast para propagar estado.
  * Permite:

    * Escuchar peers entrantes (`TCP`).
    * Responder consultas `lookup`.
    * Enviar actualizaciones periódicas a peers conocidos (`periodicGossip`).

---

### `src/client`

Contiene la lógica principal del cliente BitTorrent.

* **`main.go`**:

  * Carga metadatos del torrent.
  * Inicializa listener TCP para recibir conexiones de peers.
  * Configura overlay gossip si está habilitado.
  * Se conecta a peers locales y del tracker.
  * Ejecuta announces periódicos y de completación.
  * Maneja shutdown limpio.

* **`overlay_integration.go`**:

  * Inicializa el overlay con la lista de bootstrap peers.
  * Arranca el listener TCP de gossip.

---

## Redes Overlay en Docker Swarm

* Docker crea una **red virtual distribuida (overlay)** que conecta contenedores en distintos hosts.
* Cada host crea una **interfaz virtual VXLAN** (UDP 4789).
* La propagación de información se hace mediante **gossip unicast** en lugar de broadcast.
* Resolución de nombres:

  * Contenedor → DNS interno Docker `127.0.0.11`.
  * Docker traduce al servicio o contenedor destino.
* Limitaciones:

  * No hay broadcast UDP entre hosts.
  * El tráfico broadcast solo alcanza contenedores en el mismo host.
* Ventaja:

  * Red tolerante a fallos.
  * Actualización eventual de registros DNS distribuidos.

---

## Cómo funciona el gossip en tu DNS

1. Cada nodo tiene su **tabla de registros local**.
2. Al agregar o actualizar un registro, se notifica a los peers mediante gossip (`TCP 5300`).
3. Cada nodo replica la información de otros nodos.
4. TTL garantiza que registros expirados no se propaguen.
5. La consistencia es **eventual**, no instantánea.

---

## Mejoras posibles

1. **Seguridad**: autenticación entre nodos gossip y en la API.
2. **Redundancia**: persistir registros en disco para reinicios.
3. **Optimización de gossip**:

   * Limitar la cantidad de mensajes a peers más recientes.
   * Compresión de payloads.
4. **Balanceo de carga DNS**:

   * Round-robin entre múltiples IPs por nombre.
5. **Monitoreo**:

   * Métricas de registros, peers y latencia de gossip.
6. **Compatibilidad IPv6** y soporte completo de tipos de RR (CNAME, MX, etc.).
7. **Simulación en Docker Swarm** para probar escalabilidad del gossip.

---

## Bibliografía

* [Docker Networking Overview](https://docs.docker.com/network/)
* [Gossip Protocols in Distributed Systems](https://en.wikipedia.org/wiki/Gossip_protocol)
* [DNS Protocol RFC 1035](https://www.rfc-editor.org/rfc/rfc1035)
* [BitTorrent Protocol Specification](https://www.bittorrent.org/beps/bep_0003.html)

