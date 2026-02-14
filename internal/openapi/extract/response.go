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
		if isSuccessCode(code) {
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

