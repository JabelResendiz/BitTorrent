

- `conn.go`: Funciones para abrir/cerrar la conexión TCP con un peer (`NewPeerConn`, `Close`). Solo maneja la **conexión física**

- `handshake.go`: Función `HandShake`que envía y valida el handshake inicial del protocolo

- `message.go`: 





# Flujo
handshake -> enviar `interested` -> ReadLoop -> esperar `unchoke` -> enviar `request` -> recibir `piece`