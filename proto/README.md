# protoc-gen-mikros-openapi

A protoc/buf plugin to generate compatible [OpenAPI version 3.0.0](https://swagger.io/specification/v3/)
YAML files from protobuf HTTP API declarations.

## Usage

In order to use this schema, the following excerpt must be added into your
**buf.yaml** file:

```yaml
deps:
  - buf.build/mikros-dev/protoc-gen-mikros-openapi
```

> Note: Assuming buf version 2 is being used.

For more details on how to use the plugin and its features, check its own
[repository](https://github.com/mikros-dev/protoc-gen-mikros-openapi).
