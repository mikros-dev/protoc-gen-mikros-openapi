package extract

import (
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"
	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/internal/openapi/lookup"
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

type schemaScope string

const (
	schemaScopeRequest  schemaScope = "request"
	schemaScopeResponse schemaScope = "response"
)

// methodContext is a helper structure to hold method-specific context.
type methodContext struct {
	method           *protobuf.Method
	httpRule         *annotations.HttpRule
	httpMethod       string
	endpoint         string
	pathParameters   []string
	responseCodes    []*mikros_openapi.Response
	methodExtensions *mikros_extensions.MikrosMethodExtensions
	extensions       *mikros_openapi.OpenapiMethod
	requestMessage   *protobuf.Message
	responseMessage  *protobuf.Message
	schemaScope      schemaScope
}

// buildMethodContext centralizes extraction of annotations and path params for
// a method.
func (p *Parser) buildMethodContext(method *protobuf.Method) *methodContext {
	httpRule := lookup.LoadHTTPRule(method)
	pathParameters, _ := lookup.EndpointInformation(httpRule)

	ctx := &methodContext{
		method:           method,
		httpRule:         httpRule,
		pathParameters:   pathParameters,
		responseCodes:    lookup.LoadMethodResponseCodes(method),
		methodExtensions: mikros_extensions.LoadMethodExtensions(method.Proto),
		extensions:       mikros_openapi.LoadMethodExtensions(method.Proto),
	}

	if httpRule == nil {
		return ctx
	}

	endpoint, httpMethod := lookup.HTTPEndpoint(httpRule)
	if p.cfg.AddServiceNameInEndpoints {
		endpoint = fmt.Sprintf("/%v%v", strcase.ToKebab(p.pkg.ModuleName), endpoint)
	}

	ctx.endpoint = endpoint
	ctx.httpMethod = httpMethod

	return ctx
}

func (p *Parser) loadMethodMessages(methodCtx *methodContext) error {
	req, err := lookup.FindMessageByName(methodCtx.method.RequestType.Name, p.pkg)
	if err != nil {
		return err
	}

	resp, err := lookup.FindMessageByName(methodCtx.method.ResponseType.Name, p.pkg)
	if err != nil {
		return err
	}

	methodCtx.requestMessage = req
	methodCtx.responseMessage = resp

	return nil
}
