FROM golang:1.17

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY server.go ./

RUN go build

ENTRYPOINT ["./server"]
