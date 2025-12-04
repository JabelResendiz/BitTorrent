

> overlay/store.go

---

# 🧠 **1. `NewStore`**

Crea el almacén.
Nada sexy, pero sin esto no hay fiesta.

```go
func NewStore(ttl time.Duration) *Store {
    return &Store{
        records: make(map[string]map[string]ProviderMeta),
        ttl:     ttl,
    }
}
```

✔ Inicializa `records` vacío.
✔ Guarda la duración `ttl` para saber cuándo expirar providers viejos.

**Piensa en esto como: “arranca el mini-tracker vacío”.**

---

# 📣 **2. `Announce`**

Un peer dice: *“yo tengo este infoHash”*.
Esto lo registra.

```go
func (s *Store) Announce(infoHash string, p ProviderMeta) error
```

Pasos clave:

1. Revisa que `infoHash` y `p.Addr` no vengan vacíos.
2. Bloquea escritura (`s.mu.Lock()`).
3. Crea el submapa si no existe.
4. Actualiza `LastSeen = ahora`.
5. Inserta/actualiza el provider.

**Traducción:**

> *“Este peer vive, tiene este archivo, y lo vi ahorita mismo.”*

Esto es básicamente como el "announce" de un tracker BitTorrent clásico.

---

# 🔗 **3. `Merge`**

Esto es lo que lo vuelve **Gossip-friendly**.

```go
func (s *Store) Merge(infoHash string, providers []ProviderMeta)
```

Significa:

> *“Te paso mis providers; mezcla los tuyos con los míos y quédate con el más reciente.”*

Pasos:

1. Lock de escritura.
2. Para cada provider recibido:

   * Si no existe → agrégalo.
   * Si existe pero el nuevo tiene `LastSeen` más grande → reemplázalo.

Esto evita que un peer viejo sobrescriba datos frescos → típico en gossip anti-entropy.

---

# 🔍 **4. `Lookup`**

Esto es lo que un peer llama cuando quiere saber:

> “¿Quién tiene este infoHash **ahora**?”

```go
func (s *Store) Lookup(infoHash string, limit int) []ProviderMeta
```

Pasos:

1. Lock de lectura.
2. Obtiene providers para ese `infoHash`.
3. Filtra los que ya expiraron:
   `LastSeen >= now - ttl`
4. Ordena por `LastSeen DESC` (los más frescos primero).
5. Aplica `limit` si hay.

Resultado:
**Una lista de peers vivos, reciente y priorizada.**

---

# 🧪 **5. `ToJSON`**

Exporta los providers (para debug, API, o enviar por gossip).

```go
func (s *Store) ToJSON(infoHash string) ([]byte, error)
```

* Lock de lectura.
* Si no hay nada, retorna `[]`.
* Si hay, convierte los `ProviderMeta` en JSON.

---

# ⚡ **En resumen, sin vueltas**

Tu `Store` implementa:

| Método       | Rol           | Explicación corta                                 |
| ------------ | ------------- | ------------------------------------------------- |
| **NewStore** | Constructor   | Crea la tabla en memoria con TTL                  |
| **Announce** | Ingreso       | Registra/actualiza un provider para un infoHash   |
| **Merge**    | Gossip        | Mezcla providers remotos conservando el más nuevo |
| **Lookup**   | Consulta      | Devuelve peers vivos y ordenados por frescura     |
| **ToJSON**   | Serialización | Exporta la tabla para transmisión/log/debug       |

Este patrón es **el mismo que usan Kademlia, trackers híbridos y overlays gossip**:
un *key → set of providers*, actualizado por timestamps y con merges monotónicos.

-----------------

> overlay/gossip.go

# 🧩 1. `wireMsg`

Es el mensaje que viaja por la red.

```go
type wireMsg struct {
	Type      string
	InfoHash  string
	Providers []ProviderMeta
	Limit     int
}
```

Sirve para tres tipos de mensajes:

* `"announce"` → *“Hola, tengo este infohash.”*
* `"gossip"` → *“Esto es lo que sé sobre este infohash.”*
* `"lookup"` → *“Dame tus providers para este infohash.”*

---

# 🏠 2. `Overlay`

Este struct es el **motor del overlay distribuido**.

```go
type Overlay struct {
	Store      *Store        ← aquí vive la info (providers)
	peers      []string      ← peers conocidos iniciales (bootstrap)
	listenAddr string         ← dónde escucho conexiones
	stopCh     chan struct{}  ← señal para apagar el sistema
}
```

Piensa en `Overlay` como:

> “El nodo P2P completo: escucha, anuncia, mergea y hace gossip”.

---

# 🚀 3. `NewOverlay`

Inicializa un overlay con un TTL de **90 segundos** para los providers.

```go
func NewOverlay(listenAddr string, peers []string) *Overlay
```

Así cada nodo expira providers viejos y no se llena de basura.

---

# 📡 4. `Start`

Arranca dos procesos:

1. **Listener TCP** (`serveListener`)
2. **Loop de gossip periódico** (`periodicGossip`)

Esto da vida al nodo P2P.

---

# 🔊 5. `serveListener`

Acepta conexiones entrantes mientras no se cierre el overlay.

---

# 🎧 6. `handleConn`

Aquí entra TODO lo que llega por TCP.

Flujo:

1. Decode JSON → `wireMsg`
2. Según `Type`:

   * `"gossip"` o `"announce"` → merge al `Store`
   * `"lookup"` → responder con tus providers locales
3. Cerrar conexión

Este es el “API TCP” del overlay.

---

# 🔁 7. `periodicGossip`

Cada 8 segundos:

```go
o.gossipOnce()
```

Esto es lo que mantiene **consistencia eventual** entre nodos.

---

# 🌐 8. `gossipOnce`

El chismoso del vecindario.

Hace:

1. Saca todos los infohashes del store
2. Para cada peer bootstrap
3. Para cada infohash
4. Envía un mensaje `"gossip"` con **toda la lista de providers**

El nodo remoto en `handleConn` va a:

```go
o.Store.Merge(infoHash, providers)
```

Y listo, ambos se sincronizan.

---

# 📤 9. `sendWireMsg`

Función común para enviar mensajes por TCP.
Sin respuestas, sin bloqueos, sin dramas: fire-and-forget.

---

# 📣 10. `Announce`

Registra localmente un provider y además lo empuja a los peers.

```go
o.Store.Announce(...)
msg := wireMsg{Type: "announce", ...}
sendWireMsg()
```

Esto propaga rápidamente nueva información.

---

# 🔎 11. `Lookup`

Esta es la operación más interesante.

Hace **multi-fuente lookup**:

1. Mira localmente:

   ```go
   local := o.Store.Lookup(infoHash)
   ```
2. Contacta **máximo 3 peers** (`fanout = 3`)
3. Funde todo en un mapa por address (para evitar duplicados)
4. Ordena por frescura (`LastSeen`)
5. Retorna la lista, limitada si hace falta

Es como un DHT reducido:

> “Dame lo que yo sé y lo que unos cuantos vecinos sepan”.

---

# 📥 12. `queryPeerLookup`

Envía a un peer:

```json
{ "type":"lookup", "info_hash":..., "limit":... }
```

y recibe un array `[]ProviderMeta`.

Sin enredos.

---

# 🎯 RESUMEN EN 20 PALABRAS

Es un **overlay P2P simple** que:

* guarda providers (`Store`)
* sincroniza vía gossip
* anuncia cambios
* responde lookups
* mergea con consistencia eventual
* usa TCP + JSON

Un mini-tracker descentralizado sin DHT completa.

---
