FROM golang:1.24-alpine AS build
WORKDIR /src
COPY src/go.mod ./
RUN go mod download
COPY src/ ./
RUN go build -o /out/auth-connector ./

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/auth-connector /app/auth-connector
ENV LISTEN_ADDR=:5557
EXPOSE 5557
ENTRYPOINT ["/app/auth-connector"]
