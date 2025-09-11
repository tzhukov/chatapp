# dex_password

Minimal Dex gRPC PasswordConnector service scaffold.

- LISTEN_ADDR: ":5557" by default
- POSTGRES_DSN: postgres DSN; the chart defaults to the Bitnami release in the chatapp namespace.

Testing:
- Unit tests: go test ./...
- Integration test for Postgres repo requires TEST_PG_DSN env.