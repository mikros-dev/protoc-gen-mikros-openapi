# protoc-gen-openapi

A protoc/buf plugin to generate compatible [OpenAPI version 3.0.0](https://swagger.io/specification/v3/)
YAML files from protobuf HTTP API declarations.

## Features

This plugin provides an easy way of generating OpeAPI specification of an HTTP
service API directly from its protobuf file. It uses protobuf annotation options
to allow the user define details about the service and its endpoints.

It can be used alone or together with the [protoc-gen-mikros-extensions](https://github.com/mikros-dev/protoc-gen-mikros-extension)
plugin for messages and field names.

## Building and installing

In order to compile and install the plugin locally you'll need to follow the steps:

* Install the go compiler;
* Execute the commands:
    * `go generate`
    * `go build && go install`

## Protobuf extensions available

The following links present details about available options to be used from a
protobuf file.

* [File](docs/file.md)
* [Service](docs/service.md)
* [Method](docs/method.md)
* [Message](docs/message.md)
* [Field](docs/field.md)

For more details or a complete example, use the [examples](examples) directory.

## License

[Apache License 2.0](LICENSE)
