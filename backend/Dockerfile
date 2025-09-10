FROM golang:1.24-alpine

WORKDIR /app

COPY src/go.mod ./
COPY src/go.sum ./
RUN go mod download

COPY src/ ./src/
WORKDIR /app/src
RUN go build -o /app/main main.go

CMD ["/app/main"]
