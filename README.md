# busbar-go (Go SDK)

[![codecov](https://codecov.io/gh/GetBusbar/busbar-go/branch/dev/graph/badge.svg)](https://codecov.io/gh/GetBusbar/busbar-go)

A typed Go client for the **Busbar Admin API** (`/api/v1/admin`).

Generated from the typed OpenAPI 3.1 schema in [`openapi.json`](./openapi.json)
with [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) v2, including
typed response models such as `InfoView` and `TopologyInfo`. Open configuration
values and response unions retain their schema-defined shapes.

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
- **Generated from:** busbar OpenAPI `info.version` `1.6.0`, the bundled
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
	fmt.Println("busbar version:", info.Version)        // -> "1.6.0"
	fmt.Println("pools:", info.Topology.Pools)
	fmt.Println("config version:", info.ConfigVersion)
}
```

> Prefer `Authorization: Bearer`? Set `req.Header.Set("Authorization", "Bearer YOUR_ADMIN_TOKEN")`
> instead — the admin API accepts either.

## Request and response safeguards

The default HTTP client follows redirects only within the original origin
(scheme, host, and port), keeping `x-admin-token` out of cross-origin requests.
A client supplied with `WithHTTPClient` owns its own redirect and timeout policy.
Use a context deadline for each operation.

Typed `WithResponse` calls and standalone response parsers buffer at most
`MaxResponseBodyBytes` (64 MiB) and return an error above that limit. For larger
responses, use the raw client methods and stream and close the response body.
`ResponseParseError` retains the HTTP response metadata on read or decode
failures and unwraps the original error for `errors.Is` and `errors.As`.

To unbind a key's group, pass `UpdateKeyReq{GroupNull: true}`. Leaving both
`Group` and `GroupNull` unset preserves the existing binding; a non-nil `Group`
rebinds it. Setting both returns a JSON encoding error.

Count fields use `encoding/json.Number` in Go to preserve their exact decimal
digits, including fractional counts and values beyond the exact integer range
of `float64`. The wire representation remains a JSON **number**, never a quoted
string. Use `count.String()` to inspect or retain that decimal value; converting
it with `Float64()` can lose precision. Count has six decimal places of scale;
this representation does not compute prices or change server-supplied money.

`PUT /config/settings` now has a generated request union. Its
`PutConfigSettingsJSONRequestBody` alias retains the union's JSON methods: use
`FromPutConfigSettingsJSONBody0` for the existing root-settings body or
`FromPutConfigSettingsJSONBody1` for document/catalog edits. Existing map-literal
call sites must construct the corresponding union variant. Legacy request JSON
is unchanged; both modes also support decoding JSON into the request union.

### Differences from 1.5.5 <!-- version-check: ignore -->

Admin-token authentication and key-group unbinding already existed in that
server release. This SDK now stops cross-origin redirects and bounds buffered
responses to protect credentials and memory. Explicit null group patches are
preserved; ordinary valid API requests retain their behavior.

## Regenerating the client

The committed client (`client.gen.go`) is generated from `openapi.json`. To
re-derive it:

```bash
make generate     # adapts Count fields, then runs the pinned oapi-codegen
```

The generator version is pinned in `go.mod` (via `tools.go`). Local templates in
`templates/`, based on that pinned generator, preserve the SDK safeguards and
the three-state key-group patch and config request union JSON methods when
regenerating. CI regenerates on
every PR/push and fails if the committed client drifts (`git diff --exit-code`).

`internal/generate` reads the canonical `openapi.json` without modifying it.
Generation follows `x-busbar-count-scale: 6` and the declared
`x-busbar-exact-number` profile in a temporary derived schema. Numeric Count
values use `json.Number`, including the legacy numeric arm of exact adjustments;
the decimal-string arm remains a string. Annotated 64-bit integers use `uint64`
or `int64`, including unsigned settings limits. Existing numeric wire forms
remain numeric, and unannotated numeric fields retain their declared types.
The committed `openapi.json` is never rewritten by generation.
Generated union accessors and response parsers also decode generic JSON values
as `json.Number`, so settings can be decoded, edited and re-encoded without
rounding large integers or exact decimals. Concrete typed numeric fields keep
their declared types.

`dev` is additionally kept auto-synced with busbarAI's `dev` branch spec by
`.github/workflows/regen-from-upstream.yml`, which deliberately never touches
`main` (the released-SDK branch) — see [RELEASING.md](./RELEASING.md) for how
that works and how version tags get cut (separately, by hand).

## License

Apache-2.0 © Busbar, Inc.
