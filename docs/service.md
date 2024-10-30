# Service options

A service has the following options available:

| Name                  | Modifier | Description                |
|-----------------------|----------|----------------------------|
| [security](#security) | array    | Sets API security options. |

## security

| Name                | Type   | Modifier | Description                                                                  |
|---------------------|--------|----------|------------------------------------------------------------------------------|
| [type](#type)       | enum   | required | Sets the security type.                                                      |
| description         | string | optional | Sets a security description.                                                 |
| name                | string | required | The security schema name.                                                    |
| [in](#in)           | enum   | optional | The location of the API key.                                                 |
| [scheme](#scheme)   | enum   | required | The HTTP Authorization scheme to be used.                                    |
| bearer_format       | string | optional | A hint to the client to identify how the bearer token is formatted.          |
| [flows](#flows)     | object | optional | An object containing configuration information for the flow types supported. |
| open_id_connect_url | string | optional | OpenId Connect URL to discover OAuth2 configuration values.                  |

### type

| Name                                  |
|---------------------------------------|
| OPENAPI_SECURITY_TYPE_UNSPECIFIED     |
| OPENAPI_SECURITY_TYPE_API_KEY         |
| OPENAPI_SECURITY_TYPE_HTTP            |
| OPENAPI_SECURITY_TYPE_OAUTH2          |
| OPENAPI_SECURITY_TYPE_OPEN_ID_CONNECT |

### in

| Name                                     | Description                                    |
|------------------------------------------|------------------------------------------------|
| OPENAPI_SECURITY_API_KEY_LOCATION_QUERY  | Api Key should be passed as a query parameter. |
| OPENAPI_SECURITY_API_KEY_LOCATION_HEADER | Api Key should be passed as header parameter.  |
| OPENAPI_SECURITY_API_KEY_LOCATION_COOKIE | Api Key should be passed as cookie parameter.  |

### scheme

| Name                           |
|--------------------------------|
| OPENAPI_SECURITY_SCHEME_BASIC  |
| OPENAPI_SECURITY_SCHEME_BEARER |
| OPENAPI_SECURITY_SCHEME_OAUTH  |
| OPENAPI_SECURITY_SCHEME_DIGEST |

### flows

| Name              | Type                | Modifier | Description                                          |
|-------------------|---------------------|----------|------------------------------------------------------|
| authorization_url | string              | required | The authorization URL to be used for this flow.      |
| token_url         | string              | required | The token URL to be used for this flow.              |
| refresh_url       | string              | optional | The URL to be used for obtaining refresh tokens.     |
| scopes            | map<string, string> | optional | The available scopes for the OAuth2 security scheme. |
