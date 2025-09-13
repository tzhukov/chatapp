FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ .
# Build a static binary for linux/amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags "-s -w" -o /out/main main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/main /app/main
EXPOSE 8080
CMD ["/app/main"]
