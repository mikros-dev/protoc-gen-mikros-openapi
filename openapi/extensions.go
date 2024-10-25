package openapi

import (
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

func LoadMetadata(file *descriptor.FileDescriptorProto) *OpenapiMetadata {
	if file.Options != nil {
		v := proto.GetExtension(file.Options, E_Metadata)
		if val, ok := v.(*OpenapiMetadata); ok {
			return val
		}
	}

	return nil
}

func LoadMethodExtensions(method *descriptor.MethodDescriptorProto) *OpenapiMethod {
	if method.Options != nil {
		v := proto.GetExtension(method.Options, E_Operation)
		if val, ok := v.(*OpenapiMethod); ok {
			return val
		}
	}

	return nil
}

func LoadMessageExtensions(msg *descriptor.DescriptorProto) *OpenapiMessage {
	if msg.Options != nil {
		v := proto.GetExtension(msg.Options, E_Message)
		if val, ok := v.(*OpenapiMessage); ok {
			return val
		}
	}

	return nil
}

func LoadServiceExtensions(service *descriptor.ServiceDescriptorProto) []*OpenapiServiceSecurity {
	if service.Options != nil {
		v := proto.GetExtension(service.Options, E_Security)
		if val, ok := v.([]*OpenapiServiceSecurity); ok {
			return val
		}
	}

	return nil
}

func LoadFieldExtensions(field *descriptor.FieldDescriptorProto) *Property {
	if field.Options != nil {
		v := proto.GetExtension(field.Options, E_Property)
		if val, ok := v.(*Property); ok {
			return val
		}
	}

	return nil
}
