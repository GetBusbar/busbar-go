# Busbar Admin API — Go SDK
#
# client.gen.go is generated from openapi.json and committed so consumers
# `go get` without regenerating. `make generate` re-derives it via the
# oapi-codegen version pinned in go.mod (tools.go).

.PHONY: generate build vet verify

generate:
	go generate ./...

build:
	go build ./...

vet:
	go vet ./...

verify: build vet
	@echo "build + vet OK"
