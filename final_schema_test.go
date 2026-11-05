package busbar

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestFinalSchemaPluginConfigurationRoundTrip(t *testing.T) {
	for _, configuration := range []string{
		`{"root_keys":["llm"],"root_source":"statement","declaration_source":"linked_statement","declaration_availability":"present","sections":[{"name":"llm","declaring":true,"consumed":false,"required":true,"selects":true},{"name":"providers","declaring":false,"consumed":true,"required":false,"selects":false}],"claims":[],"settings_schema_scope":"lifecycle","input_schema_complete":false,"configuration_shape":"declared_sections","host_policy_paths":[]}`,
		`{"root_keys":[],"root_source":"unavailable","declaration_source":"unavailable","declaration_availability":"unavailable","sections":[],"claims":[],"settings_schema_scope":"lifecycle","input_schema_complete":false,"configuration_shape":"declared_sections","host_policy_paths":[]}`,
	} {
		t.Run(configuration, func(t *testing.T) {
			response := func(body string) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
			}
			schemaBody := `{"name":"llm","schema":{"type":"object","properties":{"label":{"type":"string"}}},"schema_error":null,"trust":"trusted","source":"statement","configuration":` + configuration + `}`
			schema, err := ParseGetPluginsFileSchemaResponse(response(schemaBody))
			if err != nil || schema.JSON200 == nil {
				t.Fatalf("plugin schema metadata parse: %v", err)
			}
			catalogBody := `{"items":[{"name":"llm","type":"plane","loader":"compiled-in","active":null,"target":null,"has_schema":true,"configuration":` + configuration + `}],"next_cursor":null}`
			catalog, err := ParseGetPluginsResponse(response(catalogBody))
			if err != nil || catalog.JSON200 == nil {
				t.Fatalf("plugin catalog metadata parse: %v", err)
			}
			for name, value := range map[string]any{"schema": schema.JSON200, "catalog": catalog.JSON200} {
				encoded, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				actual := additiveJSON(t, encoded)
				if name == "catalog" {
					items, ok := actual["items"].([]interface{})
					if !ok || len(items) != 1 {
						t.Fatalf("catalog items changed: %s", encoded)
					}
					actual = items[0].(map[string]interface{})
				}
				configurationJSON, err := json.Marshal(actual["configuration"])
				if err != nil {
					t.Fatal(err)
				}
				want := additiveJSON(t, []byte(configuration))
				got := additiveJSON(t, configurationJSON)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s lost configuration ownership/availability or false flags: %s", name, encoded)
				}
			}
		})
	}
}

func TestFinalSchemaLinkedDeclarationQuery(t *testing.T) {
	source := GetPluginsFileSchemaParamsSource("linked_statement")
	if !source.Valid() {
		t.Fatal("native schema linked_statement source was not generated")
	}
	request, err := NewGetPluginsFileSchemaRequest("https://example.test", "llm", &GetPluginsFileSchemaParams{Source: &source})
	if err != nil {
		t.Fatal(err)
	}
	if request.URL.Path != "/api/v1/admin/plugins/llm/schema" || request.URL.Query().Get("source") != "linked_statement" {
		t.Fatalf("linked declaration query changed: %s", request.URL)
	}
}
