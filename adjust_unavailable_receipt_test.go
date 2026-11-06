package busbar

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func finalSchemaResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestAdjustment503PreservesKnownNotAppliedReceipt(t *testing.T) {
	body := `{"operation":"adjust","operation_id":"action-1","actor":"operator","target":"key-1","fingerprint":"request-hash","reason_hash":"reason-hash","effect":"not_applied","applied":false,"recovery":"receipt","retry":"same_action_only","count_scale":6}`
	httpResponse := finalSchemaResponse(body)
	httpResponse.StatusCode = http.StatusServiceUnavailable
	httpResponse.Header.Set("x-busbar-adjust-effect", "not_applied")
	httpResponse.Header.Set("x-busbar-receipt-durable", "false")
	parsed, err := ParsePostAdjustResponse(httpResponse)
	if err != nil || parsed.JSON503 == nil {
		t.Fatalf("503 receipt response: %v", err)
	}
	receipt, err := parsed.JSON503.AsAdjustmentActionView()
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Effect != "not_applied" || receipt.Applied || receipt.Recovery != "receipt" || receipt.Retry != "same_action_only" {
		t.Fatalf("503 lost known action effect: %#v", receipt)
	}
	if parsed.HTTPResponse.Header.Get("x-busbar-adjust-effect") != "not_applied" || parsed.HTTPResponse.Header.Get("x-busbar-receipt-durable") != "false" {
		t.Fatal("503 effect and receipt durability headers lost")
	}
}

func TestAdjustment503PreservesOrdinaryUnavailableError(t *testing.T) {
	httpResponse := finalSchemaResponse(`{"error":{"code":"unavailable","message":"preparation failed"}}`)
	httpResponse.StatusCode = http.StatusServiceUnavailable
	parsed, err := ParsePostAdjustResponse(httpResponse)
	if err != nil || parsed.JSON503 == nil {
		t.Fatalf("503 error response: %v", err)
	}
	failure, err := parsed.JSON503.AsError()
	if err != nil || string(failure.Error.Code) != "unavailable" {
		t.Fatalf("503 error changed: %#v, %v", failure, err)
	}
}
