FROM golang:1.22-alpine

WORKDIR /app/src

# Copia go.mod desde src/
COPY src/go.mod ./

# Descarga dependencias
RUN go mod download

# Copia todo el código fuente
COPY src/ ./

# Compila el tracker
RUN go build -o /app/tracker ./tracker/cmd

WORKDIR /app

EXPOSE 8080 9090

# Usar ENTRYPOINT para que los argumentos CLI se pasen al binario
ENTRYPOINT ["/app/tracker"]
