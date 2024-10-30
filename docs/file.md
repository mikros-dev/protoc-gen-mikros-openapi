# File options

The file has the following options available:

| Name                  | Modifier | Description                                        |
|-----------------------|----------|----------------------------------------------------|
| [metadata](#metadata) | optional | Sets OpenAPI metadata information for the service. |

## metadata

| Name              | Type   | Modifier | Description                              |
|-------------------|--------|----------|------------------------------------------|
| [info](#info)     | object | optional | Sets main information about the service. |
| [server](#server) | object | array    | Sets servers to be used with the API.    |

### info

| Name        | Type   | Modifier | Description                                              |
|-------------|--------|----------|----------------------------------------------------------|
| title       | string | required | Sets the documentation title (usually the service name). |
| description | string | optional | An optional description of the API.                      |
| version     | string | required | The API version.                                         |

### server

| Name        | Type   | Modifier | Description                            |
|-------------|--------|----------|----------------------------------------|
| url         | string | required | The server URL.                        |
| description | string | optional | An optional description of the server. |
