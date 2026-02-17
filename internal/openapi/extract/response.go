package extract

import (
	"fmt"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

func parseOperationResponses(
	method *protobuf.Method,
	cfg *settings.Settings,
	converter *mapping.Message,
) map[string]*spec.Response {
	codes := lookup.LoadMethodResponseCodes(method)
	if len(codes) == 0 {
		return nil
	}

	var (
		responses = make(map[string]*spec.Response)
		name      = method.ResponseType.Name
		errorName = cfg.Error.DefaultName
	)

	if cfg.Mikros.UseOutboundMessages {
		name = converter.WireOutputToOutbound(name)
	}

	for _, code := range codes {
		refName := refComponentsSchemas + errorName
		if lookup.IsSuccessResponseCode(code) {
			refName = refComponentsSchemas + name
		}

		responses[fmt.Sprintf("%d", code.GetCode())] = &spec.Response{
			Description: code.GetDescription(),
			Content: map[string]*spec.Media{
				"application/json": {
					Schema: &spec.Schema{
						Ref: refName,
					},
				},
			},
		}
	}

	return responses
}

func parseComponentsResponses(pkg *protobuf.Protobuf, cfg *settings.Settings) map[string]*spec.Response {
	responses := make(map[string]*spec.Response)

	for _, method := range pkg.Service.Methods {
		for _, response := range parseMethodComponentsResponses(method, cfg) {
			responses[response.SchemaName] = response
		}
	}

	return responses
}

func parseMethodComponentsResponses(method *protobuf.Method, cfg *settings.Settings) []*spec.Response {
	codes := lookup.LoadMethodResponseCodes(method)
	if len(codes) == 0 {
		return nil
	}

	var responses []*spec.Response
	for _, code := range codes {
		if lookup.IsSuccessResponseCode(code) {
			continue
		}

		errorName := cfg.Error.DefaultName
		responses = append(responses, &spec.Response{
			SchemaName:  errorName,
			Description: "The default error response.",
			Content: map[string]*spec.Media{
				"application/json": {
					Schema: &spec.Schema{
						Ref: refComponentsSchemas + errorName,
					},
				},
			},
		})
	}

	return responses
}
