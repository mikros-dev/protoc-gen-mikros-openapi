package lookup

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

func LoadMethodResponseCodes(method *protobuf.Method) []*mikros_openapi.Response {
	if method == nil {
		return nil
	}

	extensions := mikros_openapi.LoadMethodExtensions(method.Proto)
	if extensions == nil {
		return nil
	}

	return extensions.GetResponse()
}

func LoadServiceSecurityExtensions(pkg *protobuf.Protobuf) []*mikros_openapi.OpenapiServiceSecurity {
	if pkg == nil {
		return nil
	}
	if pkg.Service == nil {
		return nil
	}

	return mikros_openapi.LoadServiceExtensions(pkg.Service.Proto)
}

func LoadMessageExtensionsByName(pkg *protobuf.Protobuf, name string) *mikros_openapi.OpenapiMessage {
	if pkg == nil {
		return nil
	}

	message, err := FindMessageByName(name, pkg)
	if err != nil {
		return nil
	}

	return mikros_openapi.LoadMessageExtensions(message.Proto)
}
