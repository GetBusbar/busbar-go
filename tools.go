//go:build tools
// +build tools

// Package tools pins the code-generation toolchain so `go generate` uses a
// reproducible oapi-codegen version. Regenerate with: make generate
package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
)
