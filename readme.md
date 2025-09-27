
# BitTorrent

[link wiki](https://wiki.theory.org/BitTorrentSpecification)
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

