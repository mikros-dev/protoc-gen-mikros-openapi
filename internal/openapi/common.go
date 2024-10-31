package openapi

import (
	"fmt"
	"slices"
	"strings"

	mextensionspb "github.com/mikros-dev/protoc-gen-mikros-extensions/mikros/extensions"
	"github.com/mikros-dev/protoc-gen-mikros-extensions/pkg/protobuf"
	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/mikros-dev/protoc-gen-openapi/internal/settings"
	openapipb "github.com/mikros-dev/protoc-gen-openapi/openapi"
)

type Media struct {
	Schema *Schema `json:"schema,omitempty"`
}

func isSuccessCode(code *openapipb.Response) bool {
	return code.GetCode() == openapipb.ResponseCode_RESPONSE_CODE_OK || code.GetCode() == openapipb.ResponseCode_RESPONSE_CODE_CREATED
}

type MessageParser struct {
	Package  *protobuf.Protobuf
	Settings *settings.Settings

	schemas map[string]bool
}

func (m *MessageParser) GetMessageSchemas(
	message *protobuf.Message,
	httpRule *annotations.HttpRule,
	methodExtensions *mextensionspb.MikrosMethodExtensions,
	pathParameters []string,
) (map[string]*Schema, error) {
	var (
		schemas            = make(map[string]*Schema)
		schemaProperties   = make(map[string]*Schema)
		requiredProperties []string
	)

	m.addParsedMessage(message.Name)

	for _, field := range message.Fields {
		if shouldHandleChildMessage(field) {
			if !m.isMessageAlreadyParsed(trimPackageName(field.TypeName)) {
				var (
					childMessage *protobuf.Message
				)

				if field.IsMessageFromPackage() {
					msg, err := findMessage(trimPackageName(field.TypeName), m.Package)
					if err != nil {
						return nil, err
					}
					childMessage = msg
				}
				if field.IsMessage() && childMessage == nil {
					msg, err := findForeignMessage(field.TypeName, m.Package)
					if err != nil {
						return nil, err
					}
					childMessage = msg
				}
				if childMessage == nil {
					continue
				}

				// Build the child message schema
				childSchemas, err := m.GetMessageSchemas(childMessage, httpRule, methodExtensions, pathParameters)
				if err != nil {
					return nil, err
				}

				for name, schema := range childSchemas {
					schemas[name] = schema
				}
			}

			// And adds as a property inside the main schema
			fieldSchema := newRefSchema(field, trimPackageName(field.TypeName), m.Package, m.Settings)
			schemaProperties[field.Name] = fieldSchema
			if fieldSchema.IsRequired() {
				requiredProperties = append(requiredProperties, field.Name)
			}

			continue
		}

		var (
			properties = openapipb.LoadFieldExtensions(field.Proto)
		)

		// Ignore fields that are not part of the body
		location := getFieldLocation(properties, httpRule, methodExtensions, field.Name, pathParameters)
		if location != "body" && m.Package.ModuleName == message.ModuleName {
			continue
		}

		fieldSchema := newSchemaFromProtobufField(field, m.Package, m.Settings)
		schemaProperties[field.Name] = fieldSchema
		if fieldSchema.IsRequired() {
			requiredProperties = append(requiredProperties, field.Name)
		}

		// Check if fieldSchema has an additionalProperty to be added as a schema.
		if fieldSchema.HasAdditionalProperties() {
			additionalSchemas, err := fieldSchema.GetAdditionalPropertySchemas(field, m)
			if err != nil {
				return nil, err
			}

			for name, scm := range additionalSchemas {
				schemas[name] = scm
			}
		}
	}

	schemas[message.Name] = &Schema{
		Type:       SchemaType_Object.String(),
		Properties: schemaProperties,
		Required:   requiredProperties,
		Message:    message,
	}

	return schemas, nil
}

func (m *MessageParser) addParsedMessage(name string) {
	if m.schemas == nil {
		m.schemas = make(map[string]bool)
	}

	m.schemas[name] = true
}

func (m *MessageParser) isMessageAlreadyParsed(name string) bool {
	_, ok := m.schemas[name]
	return ok
}

func shouldHandleChildMessage(field *protobuf.Field) bool {
	supportedFieldMessages := field.IsTimestamp() || field.IsProtoStruct() || field.IsProtoAny() || field.IsProtoValue()
	return field.IsMessageFromPackage() || field.IsMessage() && !supportedFieldMessages
}

func findForeignMessage(msgType string, pkg *protobuf.Protobuf) (*protobuf.Message, error) {
	var (
		foreignPackage = getPackageName(msgType)
		messages       []*protobuf.Message
	)

	// Load foreign messages
	for _, f := range pkg.Files {
		if f.Proto.GetPackage() == foreignPackage {
			messages = protobuf.ParseMessagesFromFile(f, f.Proto.GetPackage())
		}
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("could not load foreign messages")
	}

	// Search inside them
	msgIndex := slices.IndexFunc(messages, func(msg *protobuf.Message) bool {
		return msg.Name == trimPackageName(msgType)
	})
	if msgIndex == -1 {
		return nil, fmt.Errorf("could not find foreign message '%s'", msgType)
	}

	return messages[msgIndex], nil
}

func getPackageName(msgType string) string {
	parts := strings.Split(strings.TrimPrefix(msgType, "."), ".")
	return strings.Join(parts[:len(parts)-1], ".")
}

func getFieldLocation(
	properties *openapipb.Property,
	httpRule *annotations.HttpRule,
	methodExtensions *mextensionspb.MikrosMethodExtensions,
	fieldName string,
	pathParameters []string,
) string {
	// Get the location from our own proto annotation.
	if properties != nil && properties.GetLocation() != openapipb.PropertyLocation_PROPERTY_LOCATION_UNSPECIFIED {
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

	return "query"
}

func trimPackageName(name string) string {
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}
