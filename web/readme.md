docker build --no-cache -t bittorrent-web:local .

# si el error persiste

## Eliminar node_modules
rm -rf node_modules .pnpm-store

## Reinstalar dependencias
pnpm install

## Reconstruir Docker
docker build -t bittorrent-web:local .