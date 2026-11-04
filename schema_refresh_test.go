package busbar

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRefreshedAuditHeadEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/admin/audit/head" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"algorithm":"ed25519","head":null,"next_seq":9007199254740993,"recipe":"audit-v1","signature_domain":"audit"}`)
	}))
	defer server.Close()
	client, err := NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.GetAuditHeadWithResponse(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if response.JSON200 == nil || response.JSON200.NextSeq != 9007199254740993 || response.JSON200.Head != nil {
		t.Fatalf("unexpected audit head: %+v", response.JSON200)
	}
}

func TestRefreshedTrustApproveRequest(t *testing.T) {
	request, err := NewPostTrustApproveRequest("https://example.test/", TrustKeyReq{Key: "plane/counterparty/item"})
	if err != nil {
		t.Fatal(err)
	}
	defer request.Body.Close()
	if request.Method != http.MethodPost || request.URL.Path != "/api/v1/admin/trust/approve" {
		t.Fatalf("unexpected request %s %s", request.Method, request.URL.Path)
	}
	var body TrustKeyReq
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Key != "plane/counterparty/item" {
		t.Fatalf("key = %q", body.Key)
	}
}

func TestNamedMutationResponsesPreserveRestartMetadata(t *testing.T) {
	const body = `{"name":"events","module":"file","settings_keys":[],"reload_to_apply":["export:events"],"note":"restart required"}`
	for _, tc := range []struct {
		name  string
		parse func(*http.Response) (any, error)
	}{
		{"put", func(response *http.Response) (any, error) {
			parsed, err := ParsePutExportNameResponse(response)
			if err != nil {
				return nil, err
			}
			return parsed.JSON200, nil
		}},
		{"patch", func(response *http.Response) (any, error) {
			parsed, err := ParsePatchExportNameSettingsResponse(response)
			if err != nil {
				return nil, err
			}
			return parsed.JSON200, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := tc.parse(&http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(parsed)
			if err != nil {
				t.Fatal(err)
			}
			var preserved struct {
				ReloadToApply []string `json:"reload_to_apply"`
				Note          string   `json:"note"`
			}
			if err := json.Unmarshal(encoded, &preserved); err != nil {
				t.Fatal(err)
			}
			if len(preserved.ReloadToApply) != 1 || preserved.ReloadToApply[0] != "export:events" || preserved.Note != "restart required" {
				t.Fatalf("typed mutation response lost restart metadata: %s", encoded)
			}
		})
	}
}
