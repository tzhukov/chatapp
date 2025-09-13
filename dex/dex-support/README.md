# dex-support

Support resources for Dex in dev: authproxy ingress, headers, and Secrets.

## Local usage

Provide your local certs via values that reference files in this chart directory (or relative paths) and install:

```bash
helm upgrade --install dex-support ./dex/dex-support -n chatapp \
  --set secrets.create=true \
  --set-file secrets.caCrtFile=../../ca.crt \
  --set-file secrets.tlsCrtFile=../../ingress.local.pem \
  --set-file secrets.tlsKeyFile=../../ingress.local-key.pem
```

Or create a `local-values.yaml` (gitignored):

```yaml
secrets:
  create: true
  caCrtFile: ../../ca.crt
  tlsCrtFile: ../../ingress.local.pem
  tlsKeyFile: ../../ingress.local-key.pem
```

then:

```bash
helm upgrade --install dex-support ./dex/dex-support -n chatapp -f dex/dex-support/local-values.yaml
```

This chart does not deploy Dex itself; use the main Dex chart with `dex/dex-values.yaml`.