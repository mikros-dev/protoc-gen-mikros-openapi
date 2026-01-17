package openapi

import (
	"fmt"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

// Response describes a single response from an API Operation.
type Response struct {
	Description string            `yaml:"description,omitempty"`
	Content     map[string]*Media `yaml:"content"`
	schemaName  string
}

func parseOperationResponses(
	method *protobuf.Method,
	cfg *settings.Settings,
	converter *mapping.Message,
) map[string]*Response {
	codes := getMethodResponseCodes(method)
	if len(codes) == 0 {
		return nil
	}

	var (
		responses = make(map[string]*Response)
		name      = method.ResponseType.Name
		errorName = cfg.Error.DefaultName
	)

	if cfg.Mikros.UseOutboundMessages {
		name = converter.WireOutputToOutbound(name)
	}

	for _, code := range codes {
		refName := refComponentsSchemas + errorName
		if isSuccessCode(code) {
			refName = refComponentsSchemas + name
		}

		responses[fmt.Sprintf("%d", code.GetCode())] = &Response{
			Description: code.GetDescription(),
			Content: map[string]*Media{
				"application/json": {
					Schema: &Schema{
						Ref: refName,
					},
				},
			},
		}
	}

	return responses
}

func getMethodResponseCodes(method *protobuf.Method) []*mikros_openapi.Response {
	var (
		codes []*mikros_openapi.Response
	)

	if extensions := mikros_openapi.LoadMethodExtensions(method.Proto); extensions != nil {
		for _, c := range extensions.GetResponse() {
			codes = append(codes, c)
		}
	}

	return codes
}
