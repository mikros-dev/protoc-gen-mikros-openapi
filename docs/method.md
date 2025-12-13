# Method options

A method has the following options available:

| Name                    | Modifier | Description                                 |
|-------------------------|----------|---------------------------------------------|
| [operation](#operation) | optional | Describes a single API operation on a path. |

## operation

| Name                       | Type   | Modifier | Description                                                                        |
|----------------------------|--------|----------|------------------------------------------------------------------------------------|
| summary                    | string | required | A short summary of what the operation does.                                        |
| description                | string | required | A verbose explanation of the operation behavior.                                   |
| tags                       | string | array    | A list of tags for API documentation control.                                      |
| [response](#response)      | object | array    | The list of possible responses as they are returned from executing this operation. |
| disable_inbound_processing | bool   | optional | Disables post-processing names using mikros inbound settings for the messages.     |

### response

| Name          | Type   | Modifier | Description                          |
|---------------|--------|----------|--------------------------------------|
| [code](#code) | enum   | required | The HTTP response code.              |
| description   | string | required | A short description of the response. |

#### code

| Name                                          | HTTP code |
|-----------------------------------------------|-----------|
| RESPONSE_CODE_CONTINUE                        | 100       |
| RESPONSE_CODE_SWITCHING_PROTOCOLS             | 101       |
| RESPONSE_CODE_PROCESSING                      | 102       |
| RESPONSE_CODE_EARLY_HINTS                     | 103       |
| RESPONSE_CODE_OK                              | 200       |
| RESPONSE_CODE_CREATED                         | 201       |
| RESPONSE_CODE_ACCEPTED                        | 202       |
| RESPONSE_CODE_NON_AUTHORITATIVE_INFO          | 203       |
| RESPONSE_CODE_NO_CONTENT                      | 204       |
| RESPONSE_CODE_RESET_CONTENT                   | 205       |
| RESPONSE_CODE_PARTIAL_CONTENT                 | 206       |
| RESPONSE_CODE_MULTI_STATUS                    | 207       |
| RESPONSE_CODE_ALREADY_REPORTED                | 208       |
| RESPONSE_CODE_IM_USED                         | 226       |
| RESPONSE_CODE_MULTIPLE_CHOICES                | 300       |
| RESPONSE_CODE_MOVED_PERMANENTLY               | 301       |
| RESPONSE_CODE_FOUND                           | 302       |
| RESPONSE_CODE_SEE_OTHER                       | 303       |
| RESPONSE_CODE_NOT_MODIFIED                    | 304       |
| RESPONSE_CODE_USE_PROXY                       | 305       |
| RESPONSE_CODE_TEMPORARY_REDIRECT              | 307       |
| RESPONSE_CODE_PERMANENT_REDIRECT              | 308       |
| RESPONSE_CODE_BAD_REQUEST                     | 400       |
| RESPONSE_CODE_UNAUTHORIZED                    | 401       |
| RESPONSE_CODE_PAYMENT_REQUIRED                | 402       |
| RESPONSE_CODE_FORBIDDEN                       | 403       |
| RESPONSE_CODE_NOT_FOUND                       | 404       |
| RESPONSE_CODE_METHOD_NOT_ALLOWED              | 405       |
| RESPONSE_CODE_NOT_ACCEPTABLE                  | 406       |
| RESPONSE_CODE_PROXY_AUTH_REQUIRED             | 407       |
| RESPONSE_CODE_REQUEST_TIMEOUT                 | 408       |
| RESPONSE_CODE_CONFLICT                        | 409       |
| RESPONSE_CODE_GONE                            | 410       |
| RESPONSE_CODE_LENGTH_REQUIRED                 | 411       |
| RESPONSE_CODE_PRECONDITION_FAILED             | 412       |
| RESPONSE_CODE_REQUEST_ENTITY_TOO_LARGE        | 413       |
| RESPONSE_CODE_REQUEST_URI_TOO_LONG            | 414       |
| RESPONSE_CODE_UNSUPPORTED_MEDIA_TYPE          | 415       |
| RESPONSE_CODE_REQUESTED_RANGE_NOT_SATISFIABLE | 416       |
| RESPONSE_CODE_EXPECTATION_FAILED              | 417       |
| RESPONSE_CODE_TEAPOT                          | 418       |
| RESPONSE_CODE_MISDIRECTED_REQUEST             | 421       |
| RESPONSE_CODE_UNPROCESSABLE_ENTITY            | 422       |
| RESPONSE_CODE_LOCKED                          | 423       |
| RESPONSE_CODE_FAILED_DEPENDENCY               | 424       |
| RESPONSE_CODE_TOO_EARLY                       | 425       |
| RESPONSE_CODE_UPGRADE_REQUIRED                | 426       |
| RESPONSE_CODE_PRECONDITION_REQUIRED           | 428       |
| RESPONSE_CODE_TOO_MANY_REQUESTS               | 429       |
| RESPONSE_CODE_REQUEST_HEADER_FIELDS_TOO_LARGE | 431       |
| RESPONSE_CODE_UNAVAILABLE_FOR_LEGAL_REASONS   | 451       |
| RESPONSE_CODE_INTERNAL_SERVER_ERROR           | 500       |
| RESPONSE_CODE_NOT_IMPLEMENTED                 | 501       |
| RESPONSE_CODE_BAD_GATEWAY                     | 502       |
| RESPONSE_CODE_SERVICE_UNAVAILABLE             | 503       |
| RESPONSE_CODE_GATEWAY_TIMEOUT                 | 504       |
| RESPONSE_CODE_HTTP_VERSION_NOT_SUPPORTED      | 505       |
| RESPONSE_CODE_VARIANT_ALSO_NEGOTIATES         | 506       |
| RESPONSE_CODE_INSUFFICIENT_STORAGE            | 507       |
| RESPONSE_CODE_LOOP_DETECTED                   | 508       |
| RESPONSE_CODE_NOT_EXTENDED                    | 510       |
| RESPONSE_CODE_NETWORK_AUTHENTICATION_REQUIRED | 511       |
