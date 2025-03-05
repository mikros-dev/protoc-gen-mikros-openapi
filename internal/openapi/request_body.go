package openapi

import (
	"net/http"
	"slices"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	openapipb "github.com/mikros-dev/protoc-gen-mikros-openapi/openapi"
)

type RequestBody struct {
	Required    bool              `yaml:"required"`
	Description string            `yaml:"description,omitempty"`
	Content     map[string]*Media `yaml:"content"`
}

func parseRequestBody(method *protobuf.Method, httpMethod string, pkg *protobuf.Protobuf) *RequestBody {
	if httpMethod != http.MethodPost && httpMethod != http.MethodPut {
		return nil
	}

	var (
		required    bool
		description string
	)

	if httpMethod == http.MethodPost {
		required = true
	}
	if extensions := findRequestBodyMessageExtensions(pkg, method.RequestType.Name); extensions != nil {
		description = extensions.GetOperation().GetRequestBody().GetDescription()
	}

	return &RequestBody{
		Required:    required,
		Description: description,
		Content: map[string]*Media{
			"application/json": {
				Schema: &Schema{
					Ref: refComponentsSchemas + method.RequestType.Name,
				},
			},
		},
	}
}

func findRequestBodyMessageExtensions(pkg *protobuf.Protobuf, name string) *openapipb.OpenapiMessage {
	index := slices.IndexFunc(pkg.Messages, func(msg *protobuf.Message) bool {
		return msg.Name == name
	})
	if index == -1 {
		return nil
	}

	return openapipb.LoadMessageExtensions(pkg.Messages[index].Proto)
}
