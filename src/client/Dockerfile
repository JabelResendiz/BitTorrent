FROM golang:1.22-alpine

WORKDIR /app

# Copia go.mod y go.sum desde la raíz del proyecto
COPY go.mod ./

# Descarga dependencias
RUN go mod tidy

# Copia todo el código
COPY . .

# Compila el cliente
RUN go build -o client-bin ./client/cmd

# Define el ENTRYPOINT para que los argumentos se pasen al binario
ENTRYPOINT ["./client-bin"]
