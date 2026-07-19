// Command info calls GET /api/v1/admin/info and prints the TYPED version.
//
//	go run ./examples/info
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	busbar "github.com/GetBusbar/busbar-go"
)

func main() {
	endpoint := envOr("BUSBAR_ENDPOINT", "http://localhost:8081")
	token := os.Getenv("BUSBAR_ADMIN_TOKEN")

	// Send the admin credential as the `x-admin-token` header on every request.
	client, err := busbar.NewClientWithResponses(
		endpoint,
		busbar.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("x-admin-token", token)
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
		log.Fatalf("unexpected status %d: %s", resp.StatusCode(), resp.Body)
	}

	// resp.JSON200 is *InfoView — TYPED. .Version is a string, .Topology is a struct.
	info := resp.JSON200
	fmt.Println("busbar version:", info.Version) // -> "1.4.0"
	fmt.Println("pools:", info.Topology.Pools)
	fmt.Println("config version:", info.ConfigVersion)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
