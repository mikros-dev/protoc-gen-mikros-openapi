package extract

import (
	"slices"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"

	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"

	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

func isSuccessCode(code *mikros_openapi.Response) bool {
	return code.GetCode() == mikros_openapi.ResponseCode_RESPONSE_CODE_OK ||
		code.GetCode() == mikros_openapi.ResponseCode_RESPONSE_CODE_CREATED
}

func getFieldLocation(
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

	if httpRule.GetBody() == "*" {
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

