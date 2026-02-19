package extract

import (
	"net/http"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/openapi/spec"
)

func (p *Parser) buildRequestBody(methodCtx *methodContext) *spec.RequestBody {
	httpMethod := methodCtx.httpMethod
	if httpMethod != http.MethodPost && httpMethod != http.MethodPut && httpMethod != http.MethodPatch {
		return nil
	}

	var (
		required    bool
		description string
		contentType = "application/json"
	)

	if httpMethod == http.MethodPost {
		required = true
	}

	if extensions := lookup.LoadMessageExtensionsByName(p.pkg, methodCtx.method.RequestType.Name); extensions != nil {
		description = extensions.GetOperation().GetRequestBody().GetDescription()

		switch extensions.GetOperation().GetRequestBody().GetType() {
		case mikros_openapi.RequestBodyType_REQUEST_BODY_TYPE_MULTIPART_FORM_DATA:
			contentType = "multipart/form-data"
		default:
			contentType = "application/json"
		}
	}

	return &spec.RequestBody{
		Required:    required,
		Description: description,
		Content: map[string]*spec.Media{
			contentType: {
				Schema: &spec.Schema{
					Ref: refComponentsSchemas + methodCtx.method.RequestType.Name,
				},
			},
		},
	}
}
