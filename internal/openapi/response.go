package openapi

import (
	"fmt"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	openapipb "github.com/mikros-dev/protoc-gen-openapi/openapi"
)

type Response struct {
	Description string            `yaml:"description,omitempty"`
	Content     map[string]*Media `yaml:"content"`
	schemaName  string
}

func parseOperationResponses(method *protobuf.Method) map[string]*Response {
	codes := getMethodResponseCodes(method)
	if len(codes) == 0 {
		return nil
	}

	responses := make(map[string]*Response)

	for _, code := range codes {
		refName := refComponentsSchemas + "DefaultError"
		if isSuccessCode(code) {
			refName = refComponentsSchemas + method.ResponseType.Name
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

func getMethodResponseCodes(method *protobuf.Method) []*openapipb.Response {
	var (
		codes []*openapipb.Response
	)

	if extensions := openapipb.LoadMethodExtensions(method.Proto); extensions != nil {
		for _, c := range extensions.GetResponse() {
			codes = append(codes, c)
		}
	}

	return codes
}
