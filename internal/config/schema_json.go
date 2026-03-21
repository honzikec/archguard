package config

const configSchemaV1 = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["version"],
  "properties": {
    "version": {
      "type": "integer",
      "const": 1
    },
    "project": {
      "$ref": "#/$defs/project"
    },
    "rules": {
      "type": "array",
      "items": {
        "$ref": "#/$defs/rule"
      }
    }
  },
  "$defs": {
    "project": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "roots": {
          "type": "array",
          "items": { "type": "string" }
        },
        "include": {
          "type": "array",
          "items": { "type": "string" }
        },
        "exclude": {
          "type": "array",
          "items": { "type": "string" }
        },
        "framework": { "type": "string" },
        "language": { "type": "string" },
        "tsconfig": { "type": "string" },
        "aliases": {
          "type": "object",
          "additionalProperties": {
            "type": "array",
            "items": { "type": "string" }
          }
        }
      }
    },
    "rule": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "kind", "severity", "scope"],
      "properties": {
        "id": {
          "type": "string",
          "minLength": 1
        },
        "kind": {
          "type": "string",
          "enum": ["no_import", "no_package", "file_pattern", "no_cycle", "pattern"]
        },
        "severity": {
          "type": "string",
          "enum": ["warning", "error"]
        },
        "scope": {
          "type": "array",
          "minItems": 1,
          "items": { "type": "string" }
        },
        "target": {
          "type": "array",
          "items": { "type": "string" }
        },
        "except": {
          "type": "array",
          "items": { "type": "string" }
        },
        "template": { "type": "string" },
        "params": {
          "type": "object",
          "additionalProperties": { "type": "string" }
        },
        "message": { "type": "string" }
      }
    }
  }
}
`
