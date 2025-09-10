FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY src/main.go ./

RUN go build -o /app/main .

CMD ["/app/main"]
