# ADR 0008: Sign the device catalogue index with ed25519

- Status: accepted
- Date: 2026-09-15

## Context

`catalog.Build` is deterministic and hash-bound, but a replaced `catalog-index.json` on the device was not authenticated by this project. Handhelds must verify the index offline. A long-lived private key is not yet stored in release automation.

## Decision

Sign the exact index bytes with ed25519 and store the signature in a sidecar `catalog-index.json.sig`. Device CI generates a per-build key, signs the index, deletes the private key, and compiles the public key into the CLI and GUI with `-ldflags`. `catalog.Load` verifies the sidecar when a public key is compiled in. Local unsigned builds still load for review. Device ZIP packaging requires the sidecar.

## Consequences

### Positive

- Replacing only `catalog-index.json` on a device build fails verification.
- Signing uses the Go standard library. The CLI stays CGO-free.

### Negative

- An independent index update needs a new app build until a long-lived production key exists.
- Local `go run` and `make catalogue` stay unsigned and do not authenticate the file on disk.
