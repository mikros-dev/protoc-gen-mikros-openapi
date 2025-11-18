# Field options

A field has the following options available:

| Name                  | Modifier | Description                             |
|-----------------------|----------|-----------------------------------------|
| [property](#property) | optional | Describes a single operation parameter. |

## property

| Name                  | Type   | Modifier | Description                                                                |
|-----------------------|--------|----------|----------------------------------------------------------------------------|
| description           | string | optional | A brief description of the parameter.                                      |
| example               | string | optional | A free-form property to include an example of an instance for this schema. |
| [format](#format)     | enum   | optional | The field type.                                                            |
| required              | bool   | optional | Sets if the field is required in the message or not.                       |
| [location](#location) | enum   | optional | The field location in the request.                                         |
| hide_from_schema      | bool   | optional | Hides the field from the generated schema.                                 |

### format

| Name                      |
|---------------------------|
| PROPERTY_FORMAT_INT32     |
| PROPERTY_FORMAT_INT64     |
| PROPERTY_FORMAT_FLOAT     |
| PROPERTY_FORMAT_DOUBLE    |
| PROPERTY_FORMAT_BYTE      |
| PROPERTY_FORMAT_BINARY    |
| PROPERTY_FORMAT_DATE      |
| PROPERTY_FORMAT_DATE_TIME |
| PROPERTY_FORMAT_PASSWORD  |
| PROPERTY_FORMAT_STRING    |

### location

| Name                     |
|--------------------------|
| PROPERTY_LOCATION_BODY   |
| PROPERTY_LOCATION_QUERY  |
| PROPERTY_LOCATION_PATH   |
| PROPERTY_LOCATION_HEADER |
