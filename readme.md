
# BitTorrent



[link wiki](https://wiki.theory.org/BitTorrentSpecification)
## Requisitos

## Run

### 🛰️ Run Tracker
Abre una terminal en la raíz del proyecto(src) y ejecuta:

```bash
go run tracker/cmd/main.go
```

Esto lanzará e tracker HTTP escuchando en `localhost:8080`

Si todo va bien, se debe ver algo como:

```pgsql
2025/10/10 23:41:15 tracker listening on :8080 interval=1800s data=tracker_data.json
```
### 💻 Run Client
Abre otra terminal y ejecutar:

```bash
go run client/cmd/main.go
```

La salida esperada será algo como :

```perl
Tracker request:  http://localhost:8080/announce?info_hash=%BA%4E...
Tracker responde:  map[complete:0 incomplete:1 interval:1800 peers:]
```

En `tracker_data.json` se pueden ver los peers registrados por el tracker.

## Metainfo File (.torrent)
Los archivos `.torrent` están bencodeados, que es un formato simple para codificar información (no en texto plano), más fácil que parsear que un XML y un JSON. 

- Todos los datos en un archivo metainfo están bencodeados. Un `.torrent` es un diccionario bencodeado que contiene las siguientes claves (todas las cadenas en UTf-8):
    - info : diccionario que describe el/los archivos del torrent. Puede ser :
        - single-file (un solo archivo)
        - multi-file (multiples archivos)
    - announce: la URL del tracker (cadena)
    - announce-list : (opcional) extensión para compatibilidad retroactica(lista de lista de cadenas)
    - creation date : (opcional) fecha de creación en formato UNIX epoch (entero, segundos desde 1-ene-1970)
    - comment: (opcional) comentarios libres del autor cadena
    - created by: (opcional) nombre y versión del programa que creó el `.torrent`.
    - encoding : (opcinal) formato de codificación de cadenas usado para la parte de pieces dentro del diccionario `info`


## Metainfo File (.torrent)

Los archivos `.torrent` están **bencodeados**.  
Un `.torrent` es un **diccionario** con estas claves principales (todas cadenas en **UTF-8**):

- **info**: diccionario que describe los archivos del torrent
  - *single-file*: un archivo
  - *multi-file*: múltiples archivos
- **announce**: URL del tracker (cadena)
- **announce-list**: *(opcional)* lista de trackers alternativos
- **creation date**: *(opcional)* fecha en formato UNIX epoch (entero, segundos desde 1-ene-1970)
- **comment**: *(opcional)* texto libre del autor
- **created by**: *(opcional)* cliente que generó el torrent
- **encoding**: *(opcional)* formato de cadenas en `info`

---

### Diccionario `info`

#### Campos comunes

- **piece length**: bytes por pieza (entero)
- **pieces**: concatenación de hashes SHA-1 de 20 bytes
- **private**: *(opcional)*
  - `1`: solo usa trackers listados
  - `0` o ausente: puede usar DHT, PEX, etc.

---

#### Modo Single-File
Un torrent que **descarga un solo archivo**. El `info` contiene directamente el tamaño y nombre del archivo. Todas las piezas corresponden únicamente a ese archivo.
- **name**: nombre del archivo (cadena)
- **length**: tamaño en bytes (entero)
- **piece length**: tamaño de las piezas
- **pieces** : SHA1 concatenados
- **md5sum**: *(opcional)* hash MD5

```json
"info": {
  "name": "archivo.txt",
  "length": 123456,
  "piece length": 16384,
  "pieces": "<SHA1 concatenados>"
}
```

---

#### Modo Multi-File
Un torrent que contiene **varios archivos dentro de una carpeta raíz**. En este caso, `info` tiene una **lista de archivos `files`** en lugar de un `length` único.

```json
"info": {
  "name": "Mi_Carpeta",
  "piece length": 32768,
  "pieces": "<SHA1 concatenados>",
  "files": [
    {
      "length": 1024,
      "path": ["subdir1", "file1.txt"]
    },
    {
      "length": 2048,
      "path": ["subdir2", "file2.mp4"]
    }
  ]
}
```

---

### Notas sobre piezas

- Tamaño suele ser potencia de 2
- Históricamente: torrent ≤ 75 KB
- Recomendado: piezas ≤ 512 KB para torrents de 8–10 GB
- Usuales: 256 KB, 512 KB, 1 MB
- Todas las piezas mismo tamaño salvo la última
- En multi-file, los archivos se concatenan → piezas pueden cruzar archivos
- Cada pieza se representa por un hash SHA-1 (20 bytes) en `pieces`

---

### Ejemplo de archivo .torrent (bencodeado)

#### Versión "single-file"


- `announce`: URL del tracker.  
- `announce list` : (lista de lista de URL)
- `creation date`: fecha
- `comment` : comentarios
- `created by` : autor
- `info`: diccionario con detalles del archivo:
  - `length`: tamaño en bytes del archivo.  
  - `name`: nombre del archivo.  
  - `piece length`: tamaño en bytes de cada pieza.  
  - `pieces`: concatenación de hashes SHA-1 de 20 bytes cada uno.  
  - `private`  

#### Versión JSON 

```json
{
  "announce": "http://tracker.example.com/announce",
  "announce-list": [
    ["http://tracker.example.com/announce"],
    ["http://backuptracker.example.net/announce"]
  ],
  "creation date": 1695600000,
  "comment": "Ejemplo de torrent educativo",
  "created by": "ChatGPT TorrentMaker v1.0",
  "info": {
    "name": "archivo.txt",
    "length": 123456,
    "piece length": 16384,
    "pieces": "<concatenación binaria de SHA1 de 20 bytes cada uno>",
    "private": 0
  }
}
```
#### Misma versión bencode
```bencode
d
8:announce23:http://tracker.example.com/announce
13:announce-list
  ll
    23:http://tracker.example.com/announce
  e
  l
    33:http://backuptracker.example.net/announce
  e
e
13:creation datei1695600000e
7:comment27:Ejemplo de torrent educativo
10:created by27:ChatGPT TorrentMaker v1.0
4:infod
  4:name11:archivo.txt
  6:lengthi123456e
  12:piece lengthi16384e
  6:pieces40:<SHA1 pieza 1><SHA1 pieza 2>...
  7:privatei0e
e
e
```

---


## Tracker

El tracker no guarda “quién tiene cada pieza del archivo”, sino quién está participando en ese torrent en general.

📌 En concreto:

El torrent file se identifica por su info_hash (SHA1 del diccionario info).

Cuando un cliente hace announce al tracker, le dice:

“Estoy en el swarm del torrent con info_hash = X”

Y pasa su peer_id, ip, port, y su estado (started, stopped, completed).

El tracker anota: “Peer Y está en el torrent X”.

Opcionalmente, lleva un conteo de cuántos peers están completos (seeders) y cuántos no (leechers).

Pero no sabe si tienes la pieza #5 o la #200. Eso lo sabe solo cada peer, y te lo dice luego vía bitfield o mensajes have.

- El **tracker** es un servicio HTTP/HTTPS que responde a sus solicitudes **HTTP GET**. Las solicitudes incluyen métricas de los clientes que ayudan al tracker a mantener estadísticas generales sobre el torrent. 

- La respeta incluye una **lista de pares (peers)** que ayuda al cliente a participar en el torrent.


### Parámetros de la solicitud del cliente al tracker

Los parámetros usados en la solicitud **GET** del cliente -> tracker son los siguientes :

- **info_hash :** hash SHA1 de 20 bytes (codificado en URL) del valor de la clave `info` del archivo metainfo . Este valor será un diccionario bencodeado, dado lo que define la clave `info`

- **peer_id :** cadena de 20 bytes (codificada en URL) usada como ID único del cliente, generado al inicio. Puede ser cualquier valor; no hay reglas para generarlo , único en la máquina local (incluir ID de proceso + timestamp)

- **port :** el número de puerto en el que escucha el cliente. Los puertos reservados para BitTorrent son 6881-6889. Si no puede usar uno de ellos, algunos clientes simplemente abandonan

- **uploaded :** total de bytes subidos desde el cliente envión el evento `started` al tracker (en ASCII base 10)

- **downloaded :** total de bytes descargados desde el evento `started` (en ASCII base 10)

- **left :** número de bytes que el cliente aún debe descargar (en ASCII base 10), es decir lo que falta para tener el 100% del torrent

- **compact :** si se establece en `1`, el cliente acepta una respuesta compacta. En ese caso , la lista de peers es reemplzada por una cadean binaria de 6 bytes por peer:
    - 4 bytes -> IP en network by order
    - 2 bytes -> puerto en network byte order
    - ALgunos trackers solo aceptan compact = 1, rechazando otras solicitudes

- **no_peer_id :** indica que el tracker puede omitir el campo `peer_id` en la lista de peers. Ignorado si `compact = 1`

- **event :** si se especifica , debe ser uno de:
    - `started` : el primer request al tracker debe incluirlo
    - `stopped` : cuando el cliente se apaga ordenadamente
    - `completed` : cuando el torrent termina al 100% (pero no si ya estaba completo al inicio)
    - `vacío` : igual a no especificarlo (solicitudes regulares)

- **ip (opcional) :** la IP real del cliente. Puede estar en formato IPv4o IPv6

- **numwant (opcional) :**numero de peers que el cliente desea recibir. Puede ser `0`. Si se omite , por defecto suelen ser de `50 peers`

- **key (opcional) :** identificador extra no compartido con otros peers. Sirve para que un cliente demuestre su identidad si cambia de IP

- **trackerid (opcional) :** si el tracker devolvió un `tracker_id` en una announce previa, debe reenviarse aquí

```http
GET /announce?info_hash=%12%34%56%78%9A%BC%DE%F1%23%45%67%89%AB%CD%EF%12%34%56%78%9A
&peer_id=-AZ2060-6wfG2wk6wWLc
&port=6881
&uploaded=0
&downloaded=0
&left=123456789
&compact=1
&event=started HTTP/1.1
Host: tracker.ejemplo.com:6969
User-Agent: MiBitTorrentCliente/1.0
Connection: close
```

### Parámetros de la respuesta del tracker

EL tracker responde con un documento `text/plain` que consiste en un diccionario bencodeado con las siguientes claves:

- **failure reason :** si está presente, entonces no puede haber ninguna otra clave. EL valor es un mensjae de error legible por humanos que explica por qué falló la solicitud (string)

- **warning message (nuevo, opcional):** Similar a failure reason, pero la respuesta aún se procesa normalmente. El mensaje de advertencia se muestra igual que un error.

- **interval :** intervalo en segundos que el cliente debe esperar entre envíos regulares de solicitud al tracker.

- **min interval (opcional) :** intervalo mínimo de announce. SI está presente, los clientes no deben reenviar announce con más frecuencia que este.

- **tracker id :** una cadena que el cliente debe enviar en sus próximos announces. Si está ausente y un announce previo envió un `tracker id` , no se debe descartar el valor antiguo; se debe seguir usando.

- **complete :** número de peers con el archivo completo, es decir, `seeders` (entero)

- **incomplete :** Número de peers no-seeders (`leechers`)

- **peers (model diccionario) :** El valor es una lista de diccionarios, cada uno con las siguientes claves :
    - **peer id :** ID auto seleccionado del peer, como se descrbió en la solicitud al tracker (string)
    - **ip :** dirección ip del peer , ya sea IPv6, IPv4 o nombres DNS(string)
    - **port :** Número de puerto del peer (entero)

- **peers (modelo binario) :** En lugar de usar el modelo de diccionario descrito arriba, el valor de `peers` puede ser una cadena que consiste en múltiplos de 6 bytes. Los primeros 4 bytes son las dirección IP y los últimos 2 bytes son el número de puerto. Todo en notación de red (big endian).


Respuesta en modo "diccionario" (no compacta)

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 210

d
  8:intervali1800e
  8:completei12e
  10:incompletei34e
  5:peersl
    d2:ip13:192.168.1.210:porti6881e7:peer id20:-ABCD1234567890ABCDEe
    d2:ip13:192.168.1.211:porti6882e7:peer id20:-XYZ9876543210XYZabcde
  ee
e
```

Respuesta en modo "compacto" (más usado en la práctica)

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 68

d8:intervali1800e8:completei12e10:incompletei34e5:peers12:\xC0\xA8\x01\xD2\x1A\xE1\xC0\xA8\x01\xD3\x1A\xE2e
```


### Convención de Tracker Scrape

Los `trackers` (opcional) soportan otra forma de petición , que consulta el estado de un `torrent` en particular (o de todos los `torrents`) que el tracker este gestionando. A eso se le conoce como `"la página de scrape"` porque automatiza el proceso, de otro modo tedioso, la página de estadísticas del tracker.

- Es importante para cuando debamos hacer un interfaz gráfica , consola con info detallada
- Optimizaciones de decisiones internas (clientes más avanzados), antes de unirte a un torrent, se pregunta al tracker si vale la pena entrar, así evitas descargar un `torrent` con 0 seeders.

La URL del `scrape` también se utiliza el método HTTP GET , similar al descrito anteriormente. Sin embargo, la URL base es diferente. Para obtenerla :

1. Comenzar con la URL de announce
2. Localizar la última '/' en ella
3. Si el texto inmediatamente después de ese '/' no es un 'announce', se considerará que el tracker no soporta la convención scrape
4. Si sí lo es, se sustituye 'announce' por 'scrape' para obtener la URL del scrape

La URL de `scrape` puede complementarse con el parámetro opcional `info_hash`, un valor de 20 bytes. Esto restringe el informe del tracker a ese torrent en particular (o de lo contrario, devuelve estadísticas de todos los torrents que el tracker gestiona, no es muy recomendable porque ocupa más carga y ancho de banda). 

La respuesta de este método HTTP GET es un documento `text/plain` que consiste en un diccionario codificado en bencode, con las siguientes claves:

- **files:** un diccioanrio que contiene un par clave/valor por cada torrent del que existan estadísticas
  - Cada clave es un `info_hash` binario de 20 bytes
  - El valor asociado es otro diccionario con :
    - **complete:** numero de pares con el archivo completo (semillas o seeders)(entero)
    - **downloaded:** numero total de veces que el tracker registró una finalización(`event= completed`, es decir un cliente terminó de descargar el torrent)
    - **incomplete:** numero de pares sin el archivo completo(leechers)(entero)
    -**name(opcional):** nombre interno del torrent, especificado por el campo `name` en la sección `info` del archivo `.torrent`

#### Respuesta del tracker al scrape
```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 68

d5:filesd20:....................(info_hash)d8:completei5e10:downloadedi50e10:incompletei10eeee
```

#### Solicitud del scrape

```http
GET /scrape?info_hash=%12%34%56%78... HTTP/1.1
Host: tracker.ejemplo.com:6969
```


## Peer wire protocol (TCP)

- un cliente debe mantener información de estado para cada conexión que tenga con un peer remoto.
  - **chocked:** indica si el peer remoto ha "estrangulado" a este cliente. Cuando sucede , le notifica que no responderá a solicitudes hasta que sea desestrangulado. El cliente no debe enviar solicitudes de bloques y debe considerar todas las solicitudes pendientes como descartadas por el peer remoto
  - **interested:** indica si el peer remoto está interesado en algo que este cliente ofrece. Es una notificación de que el peer remoto empezará a pedir bloques cuando el cliente deje de estrangularlo
  - ***am_choking:*** este cliente está estrangulando al peer (inicial = 1)
  - ***am_interested:*** este cliente está interesaod en el peer (inicial = 0)
  - ***peer_choking:*** peer está estrangulando al cliente (inicial = 1)
  - ***peer_interested:*** peer está interesado en este cliente (peer_interested = 0)
  - un bloque se **descarga** cuando el (am_interested = 1) y (peer_choking=0)
  - un bloque se **sube** cuando el (peer_interested = 1) y (am_choking = 0)

### Handshake 
> es un mensaje obligatorio y debe ser el primer mensaje transmitido por el cliente. Tiene una longitud de **(49 +  len(pstr)) bytes**.

```php-template
handshake: <pstrlen><pstr><reserved><info_hash><peer_id>
```
- **pstrlen:** longitud de la cadena `pstr` , como un bytes bruto único
- **pstr:** identificador de cadena del protocolo
- **reserved:** 8 bytes reservados. Todas las implementaciones actuales usan 0. Cada bit en estos bytes puede usarse para cambiar el comportameinto del protocolo
- **info_hash:** Es el mismo `info_hash` que se transmite en las solicitudes al tracker (20 bytes)
- **peer_id:** cadena de 20 bytes usadas como ID único del cliente (el mismo que se transmite en las solicitudes al tracker)
- en la versión 1.0 del protcolo BitTorrent, `pstrlen=19` y `pstr="BitTorrent protocol"`


- el iniciador de la conexión debe transmitir su `handshake` inmediatamente. EL receptor puede esperar el `handshake` del inciador si puede servir múltiples torrent simultaneamente. Sin embargo, el receptor debe responder tan pronto vea la parte `info_hash` del `handshake` (el peer id presumiblemente se enviará después de que el receptro envíe su `handshake`). La función de verificación NAT del tracker no envía el campo `peer_id` del handshake.
- Si uyn cliente recibe un `handshake` con un `info_hash` que no está sirviendo actualmente , debe cerrar la conexión
- Si el iniciador de la conexión recibe un `handshake` cuyo `peer_id` no coincide con el peer_id esperado , debe cerrar la conexión. Es decir se espera que el `peer_id` drecibido por el tracker coincida con el del handshake.

### Tipo de datos
todos los enteros en el protocolo de peer por cable se codifican como valores de cuatro bytes en **big_endian**. Esto incluye el prefijo de longitud en todos los mensajes que vienen después del handshake

### Flujo de Mensajes
El protocolo consiste en un handshake inicial. Despues , los peers se comunican mediante un intercambio de mensajes prefijados con su longitud. El prefijo de longitud es un entero.

### Mensajes
> Todos los mensajes restantes en el protocolo toman la forma de `<prefijo de longitud><ID de mensaje><carga útil>`. El prefijo de longitud es un valor de 4 bytes en big-endian. EL ID del mensaje es un solo byte decimal. La carga útil depende del mensaje

- ***keep-alive (len=0000):*** mensaje de 0 bytes, especificado con el prefijo de longitud en 0. No tiene ID de mensaje ni carga útil. Los peers pueden cerrar una conexion si no reciben mensajes (keep-alive o cualquier otro) durante un período de tiempo , por lo que se debe enviar un keep-alive para mantener la conexión viva si no se ha enviado ningún comando durante un tiempo determinado. Suele ser de 2 minutos.

- ***choke (<len=0001><id=0>):*** el mensaje choke tiene longitud fija y no tiene carga útil

- ***unchoke (<len= 0001><id=1>):*** tiene longitud fija y sin carga

- ***interested (<len=0001><id=2>):*** lo mismo que los otros dos

- ***not interested (<len=00001><id=3>):*** lo mismo

- ***have (<len= 00005><id=4><piece index>):*** longitud fija. La carga útil es el índice basado en cero de un piece que acaba de ser descargado y verificado mediante hash

- ***bitfiel (<len=0001+X><id=5><bitfield>):*** mensaje bitfiedl solo puede enviarse inmediatamente después de completar el handshake, antes de enviar cualquier otro mensaje. Es opcional y no es necesario si un cliente no tien piezas.

- ***request(<len=0013><id=6><begin><length>):*** mensaje tiene longitud fija y se usa para solicitar un bloque. La carga útil contiene: index(indece de la pieza), begin(desplazamiento de bytes dentro de la pieza), length(longitud solicitada)

- ***piece(<len=0009+X><id=7><index><begin><block>):*** mensaje de longitud variable, donde X es la longitud del bloque. La carga útil contiene : index (indice de la pieza), begin(desplazamineto dentro de al pieza), block(datos,  subconjunto de al pieza especificada).

- ***cancel(<len=0013><id=8><index><being><length>):*** longitud fija, y se usa para cancelar solicitudes de bloques. La carga útil es identifca a la del mensaje request. Se usa tipicamente durante la fase End Game.

- ***port(<len=0003><id=9><listen-port>):*** mensaje port es envidado por versiones recientes de Mainline que implementar un tracker DHT. listen-port (puerto donde el nodo DHT del peer escucha), este peer debe ser insertado en la tabla de ruteo local si se soporta DHT.


## Algoritmos

Estrategias internas de los clientes BitTorrent para mejorar rendimiento y eficiencia. El protocolo base define qué mensajes se pueden enviar(interested, request, piece, ...) pero no cuándo ni cuántos mandar.

### Cola (Queuing)

- ***Problema:*** imaginemos que cada bloque de 16 KB se decarga, y recién cuando termina uno, el cliente pide el siguiente. Eso significa esperar un round trip completo (el tiempo entre enviar solicitud y recibir el bloque). En redes con alta latencia o mucho ando de banda , ese tiempo muerto desperdicia capacidad de descarga.

- ***Solución:*** los clientes mantienen una cola de solicitudes pendientes ("request outstanding"). Así mientras descargan un bloque , ya tienen varios más pedidos. Cuando uno llega, el siguiente ya está en camino. Es mejor hacer 10 request en paralelo que 1 sola request, para mantener el canal lleno y aprovechar el ancho de banda. 


### Super Seeding

- Cuando eres el primer seed (el que tiene el archivo completo), la meta es dsitribuir piezas únicas lo más eficientemente posible. 
- La idea es que el seed finge no tener todas las piezas y solo "anuncia" a los peers una pieza cada vez. Eso así para compartir piezas diferentes con cada peers con el objetivo de que luego entre ellos se lo intercambien. Reduce así la cnatidad total de datos que el seed necesita subir para que se genere otro seed. 
- Solo se recomineda al sembrar un torrent nuevo (cuando es el primero)


### Estrategia de descarga de piezas

- Los clientes pueden elegir descargar piezas en orden aleatorio. Una estrategia mejor es descargar las peizas en orden de rareza creciente (rarest first)
- El cliente puede determinar esto manteniendo el `bitfield` inicial de cada peer y actualizandolo con cada mensaje `have`.
- luego puede descargar las piezas que aparezcan con menor frecuencia en esos bitfield.
- Cualquiera estrategia rarest first debería incluir algo de aleatorización entre las piezas menos comunes, ya que si muchos clientes intentan descargar la misma pieza más rara, se producirá el efecto contrario.


### End Game
- Cuando una descarga está casi completa, hay una tendencia a que los últimos bloques lleguen lentamente
- Para acelerar esto , el cliente envía solicitudes de todos los bloques faltantes a todos sus peers.
- Para evitar que esto se vuelva ineficiente , el cliente también envía un mensaje cancel a todos los demás cada vez que llega un bloque.

[overhead del protocolo](http://hal.inria.fr/inria-00000156/en)

### Choking y Optimistic unchoking

- El protocolo usa `choking` para controlar con quién subes datos. No puedes subir a todos a la vez sin romper TCP, así que subes solo a algunos. 
- La regla básica es : cada 10 seg, eliges 4 peers que te suben más rápido (unchokeas). los demás bloqueas sus solicitudes. Así implementar un tit-for-tat "tu me das velocidad , yo te doy velocidad"
- la versión optimizada es cada 30seg eliges uno al azar (aunque no te esté dadno nada) para probar si podría ser mejor que tus actaules 4. Si resulta ser rápido, entra en el grupo y otro sale.


### Anti-snubbing
- A veces un peer deja de enviarte piezas (te ignora)
- Si pasa más de 1 min sin recibir datos, el cliente lo marca como "snubbed" y deja de subirle, salvo en el caso de `optimistic unchoke`
- EL objetivo es evitar perder tiepo con peers que no colaboran.


# Extensiones oficiales del protocolo

## Extensiones Fast Peers
- bit reservado : el tercer bit menos significativo del 8° byte reservado `reserved[7] |= 0x04`
- esto permite acelerar el arranque de un peer nuevo en el swarm (la rede de pares compartiendo un torrent).
- Normalmente , si un peer esta choked, no puede pedir piezas
- Con esta extensión , ciertos peers pueden descargar piezas específicas aunque estén choked, lo que acelera el sincronización inicial

## DHT
- bit reserado `reserved[7] |= 0x01` (último bir del octavo byte).
- permite descubir peers sin necesidad de un tracker centralizado. Cada peer se convierte en un nodo de una red DHT, donde se guarda información sobre qué oeers tiene qué torrents.
- el sistema sigue funcionando si el tracker cae
- los peers se buscan entre sí usanod una table hash distribuida (basada en kademlia)
- BEP-32 agrega soporte para IPV6

## Connection Obfuscation( Message Stream Encryption- MSE)

- no tiene bit reservado específico
- permite cifrar o camuflar las conexiones BitTorrent para evitar que los proveedores de internet, detecten o limiten el trafico torrent. 
- ofusca el handsahke y los mensajes del protocolo
- ayuda a evadir el trafic shapping o throttling
- mejora la privacidad

## WebSeeding
- no usa bit reservado
- permite que un servidor HTTP actúe como seed(fuente de datos), además de los peers normales
- en resumen, pueeds descargar partes del torrent desde un servidor web, no solo de otros usuarios

## Extension Protocol 
- bit reservado `reserved[5] = 0x10` 8caurto bit mas signficativo del sexto byte
- define una forma genérica para anuciar y negociar extensiones entre cliente
- cada extensión adicional (por ejemplo DHT, metadata exchange, peer exchange) se anuncia y negocia mediante este protocolo

## extensión negotiation protocol
- bite servado el 47 y 48
- permite decidir que extensio usar si ambos peers soportan varias.
- evita conflictos cuando dos cliete implementan diferentes sistema de extensión.

## bittorrent location aware-protocol
- bit reservado: 21
- permite que los peers tomen en cuanta la ubicación geográfica de otros peers. De esa forma pueden prefierir descargar de peers más cercanos, reduciendo latencia y carga de red

## SimpleBT extension protoc0l
- bit reservado primer byte `0x01`
- agrega intercambio de informacion de peers y estadísticas de conexión.
- fue usado en versiones antiguas de SImpleBot

## BitComet Extension Protocol
- bit reservado primeros dos bytes `ex`
- usado para intecambir informacion adicional(autenticacipon, estadísticas, mensaje del chat)
- no está docuemntado oficialmente, se conoce por ingeniería inversa