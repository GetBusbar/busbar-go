# busbar-go (Go SDK)

[![CI](https://github.com/GetBusbar/busbar-go/actions/workflows/ci.yml/badge.svg)](https://github.com/GetBusbar/busbar-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/GetBusbar/busbar-go/branch/main/graph/badge.svg)](https://codecov.io/gh/GetBusbar/busbar-go)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A typed Go client for the **Busbar Admin API** (`/api/v1/admin`).

Generated from the typed OpenAPI 3.1 schema in [`openapi.json`](./openapi.json)
with [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) v2, so every
response is a real struct (`InfoView`, `TopologyInfo`, ...) — not
`interface{}`/`json.RawMessage`.

```
import busbar "github.com/GetBusbar/busbar-go"
```

## Versioning

- **SDK version:** the newest `v0.x.y` tag on this repo, listed on the
  [releases page](https://github.com/GetBusbar/busbar-go/releases). The SDK carries its
  own semantic version, independent of the busbar server it talks to. No number is
  written down here on purpose: a Go module's version is a git tag, with no in-repo
  manifest to check a README claim against, so a hardcoded one would only ever be as
  fresh as the last person who remembered to edit it.
- **Generated from:** busbar OpenAPI `info.version` `1.5.3`, the bundled
  [`openapi.json`](./openapi.json).

It targets the frozen, additive-only `/api/v1/admin` surface. `dev` is kept
continuously synced with busbarAI's `dev` branch spec (see
[RELEASING.md](./RELEASING.md)).

The spec version above is not maintained by hand alone: CI runs
`.github/check-readme-versions.py`, which fails the build if it stops matching
`openapi.json`. It went stale silently once, so now it cannot.

Go modules publish via git tags — there is no registry token. `go get` fetches
straight from this repo:

```bash
go get github.com/GetBusbar/busbar-go@latest   # or @vX.Y.Z to pin an exact release
```

### History

`v0.2.0` was the breaking release: the busbar `1.5.0` spec added `operationId`s, which
renamed every generated symbol.

## Usage

The admin API authenticates with an `x-admin-token` header. Attach it with a
request editor, then call `GET /info` (see [`examples/info`](./examples/info)):

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	busbar "github.com/GetBusbar/busbar-go"
)

func main() {
	client, err := busbar.NewClientWithResponses(
		"http://localhost:8081",
		busbar.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("x-admin-token", "YOUR_ADMIN_TOKEN")
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.GetInfoWithResponse(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if resp.JSON200 == nil {
		log.Fatalf("status %d: %s", resp.StatusCode(), resp.Body)
	}

	// resp.JSON200 is *InfoView — TYPED. .Version is string, .Topology is a struct.
	info := resp.JSON200
	fmt.Println("busbar version:", info.Version)        // -> "1.5.3"
	fmt.Println("pools:", info.Topology.Pools)
	fmt.Println("config version:", info.ConfigVersion)
}
```

> Prefer `Authorization: Bearer`? Set `req.Header.Set("Authorization", "Bearer YOUR_ADMIN_TOKEN")`
> instead — the admin API accepts either.

## Regenerating the client

The committed client (`client.gen.go`) is generated from `openapi.json`. To
re-derive it:

```bash
make generate     # runs the pinned oapi-codegen via go run
```

The generator version is pinned in `go.mod` (via `tools.go`). CI regenerates on
every PR/push and fails if the committed client drifts (`git diff --exit-code`).

`dev` is additionally kept auto-synced with busbarAI's `dev` branch spec by
`.github/workflows/regen-from-upstream.yml`, which deliberately never touches
`main` (the released-SDK branch) — see [RELEASING.md](./RELEASING.md) for how
that works and how version tags get cut (separately, by hand).

## License

Apache-2.0 © Busbar, Inc.
