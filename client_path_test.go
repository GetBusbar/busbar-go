package busbar

import "testing"

func TestPathDotSegmentsRemainResourceNames(t *testing.T) {
	for _, name := range []string{".", ".."} {
		req, err := NewGetExportNameRequest("https://example.test/", name)
		if err != nil {
			t.Fatal(err)
		}
		want := "/api/v1/admin/export/" + name
		if req.URL.Path != want {
			t.Errorf("name %q resolved to %q, want %q", name, req.URL.Path, want)
		}
	}
}
