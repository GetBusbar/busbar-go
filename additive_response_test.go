package busbar

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func additiveJSON(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value map[string]interface{}
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestTelemetryExactStringsAndErrors(t *testing.T) {
	const body = `{"process_boot_nonce":"0123456789abcdef0123456789abcdef","uptime_since_start_ms":"9007199254740993","captured_at_unix_ms":"1793836800123","upper_bounds_ms":[1,5,10,25,50,100,250,500,1000,2500,5000,10000,30000,60000,120000,null],"identities":[{"plane":"llm","phase":"LlmProducerDrive","pool":"default","ingress_protocol":"openai","count":"9007199254740993","errors":"1","client_errors":"2","sum_us":"18446744073709551615","bins":["0","0","0","0","0","0","9007199254740993","0","0","0","0","0","0","0","0","0"]}],"omitted_identities":"0","truncated":false,"dropped":{"invalid_duration":"0","integer_overflow":"0","series_capacity":"0","boot_nonce_unavailable":"0","histogram_capacity":"0","unregistered_labels":"0"}}`
	response := func(status int, body string) *http.Response {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
	}
	parsed, err := ParseGetTelemetryResponse(response(200, body))
	if err != nil || parsed.JSON200 == nil {
		t.Fatalf("telemetry decode: %v", err)
	}
	encoded, err := json.Marshal(parsed.JSON200)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(additiveJSON(t, encoded), additiveJSON(t, []byte(body))) {
		t.Fatalf("telemetry strings or nullable bounds changed: %s", encoded)
	}
	for _, status := range []int{401, 503} {
		code := "unavailable"
		if status == 401 {
			code = "unauthorized"
		}
		parsed, err := ParseGetTelemetryResponse(response(status, `{"error":{"code":"`+code+`","message":"collection unavailable"}}`))
		if err != nil {
			t.Fatal(err)
		}
		var envelope *Error
		if status == 401 {
			envelope = parsed.JSON401
		} else {
			envelope = parsed.JSON503
		}
		if envelope == nil || string(envelope.Error.Code) != code || envelope.Error.Message != "collection unavailable" || parsed.StatusCode() != status {
			t.Fatalf("telemetry error lost its typed envelope: %+v", parsed)
		}
	}
}

func TestConfigSettingsModesPreserveWireAndResponses(t *testing.T) {
	cases := []struct{ name, request, response string }{
		{"legacy", `{"limits":{"request_body_max_bytes":5000123}}`, `{"applied":true,"config_version":8,"settings":{"limits":{"request_body_max_bytes":5000123}},"reload_to_apply":[]}`},
		{"document", `{"document_patch":{"set":{"/limits/request_body_max_bytes":5000123,"/rate_card/llm/input":0.125,"/tls":null},"remove":["/providers/retired"]},"providers_patch":{"set":{"/example/base_url":"https://api.example.com"}},"reset_document_paths":["/limits"],"reset_providers_paths":["/example"]}`, `{"applied":true,"persisted":true,"config_version":9,"configuration":{"config_version":9,"state":"restart_pending","source":"installed_generation","config_locked":false,"values":{"limits":{"request_body_max_bytes":5000123}},"reload_to_apply":["listen"]},"reload_to_apply":["listen"]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" || r.URL.Path != "/api/v1/admin/config/settings" || r.Header.Get("If-Match") != `"7"` {
					t.Errorf("unexpected config request: %s %s %s", r.Method, r.URL.Path, r.Header.Get("If-Match"))
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(additiveJSON(t, raw), additiveJSON(t, []byte(tc.request))) {
					t.Errorf("request changed: %s", raw)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.response)
			}))
			defer server.Close()
			client, err := NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			var body PutConfigSettingsJSONRequestBody
			if err := json.Unmarshal([]byte(tc.request), &body); err != nil {
				t.Fatal(err)
			}
			etag := `"7"`
			got, err := client.PutConfigSettingsWithResponse(context.Background(), &PutConfigSettingsParams{IfMatch: &etag}, body)
			if err != nil || got.JSON200 == nil {
				t.Fatalf("config union response: %v", err)
			}
			encoded, err := json.Marshal(got.JSON200)
			if err != nil {
				t.Fatal(err)
			}
			actual, expected := additiveJSON(t, encoded), additiveJSON(t, []byte(tc.response))
			for key, value := range expected {
				if !reflect.DeepEqual(actual[key], value) {
					t.Errorf("response %s changed: %s", key, encoded)
				}
			}
		})
	}
}

func TestTrustExpectedDigestAndConflict(t *testing.T) {
	digest := "reviewed-digest"
	body := TrustKeyReq{Key: "instance/peer", ExpectedDigest: &digest}
	for _, operation := range []string{"approve", "revoke"} {
		t.Run(operation, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/api/v1/admin/trust/"+operation {
					t.Errorf("wrong trust operation")
				}
				raw, _ := io.ReadAll(r.Body)
				if !reflect.DeepEqual(additiveJSON(t, raw), map[string]interface{}{"key": "instance/peer", "expected_digest": digest}) {
					t.Errorf("digest guard missing: %s", raw)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(409)
				_, _ = io.WriteString(w, `{"error":{"code":"conflict","message":"sighting changed"}}`)
			}))
			defer server.Close()
			client, err := NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			var envelope *Error
			if operation == "approve" {
				result, err := client.PostTrustApproveWithResponse(context.Background(), body)
				if err != nil {
					t.Fatal(err)
				}
				envelope = result.JSON409
			} else {
				result, err := client.PostTrustRevokeWithResponse(context.Background(), body)
				if err != nil {
					t.Fatal(err)
				}
				envelope = result.JSON409
			}
			if envelope == nil || envelope.Error.Code != "conflict" {
				t.Fatalf("trust conflict missing: %+v", envelope)
			}
		})
	}
	body.ExpectedDigest = nil
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := additiveJSON(t, encoded)["expected_digest"]; present {
		t.Fatalf("omitted digest sent as null: %s", encoded)
	}
}

func TestNamedSettingsEditAndSafeReadback(t *testing.T) {
	const bodyText = `{"edit":{"set":{"endpoint":"https://collector.example","optional":null},"remove":["obsolete"]}}`
	const responseText = `{"name":"metrics","module":"prometheus","settings_keys":["endpoint","optional","secret"],"configuration":{"config_version":9,"source":"installed_configuration","state":"configured","settings_public":{"endpoint":"https://collector.example"}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/api/v1/admin/export/metrics/settings" {
			t.Errorf("wrong named edit endpoint")
		}
		raw, _ := io.ReadAll(r.Body)
		if !reflect.DeepEqual(additiveJSON(t, raw), additiveJSON(t, []byte(bodyText))) {
			t.Errorf("named settings edit changed: %s", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, responseText)
	}))
	defer server.Close()
	client, err := NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	var body PatchExportNameSettingsJSONRequestBody
	if err := json.Unmarshal([]byte(bodyText), &body); err != nil {
		t.Fatal(err)
	}
	result, err := client.PatchExportNameSettingsWithResponse(context.Background(), "metrics", nil, body)
	if err != nil || result.JSON200 == nil {
		t.Fatalf("named response: %v", err)
	}
	encoded, err := json.Marshal(result.JSON200)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(additiveJSON(t, encoded)["configuration"], additiveJSON(t, []byte(responseText))["configuration"]) {
		t.Fatalf("configured public settings lost: %s", encoded)
	}
}
