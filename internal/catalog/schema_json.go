package catalog

const patternSchemaV1 = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["id", "name", "category", "description", "sources", "detection", "rule_template"],
  "properties": {
    "id": { "type": "string", "minLength": 1 },
    "name": { "type": "string", "minLength": 1 },
    "category": { "type": "string", "minLength": 1 },
    "description": { "type": "string", "minLength": 1 },
    "sources": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["title", "url", "license"],
        "properties": {
          "title": { "type": "string", "minLength": 1 },
          "url": { "type": "string", "minLength": 1 },
          "license": { "type": "string", "minLength": 1 }
        }
      }
    },
    "detection": {
      "type": "object",
      "additionalProperties": false,
      "required": ["required_facts", "heuristic"],
      "properties": {
        "required_facts": {
          "type": "array",
          "minItems": 1,
          "items": {
            "type": "string",
            "enum": ["imports", "ast"]
          }
        },
        "heuristic": {
          "oneOf": [
            {
              "type": "object",
              "additionalProperties": false,
              "required": ["type", "params"],
              "properties": {
                "type": { "const": "prevalence_boundary" },
                "params": {
                  "type": "object",
                  "additionalProperties": false,
                  "required": ["source_globs", "target_globs", "max_prevalence", "min_support"],
                  "properties": {
                    "source_globs": {
                      "type": "array",
                      "minItems": 1,
                      "items": { "type": "string", "minLength": 1 }
                    },
                    "target_globs": {
                      "type": "array",
                      "minItems": 1,
                      "items": { "type": "string", "minLength": 1 }
                    },
                    "max_prevalence": {
                      "type": "number",
                      "minimum": 0
                    },
                    "min_support": {
                      "type": "integer",
                      "minimum": 1
                    }
                  }
                }
              }
            },
            {
              "type": "object",
              "additionalProperties": false,
              "required": ["type", "params"],
              "properties": {
                "type": { "const": "prevalence_package_boundary" },
                "params": {
                  "type": "object",
                  "additionalProperties": false,
                  "required": ["source_globs", "package_globs", "max_prevalence", "min_support"],
                  "properties": {
                    "source_globs": {
                      "type": "array",
                      "minItems": 1,
                      "items": { "type": "string", "minLength": 1 }
                    },
                    "package_globs": {
                      "type": "array",
                      "minItems": 1,
                      "items": { "type": "string", "minLength": 1 }
                    },
                    "max_prevalence": {
                      "type": "number",
                      "minimum": 0
                    },
                    "min_support": {
                      "type": "integer",
                      "minimum": 1
                    }
                  }
                }
              }
            },
            {
              "type": "object",
              "additionalProperties": false,
              "required": ["type", "params"],
              "properties": {
                "type": { "const": "construction_new_outside_root" },
                "params": {
                  "type": "object",
                  "additionalProperties": false,
                  "required": ["scope_globs", "service_globs", "allowed_new_globs", "service_name_regex", "max_prevalence", "min_support"],
                  "properties": {
                    "scope_globs": {
                      "type": "array",
                      "minItems": 1,
                      "items": { "type": "string", "minLength": 1 }
                    },
                    "service_globs": {
                      "type": "array",
                      "minItems": 1,
                      "items": { "type": "string", "minLength": 1 }
                    },
                    "allowed_new_globs": {
                      "type": "array",
                      "minItems": 1,
                      "items": { "type": "string", "minLength": 1 }
                    },
                    "service_name_regex": {
                      "type": "string",
                      "minLength": 1
                    },
                    "max_prevalence": {
                      "type": "number",
                      "minimum": 0
                    },
                    "min_support": {
                      "type": "integer",
                      "minimum": 1
                    }
                  }
                }
              }
            }
          ]
        }
      }
    },
    "rule_template": {
      "type": "object",
      "additionalProperties": false,
      "required": ["kind", "template", "defaults"],
      "properties": {
        "kind": {
          "type": "string",
          "const": "pattern"
        },
        "template": {
          "type": "string",
          "enum": ["dependency_constraint", "construction_policy"]
        },
        "defaults": {
          "type": "object",
          "additionalProperties": false,
          "required": ["severity"],
          "properties": {
            "severity": {
              "type": "string",
              "enum": ["warning", "error"]
            },
            "params": {
              "type": "object",
              "additionalProperties": { "type": "string" }
            }
          }
        }
      }
    }
  }
}
`
