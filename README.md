# busbar-go (Go SDK)

A typed Go client for the **Busbar Admin API** (`/api/v1/admin`).

Generated from the typed OpenAPI 3.1 schema in [`openapi.json`](./openapi.json)
with [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) v2, so every
response is a real struct (`InfoView`, `TopologyInfo`, ...) — not
`interface{}`/`json.RawMessage`.

```
import busbar "github.com/GetBusbar/busbar-go"
```

## Versioning

The SDK carries its **own** semantic version, independent of the busbar server /
OpenAPI `info.version` (currently `1.4.0`). This first cut is tagged **`v0.1.0`**.
It targets the frozen, additive-only `/api/v1/admin` surface.

Go modules publish via git tags — there is no registry token. `go get` fetches
straight from this repo:

```bash
go get github.com/GetBusbar/busbar-go@v0.1.0
```

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

	resp, err := client.GetApiV1AdminInfoWithResponse(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if resp.JSON200 == nil {
		log.Fatalf("status %d: %s", resp.StatusCode(), resp.Body)
	}

	// resp.JSON200 is *InfoView — TYPED. .Version is string, .Topology is a struct.
	info := resp.JSON200
	fmt.Println("busbar version:", info.Version)        // -> "1.4.0"
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

## License

Apache-2.0 © Busbar, Inc.
