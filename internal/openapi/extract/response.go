package extract

import (
	"fmt"
	"sort"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/mapping"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/settings"
)

func (p *Parser) buildOperationResponses(
	methodCtx *methodContext,
	converter *mapping.Message,
) map[string]*spec.Response {
	var (
		responses         = make(map[string]*spec.Response)
		successSchemaName = methodCtx.method.ResponseType.Name
		errorName         = p.cfg.Error.DefaultName
	)

	if p.cfg.Mikros.UseOutboundMessages {
		successSchemaName = converter.WireOutputToOutbound(successSchemaName)
	}

	for _, code := range mergedMethodResponses(methodCtx, p.cfg) {
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

	if len(responses) == 0 {
		return nil
	}

	return responses
}

func responseDescriptionOrDefault(code *mikros_openapi.Response) string {
	if code.GetDescription() != "" {
		return code.GetDescription()
	}

	// Response code with no description will have a default message.
	return fmt.Sprintf("HTTP %d response", code.GetCode())
}

func (p *Parser) buildComponentResponses() map[string]*spec.Response {
	if !p.shouldBuildDefaultErrorComponentResponse() {
		return nil
	}

	errorName := p.cfg.Error.DefaultName
	return map[string]*spec.Response{
		errorName: {
			Description: p.cfg.Error.DefaultDescription,
			Content: map[string]*spec.Media{
				"application/json": {
					Schema: &spec.Schema{
						Ref: refComponentsSchemas + errorName,
					},
				},
			},
		},
	}
}

func (p *Parser) shouldBuildDefaultErrorComponentResponse() bool {
	if len(p.cfg.Error.Responses) > 0 {
		return true
	}

	for _, method := range p.pkg.Service.Methods {
		for _, code := range lookup.LoadMethodResponseCodes(method) {
			if lookup.IsSuccessResponseCode(code) {
				continue
			}

			return true
		}
	}

	return false
}

func mergedMethodResponses(methodCtx *methodContext, cfg *settings.Settings) []*mikros_openapi.Response {
	merged := make(map[mikros_openapi.ResponseCode]*mikros_openapi.Response)

	// Add the default success response
	successCode := mikros_openapi.ResponseCode(cfg.Operation.DefaultSuccessCode)
	merged[successCode] = &mikros_openapi.Response{
		Code:        &successCode,
		Description: &cfg.Operation.DefaultSuccessDescription,
	}

	// cfg defined default error codes
	for _, r := range cfg.Error.Responses {
		code := mikros_openapi.ResponseCode(r.Code)
		desc := r.Description
		merged[code] = &mikros_openapi.Response{
			Code:        &code,
			Description: &desc,
		}
	}

	responses := methodCtx.responseCodes
	if hasAnySuccessResponse(responses) {
		for code := range merged {
			if lookup.IsSuccessResponseCode(&mikros_openapi.Response{Code: &code}) {
				delete(merged, code)
			}
		}
	}

	// Response codes defined in the proto file. Here we'll override the default
	// codes (success and errors) if they are defined.
	for _, r := range responses {
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

func hasAnySuccessResponse(response []*mikros_openapi.Response) bool {
	for _, r := range response {
		if lookup.IsSuccessResponseCode(r) {
			return true
		}
	}

	return false
}
