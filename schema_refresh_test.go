package busbar

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
