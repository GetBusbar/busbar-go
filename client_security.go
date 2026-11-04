package busbar

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// The admin token uses a custom header that net/http otherwise forwards even
// across origins. Keep redirects within the original origin, including its port.
func newDefaultHTTPClient() *http.Client {
	return &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if len(via) == 0 {
			return nil
		}
		original := via[0].URL
		if !strings.EqualFold(req.URL.Scheme, original.Scheme) || !strings.EqualFold(req.URL.Hostname(), original.Hostname()) || effectivePort(req.URL) != effectivePort(original) {
			return http.ErrUseLastResponse
		}
		return nil
	}}
}

// MaxResponseBodyBytes bounds buffered response parsing. Use the raw Client
// methods to stream responses larger than this limit.
const MaxResponseBodyBytes int64 = 64 << 20

func readResponseBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, MaxResponseBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxResponseBodyBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", MaxResponseBodyBytes)
	}
	return data, nil
}

// URL resolution removes literal dot segments. Escape these two names before
// inserting already-escaped path parameters into the operation URL.
func escapeDotPathParameter(value string) string {
	switch value {
	case ".":
		return "%2E"
	case "..":
		return "%2E%2E"
	default:
		return value
	}
}

// Normalize media types for the generator's JSON dispatch without mutating the
// caller's original HTTP response or exposing a normalized header to callers.
func responseForJSONParsing(response *http.Response) *http.Response {
	contentType := response.Header.Get("Content-Type")
	if contentType == "application/json" {
		return response
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	normalized := ""
	if err == nil && (mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")) {
		normalized = "application/json"
	}
	// Avoid copying headers when the generated dispatch already agrees.
	if (normalized != "") == strings.Contains(contentType, "json") {
		return response
	}
	clone := *response
	clone.Header = response.Header.Clone()
	clone.Header.Set("Content-Type", normalized)
	return &clone
}

func effectivePort(endpoint *url.URL) string {
	if port := endpoint.Port(); port != "" {
		return port
	}
	switch strings.ToLower(endpoint.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

// ResponseParseError preserves HTTP metadata when reading or decoding a response
// fails. Its message contains the status and cause, never the response body or
// headers. HTTPResponse.Body has been closed by the parser.
type ResponseParseError struct {
	HTTPResponse *http.Response
	Err          error
}

func (e *ResponseParseError) Error() string {
	if e.HTTPResponse == nil {
		return fmt.Sprintf("parse HTTP response: %v", e.Err)
	}
	return fmt.Sprintf("parse HTTP response (status %d): %v", e.HTTPResponse.StatusCode, e.Err)
}

// Unwrap preserves errors.Is and errors.As access to the original cause.
func (e *ResponseParseError) Unwrap() error { return e.Err }
