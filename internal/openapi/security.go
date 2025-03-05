package openapi

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	openapipb "github.com/mikros-dev/protoc-gen-mikros-openapi/openapi"
)

type Security struct {
	Type         string `yaml:"type"`
	Scheme       string `yaml:"scheme"`
	BearerFormat string `yaml:"bearerFormat,omitempty"`
}

func parseOperationSecurity(pkg *protobuf.Protobuf) []map[string][]string {
	if extensions := openapipb.LoadServiceExtensions(pkg.Service.Proto); extensions != nil {
		security := make([]map[string][]string, len(extensions))
		for i, extension := range extensions {
			security[i] = map[string][]string{
				extension.GetName(): {},
			}
		}

		return security
	}

	return nil
}

func parseComponentsSecurity(pkg *protobuf.Protobuf) map[string]*Security {
	if extensions := openapipb.LoadServiceExtensions(pkg.Service.Proto); extensions != nil {
		security := make(map[string]*Security)
		for _, extension := range extensions {
			security[extension.GetName()] = &Security{
				Type:         securityTypeToString(extension.GetType()),
				Scheme:       securitySchemeToString(extension.GetScheme()),
				BearerFormat: extension.GetBearerFormat(),
			}
		}

		return security
	}

	return nil
}

func securityTypeToString(securityType openapipb.OpenapiSecurityType) string {
	switch securityType {
	case openapipb.OpenapiSecurityType_OPENAPI_SECURITY_TYPE_API_KEY:
		return "apiKey"
	case openapipb.OpenapiSecurityType_OPENAPI_SECURITY_TYPE_HTTP:
		return "http"
	case openapipb.OpenapiSecurityType_OPENAPI_SECURITY_TYPE_OAUTH2:
		return "oauth2"
	case openapipb.OpenapiSecurityType_OPENAPI_SECURITY_TYPE_OPEN_ID_CONNECT:
		return "openIdConnect"
	}

	return ""
}

func securitySchemeToString(securityScheme openapipb.OpenapiSecurityScheme) string {
	switch securityScheme {
	case openapipb.OpenapiSecurityScheme_OPENAPI_SECURITY_SCHEME_BASIC:
		return "basic"
	case openapipb.OpenapiSecurityScheme_OPENAPI_SECURITY_SCHEME_BEARER:
		return "bearer"
	case openapipb.OpenapiSecurityScheme_OPENAPI_SECURITY_SCHEME_DIGEST:
		return "digest"
	case openapipb.OpenapiSecurityScheme_OPENAPI_SECURITY_SCHEME_OAUTH:
		return "oauth"
	}

	return ""
}
