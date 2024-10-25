//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative openapi/openapi.proto
package main
