package lookup

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"
)

func LoadHTTPRule(method *protobuf.Method) *annotations.HttpRule {
	if method == nil {
		return nil
	}

	return mikros_extensions.LoadGoogleAnnotations(method.Proto)
}

func HTTPEndpoint(httpRule *annotations.HttpRule) (string, string) {
	if httpRule == nil {
		return "", ""
	}

	return mikros_extensions.GetHTTPEndpoint(httpRule)
}

func EndpointInformation(httpRule *annotations.HttpRule) ([]string, string) {
	if httpRule == nil {
		return nil, ""
	}

	endpoint, method := HTTPEndpoint(httpRule)
	return mikros_extensions.RetrieveParameters(endpoint), method
}
