FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod ./

RUN go mod tidy

COPY . .

RUN go build -o dns-bin ./dns/cmd

CMD ["./dns-bin"]
