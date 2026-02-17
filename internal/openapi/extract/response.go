package extract

import (
	"fmt"
	"sort"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
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
		responses         = make(map[string]*spec.Response)
		successSchemaName = method.ResponseType.Name
		errorName         = cfg.Error.DefaultName
	)

	if cfg.Mikros.UseOutboundMessages {
		successSchemaName = converter.WireOutputToOutbound(successSchemaName)
	}

	for _, code := range mergedMethodResponses(method, cfg) {
		refName := refComponentsSchemas + errorName
		if lookup.IsSuccessResponseCode(code) {
			refName = refComponentsSchemas + successSchemaName
		}

		responses[fmt.Sprintf("%d", code.GetCode())] = &spec.Response{
			Description: responseDescriptionOrDefault(code),
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
	var responses []*spec.Response
	for _, code := range mergedMethodResponses(method, cfg) {
		if lookup.IsSuccessResponseCode(code) {
			continue
		}

		errorName := cfg.Error.DefaultName
		responses = append(responses, &spec.Response{
			SchemaName:  errorName,
			Description: cfg.Error.DefaultDescription,
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

func mergedMethodResponses(method *protobuf.Method, cfg *settings.Settings) []*mikros_openapi.Response {
	merged := make(map[mikros_openapi.ResponseCode]*mikros_openapi.Response)

	for _, r := range cfg.Error.Responses {
		code := mikros_openapi.ResponseCode(r.Code)
		desc := r.Description
		merged[code] = &mikros_openapi.Response{
			Code:        &code,
			Description: &desc,
		}
	}

	for _, r := range lookup.LoadMethodResponseCodes(method) {
		code := r.GetCode()
		if code == mikros_openapi.ResponseCode_RESPONSE_CODE_UNSPECIFIED {
			continue
		}

		merged[code] = r
	}

	codes := make([]int, 0, len(merged))
	for code := range merged {
		codes = append(codes, int(code))
	}
	sort.Ints(codes)

	out := make([]*mikros_openapi.Response, 0, len(codes))
	for _, code := range codes {
		out = append(out, merged[mikros_openapi.ResponseCode(code)])
	}

	return out
}

func responseDescriptionOrDefault(code *mikros_openapi.Response) string {
	if code.GetDescription() != "" {
		return code.GetDescription()
	}

	// Response code with no description will have a default message.
	return fmt.Sprintf("HTTP %d response", code.GetCode())
}
