FROM golang:1.22-alpine

WORKDIR /app/src

# Copia go.mod desde src/
COPY src/go.mod ./

# Descarga dependencias
RUN go mod download

# Copia todo el código fuente
COPY src/ ./

# Compila el cliente
RUN go build -o /app/client ./client/cmd

WORKDIR /app

EXPOSE 6881
EXPOSE 9091

# Define el ENTRYPOINT para que los argumentos se pasen al binario
ENTRYPOINT ["/app/client"]
