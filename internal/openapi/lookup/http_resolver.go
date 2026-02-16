package lookup

import (
	"slices"
	"strings"

	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
	"google.golang.org/genproto/googleapis/api/annotations"
)

// LoadHTTPRule returns the HTTP rule for the given method.
func LoadHTTPRule(method *protobuf.Method) *annotations.HttpRule {
	if method == nil {
		return nil
	}

	return mikros_extensions.LoadGoogleAnnotations(method.Proto)
}

// HTTPEndpoint returns the endpoint and method for the given HTTP rule.
func HTTPEndpoint(httpRule *annotations.HttpRule) (string, string) {
	if httpRule == nil {
		return "", ""
	}

	return mikros_extensions.GetHTTPEndpoint(httpRule)
}

// EndpointInformation returns the path parameters and method for the given HTTP rule.
func EndpointInformation(httpRule *annotations.HttpRule) ([]string, string) {
	if httpRule == nil {
		return nil, ""
	}

	endpoint, method := HTTPEndpoint(httpRule)
	return mikros_extensions.RetrieveParameters(endpoint), method
}

// FieldLocation returns the location of the given field in a request.
func FieldLocation(
	properties *mikros_openapi.Property,
	httpRule *annotations.HttpRule,
	methodExtensions *mikros_extensions.MikrosMethodExtensions,
	fieldName string,
	pathParameters []string,
) string {
	// Get the location from our own proto annotation.
	if properties != nil && properties.GetLocation() != mikros_openapi.PropertyLocation_PROPERTY_LOCATION_UNSPECIFIED {
		return strings.ToLower(strings.TrimPrefix(properties.GetLocation().String(), "PROPERTY_LOCATION_"))
	}

	// Try to guess the location from field parameters.
	if slices.Contains(pathParameters, fieldName) {
		return "path"
	}

	if httpRule != nil && httpRule.GetBody() == "*" {
		return "body"
	}

	if methodExtensions != nil && methodExtensions.GetHttp() != nil {
		if slices.Contains(methodExtensions.GetHttp().GetHeader(), fieldName) {
			return "header"
		}
	}

	// Field has no annotation
	return "query"
}
