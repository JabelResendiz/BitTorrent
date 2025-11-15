

- `conn.go`: Funciones para abrir/cerrar la conexión TCP con un peer (`NewPeerConn`, `Close`). Solo maneja la **conexión física**

- `handshake.go`: Función `HandShake`que envía y valida el handshake inicial del protocolo

- `message.go`: 





# Flujo
handshake -> enviar `interested` -> ReadLoop -> esperar `unchoke` -> enviar `request` -> recibir `piece`



# Conn.go

- funciones `NewPeerConn` y, abre una conexion TCP con otro peer (addr) , con un timeout de 5 segundo. Luego devuelve una estructura PeerConn, que representa una conexion activa con un peer remoto.`NewPeerConnFromConn` , se usa para conexiones entrantes , cuando otro peer se conecta a mi, no para las que yo inicio

# Handshake.go

- antes de que dos peers comiencen a intermcabiar piezas de archivo, deben establecer una conexion TCP y realiza un handshake. Este handsahek sirve para 
1. Confirmar que ambos estan hablando el protocolo Bittorrent
2. Verificar que ambos etan interesados en el mismo torrent(comparando el info_hash)
3. Identificar el peer remoot (peer_id)


1. Se crea una conexion TCP -> NewPeerConn
2. Se realiza el handshake -> HandShake
3. Se intercambia el bitfield inicial
4. Se envian mensajes Interested/Unchoke
5. El cliente solicita piezas con MsgRequest
6. El peer responde con MsgPiece
7. Manager coordina los peers y actauliza el progreso global