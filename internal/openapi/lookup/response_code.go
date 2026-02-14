package lookup

import (
	"github.com/mikros-dev/protoc-gen-mikros-openapi/pkg/mikros_openapi"
)

func IsSuccessCode(code *mikros_openapi.Response) bool {
	return code.GetCode() == mikros_openapi.ResponseCode_RESPONSE_CODE_OK ||
		code.GetCode() == mikros_openapi.ResponseCode_RESPONSE_CODE_CREATED
}
