# BIBLIOGRAFÍA

# IMPLEMENTACIÓN DE UN SERVIDOR DE DNS DISTRIBUIDO

## REDES OVERLAY EN DOCKER SWARM

- Docker crea una **red virtual distribuida (overlay)** que conecta contenedores en distintos hosts como si compartieran una misma LAN.
- Cada host crea una **interfaz virtual VXLAN**, que encapsula los paquetes UDP sobre el puerto **4789**.
- Docker utiliza un **protocolo gossip** (basado en SWIM/Serf) para intercambiar la tabla de rutas y el estado de los endpoints de la red overlay.
- Cada contenedor obtiene una **IP virtual interna** que solo es válida dentro de la red overlay.

### Resolución de nombres interna

Cuando un contenedor intenta resolver un nombre como `tracker`, el proceso es el siguiente:

1. La consulta DNS dentro del contenedor se envía a `127.0.0.11`, el **DNS interno de Docker**.
2. Docker verifica si el nombre pertenece a un **servicio registrado** en la red overlay.
3. Docker resuelve `tracker` a la **IP virtual del servicio** (o a una lista de IPs si hay múltiples réplicas).
4. La **capa overlay** encapsula el paquete y lo envía a través del túnel VXLAN al host donde reside el contenedor destino.

---

## LIMITACIONES DEL BROADCAST EN REDES OVERLAY

- Las redes overlay **no soportan broadcast ni multicast reales**.
- En una red Swarm, los paquetes enviados a direcciones como `255.255.255.255` o `10.x.x.255` **solo llegan a los contenedores dentro del mismo host**.
- Esto ocurre porque:
  - VXLAN encapsula tráfico **unicast**, no broadcast.
  - Los bridges virtuales de Docker **filtran broadcast y multicast** por diseño.
  - La capa VXLAN opera en **nivel 3 (IP)**, no en nivel 2 (Ethernet).

> En resumen: **no es posible enviar mensajes broadcast UDP entre todos los nodos de un Swarm**. Solo se propagan dentro del host local.

---

## PROPAGACIÓN DE INFORMACIÓN ENTRE NODOS

Como alternativa al broadcast, Docker Swarm utiliza un **protocolo gossip unicast**:

- Cada nodo mantiene una lista parcial de peers.
- Envía actualizaciones de estado periódicamente a algunos de ellos.
- Los peers replican los mensajes con sus propios contactos.
- De esta forma, la información termina distribuyéndose por todo el cluster sin necesidad de broadcast.

Este enfoque:
- Reduce el tráfico total en la red.
- Permite tolerancia a fallos (si un nodo cae, el resto sigue propagando información).
- Es el mismo mecanismo que usa Docker internamente para sincronizar su DNS y el estado de los servicios.

---

## DISEÑO DE UN DNS DISTRIBUIDO

En un DNS distribuido diseñado para funcionar en Swarm:

1. **Cada nodo ejecuta un servidor DNS local** (por ejemplo, en un contenedor).
2. Cada uno mantiene una **tabla local de registros** `(nombre → IP)` correspondiente a los contenedores o servicios de su host.
3. Periódicamente, cada nodo **propaga los cambios** (altas, bajas o actualizaciones de registros) a otros nodos mediante **gossip unicast**.
4. Si un nodo se desconecta, los demás **mantienen su información local** hasta que expire su TTL.

Esto garantiza que:
- No exista un único punto de fallo.
- La resolución funcione incluso si un nodo cae.
- Los registros se mantengan eventualmente consistentes entre todos los nodos.

---

## FORMATO DEL MENSAJE DE DNS

### 1. Estructura general

Un mensaje DNS se divide en las siguientes secciones:

- **Header (12 bytes)**: contiene identificador, flags y conteos de preguntas/respuestas.
- **Question Section**: el dominio o nombre solicitado.
- **Answer Section**: la respuesta (si existe).
- **Authority Section / Additional Section**: opcionales, para delegación o datos complementarios.

### 2. Tipos de registros (RR Types)

| Tipo | Descripción |
|------|--------------|
| **A** | Dirección IPv4 |
| **AAAA** | Dirección IPv6 |
| **CNAME** | Alias de otro dominio |
| **MX** | Servidor de correo |
| **NS** | Servidor de nombres |
| **TXT** | Texto libre (informativo) |

### 3. TTL (Time to Live)

- Cada registro incluye un valor **TTL**, expresado en segundos.
- Indica cuánto tiempo puede mantenerse en caché antes de volver a consultarse.
- Es esencial en sistemas distribuidos para controlar la caducidad de información desactualizada.

### 4. Clases

- Generalmente se utiliza la clase **IN (Internet)**.
- Otras clases existen, pero son raras en la práctica.

### 5. Jerarquía y delegación

- Un servidor puede ser **autoridad** de un dominio o subdominio (por ejemplo, `example.com`).
- Si no conoce la respuesta, puede **reenviar la consulta** a otro servidor superior o raíz.
- Esto permite que los servidores formen una **jerarquía distribuida**.

---

## REQUISITOS MÍNIMOS DE UNA IMPLEMENTACIÓN DNS

Una implementación mínima y correcta debe:

1. Responder con los campos DNS adecuados.
2. Mantener una **zona local** de dominios e IPs.
3. Responder con **`NXDOMAIN`** si el dominio no existe.
4. Reenviar o preguntar a otros nodos si no conoce la respuesta.
5. Implementar **TTL** y mecanismos de expiración de caché.
6. En un entorno distribuido (como Docker Swarm), **sincronizar su zona** usando gossip unicast.

---

## CONCLUSIÓN

En redes overlay de Docker Swarm:
- El **broadcast** no está disponible entre nodos.
- La forma correcta de propagar información es mediante **gossip unicast**.
- Esto permite construir un **DNS distribuido tolerante a fallos**, donde cada nodo mantiene su propia copia de los registros y la actualiza gradualmente.

> En otras palabras, la replicación basada en gossip sustituye al broadcast, y la consistencia eventual sustituye a la sincronización instantánea.

