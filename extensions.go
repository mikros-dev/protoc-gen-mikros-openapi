//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative openapi/mikros_openapi.proto
package main
