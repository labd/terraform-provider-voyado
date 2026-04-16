# terraform-provider-voyado

Terraform provider for [Voyado Engage](https://developer.voyado.com/docs/api/the-engage-api) — currently the [`voyado_interaction_schema`](https://developer.voyado.com/docs/loyalty/interactions#the-interactionschemas-endpoint) resource, which maps to Engage API v3:

- `POST /api/v3/interactionschemas`
- `GET /api/v3/interactionschemas/{interactionSchemaId}`
- `DELETE /api/v3/interactionschemas/{interactionSchemaId}`

## Requirements

- [Go](https://go.dev/dl/) 1.26 or newer (see `go.mod`)

## Provider configuration

| Argument     | Description |
|-------------|-------------|
| `api_url`   | Base URL of the Engage API (for example `https://mytenant.voyado.com` or `https://mytenant.staging.voyado.com`). |
| `api_key`   | Engage API key (sent as the `apikey` header). |
| `user_agent`| Optional. Override the default `User-Agent` (Voyado recommends a descriptive value). |

## Resource: `voyado_interaction_schema`

| Attribute       | Description |
|----------------|-------------|
| `schema_id`    | Unique schema id (`id` in the API). |
| `display_name` | `displayName` in the API. |
| `json_schema`  | JSON object as a string, sent as `jsonSchema` (see Voyado’s interaction schema rules). |
| `id`           | Computed; same as `schema_id`. |

Engage does not support updating a schema in place. Changing `schema_id`, `display_name`, or `json_schema` forces replacement (delete then create). Deleting a schema removes all interactions for that schema.

### Example

```hcl
terraform {
  required_providers {
    voyado = {
      source = "labd/voyado"
    }
  }
}

provider "voyado" {
  api_url = "https://mytenant.staging.voyado.com"
  api_key = var.engage_api_key
}

resource "voyado_interaction_schema" "reuse" {
  schema_id    = "Reuse-Spring-2023"
  display_name = "Reuse Spring Sale"
  json_schema = jsonencode({
    "$schema" = "https://json-schema.org/draft/2020-12/schema"
    type      = "object"
    properties = {
      name = {
        type              = "string"
        displayName       = "Name"
        showInContactCard = "true"
        sortOrder         = "0"
      }
    }
    required = ["name"]
  })
}
```

Import an existing schema by id:

```bash
terraform import voyado_interaction_schema.reuse Reuse-Spring-2023
```