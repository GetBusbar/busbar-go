// Command generate derives Go exact-number mappings without changing openapi.json.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	input, err := os.ReadFile("openapi.json")
	if err != nil {
		return err
	}
	derived, err := deriveSchema(input)
	if err != nil {
		return err
	}
	if output := os.Getenv("BUSBAR_GO_DERIVED_SCHEMA"); output != "" {
		if err := os.WriteFile(output, derived, 0600); err != nil {
			return err
		}
	}
	dir, err := os.MkdirTemp("", "busbar-go-generate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "openapi.json")
	if err := os.WriteFile(path, derived, 0600); err != nil {
		return err
	}
	cmd := exec.Command("go", "run", "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen", "--config=oapi-codegen.yaml", path)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return rewriteJSONDecoders("client.gen.go")
}

func deriveSchema(input []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(input))
	// Schema bounds and examples must retain their exact decimal spelling too.
	decoder.UseNumber()
	var schema any
	if err := decoder.Decode(&schema); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("expected one OpenAPI JSON document")
	}
	if err := adaptCounts(schema); err != nil {
		return nil, err
	}
	return json.MarshalIndent(schema, "", "  ")
}

func adaptCounts(value any) error {
	switch node := value.(type) {
	case map[string]any:
		// String enums and constants already exclude non-string JSON values.
		// oapi-codegen otherwise emits interface{} for enum-only response
		// headers, which its string header binder cannot decode.
		if _, typed := node["type"]; !typed {
			if _, stringConstant := node["const"].(string); stringConstant {
				node["type"] = "string"
			}
			if values, ok := node["enum"].([]any); ok && len(values) > 0 {
				stringsOnly := true
				for _, value := range values {
					if _, ok := value.(string); !ok {
						stringsOnly = false
						break
					}
				}
				if stringsOnly {
					node["type"] = "string"
				}
			}
		}
		if err := adaptExactNumber(node); err != nil {
			return err
		}
		if scale, annotated := node["x-busbar-count-scale"]; annotated {
			if scale != json.Number("6") {
				return fmt.Errorf("unsupported x-busbar-count-scale: %v", scale)
			}
			if node["type"] != "number" {
				return fmt.Errorf("count schema must retain numeric JSON wire type")
			}
			node["x-go-type"] = "json.Number"
			// The generated response parsers already import encoding/json. Adding
			// x-go-type-import duplicates that import in oapi-codegen v2.8.
			delete(node, "x-go-type-import")
		}
		for _, child := range node {
			if err := adaptCounts(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range node {
			if err := adaptCounts(child); err != nil {
				return err
			}
		}
	}
	return nil
}

// adaptExactNumber follows the server's lossless-number-token-v1 metadata.
// Builtin 64-bit integers remain numeric JSON; exact Count numeric arms use
// json.Number, including the legacy numeric arm of a string-or-number union.
func adaptExactNumber(node map[string]any) error {
	value, annotated := node["x-busbar-exact-number"]
	if !annotated {
		return nil
	}
	metadata, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("x-busbar-exact-number must be an object")
	}
	if format, present := metadata["integer_format"]; present {
		if !schemaHasType(node, "integer") {
			return fmt.Errorf("exact integer metadata requires an integer JSON wire type")
		}
		switch format {
		case "uint64", "uint":
			setGoType(node, "uint64")
		case "int64":
			setGoType(node, "int64")
		default:
			return fmt.Errorf("unsupported exact integer format: %v", format)
		}
		return nil
	}
	if bits, present := metadata["bits"]; present {
		if bits != json.Number("64") || !schemaHasType(node, "integer") {
			return fmt.Errorf("unsupported exact integer width or wire type: %v", bits)
		}
		switch metadata["signed"] {
		case true:
			setGoType(node, "int64")
		case false:
			setGoType(node, "uint64")
		default:
			return fmt.Errorf("exact integer signedness must be a boolean")
		}
		return nil
	}
	if bits, present := metadata["mantissa_bits"]; present {
		if bits != json.Number("128") || metadata["scale"] != json.Number("6") || metadata["signed"] != true {
			return fmt.Errorf("unsupported exact Count range or scale")
		}
		if schemaHasType(node, "number") {
			setGoType(node, "json.Number")
			return nil
		}
		if schemaHasType(node, "string") {
			return nil
		}
		if branches, ok := node["anyOf"].([]any); ok {
			found := false
			for _, branch := range branches {
				if child, ok := branch.(map[string]any); ok && schemaHasType(child, "number") {
					setGoType(child, "json.Number")
					found = true
				}
			}
			if found {
				return nil
			}
		}
		return fmt.Errorf("exact Count metadata requires a number, string, or numeric union")
	}
	return fmt.Errorf("unsupported x-busbar-exact-number metadata")
}

func schemaHasType(node map[string]any, wanted string) bool {
	if node["type"] == wanted {
		return true
	}
	if types, ok := node["type"].([]any); ok {
		found := false
		for _, value := range types {
			if value == wanted {
				found = true
			} else if value != "null" {
				return false
			}
		}
		return found
	}
	return false
}

func setGoType(node map[string]any, name string) {
	node["x-go-type"] = name
	if name == "uint64" || name == "int64" {
		node["format"] = name
	}
	// All generated clients already import encoding/json for their parsers.
	// Explicit x-go-type-import would duplicate that import in oapi-codegen v2.8.
	delete(node, "x-go-type-import")
}

// Rewrite decoder calls in the owning generation recipe so union accessors,
// additional-property models and response parsers share the same lossless JSON
// behavior. Concrete numeric destination types retain encoding/json semantics.
func rewriteJSONDecoders(path string) error {
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, path, input, parser.ParseComments)
	if err != nil {
		return err
	}
	replaced := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Unmarshal" {
			return true
		}
		name, ok := selector.X.(*ast.Ident)
		if ok && name.Name == "json" {
			call.Fun = ast.NewIdent("unmarshalJSONExact")
			replaced++
		}
		return true
	})
	if replaced == 0 {
		return fmt.Errorf("generator emitted no JSON decoder calls to adapt")
	}
	var output bytes.Buffer
	if err := format.Node(&output, positions, file); err != nil {
		return err
	}
	return os.WriteFile(path, output.Bytes(), 0644)
}
