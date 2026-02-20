package lookup

import (
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

// LoadMethodResponseCodes returns the list of response codes defined for the
// given method.
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

// LoadServiceSecurityExtensions returns the list of security extensions defined for the
// given service.
func LoadServiceSecurityExtensions(pkg *protobuf.Protobuf) []*mikros_openapi.OpenapiServiceSecurity {
	if pkg == nil {
		return nil
	}
	if pkg.Service == nil {
		return nil
	}

	return mikros_openapi.LoadServiceExtensions(pkg.Service.Proto)
}

// LoadMessageExtensionsByName finds a message by its name and returns tits
// protobuf extensions.
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

// IsSuccessResponseCode returns true if the given response code is a success
// code.
func IsSuccessResponseCode(code *mikros_openapi.Response) bool {
	c := int(code.GetCode())
	return c >= 200 && c < 300
}
