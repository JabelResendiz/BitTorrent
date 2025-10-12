FROM golang:1.22-alpine

WORKDIR /app

# Copia go.mod y go.sum desde la raíz del proyecto
COPY go.mod ./

# Descarga dependencias
RUN go mod tidy

# Copia todo el código
COPY . .

# Compila solo el tracker
RUN go build -o tracker2 ./tracker/cmd

EXPOSE 8080

CMD ["./tracker2"]
