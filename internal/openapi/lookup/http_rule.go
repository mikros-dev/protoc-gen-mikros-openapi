package lookup

import (
	mikros_extensions "github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf/extensions"
	"google.golang.org/genproto/googleapis/api/annotations"
)

func HTTPEndpoint(httpRule *annotations.HttpRule) (string, string) {
	return mikros_extensions.GetHTTPEndpoint(httpRule)
}

func EndpointInformation(httpRule *annotations.HttpRule) ([]string, string) {
	endpoint, method := HTTPEndpoint(httpRule)
	return mikros_extensions.RetrieveParameters(endpoint), method
}
