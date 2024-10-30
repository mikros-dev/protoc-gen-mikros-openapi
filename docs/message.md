# Message options

A message has the following options available:

| Name                | Modifier | Description                |
|---------------------|----------|----------------------------|
| [message](#message) | optional | Available message options. |

## message

| Name                    | Type   | Modifier | Description                                  |
|-------------------------|--------|----------|----------------------------------------------|
| [operation](#operation) | object | required | Sets the message operation required options. |

### operation

| Name                          | Type   | Modifier | Description                      |
|-------------------------------|--------|----------|----------------------------------|
| [request_body](#request_body) | object | required | Describes a single request body. |

#### request_body

| Name        | Type   | Modifier | Description                              |
|-------------|--------|----------|------------------------------------------|
| description | string | required | A brief description of the request body. |
