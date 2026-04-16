resource "voyado_interaction_schema" "form_submission" {
  schema_id    = "form-submission"
  display_name = "Form submission"
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
