package brief

const briefSchemaV1 = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "policies"],
  "properties": {
    "version": {
      "type": "integer",
      "const": 1
    },
    "project": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "roots": {
          "type": "array",
          "items": { "type": "string", "minLength": 1 }
        },
        "include": {
          "type": "array",
          "items": { "type": "string", "minLength": 1 }
        },
        "exclude": {
          "type": "array",
          "items": { "type": "string", "minLength": 1 }
        },
        "framework": { "type": "string", "minLength": 1 },
        "language": { "type": "string", "minLength": 1 },
        "tsconfig": { "type": "string", "minLength": 1 },
        "aliases": {
          "type": "object",
          "additionalProperties": {
            "type": "array",
            "items": { "type": "string", "minLength": 1 }
          }
        }
      }
    },
    "layers": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["id", "paths"],
        "properties": {
          "id": { "type": "string", "minLength": 1 },
          "paths": {
            "type": "array",
            "minItems": 1,
            "items": { "type": "string", "minLength": 1 }
          }
        }
      }
    },
    "policies": {
      "type": "array",
      "minItems": 1,
      "items": { "$ref": "#/$defs/policy" }
    }
  },
  "$defs": {
    "base_policy": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "type": {
          "type": "string",
          "enum": ["deny_import", "deny_package", "file_pattern", "no_cycle", "construction_policy"]
        },
        "severity": { "type": "string", "enum": ["warning", "error"] },
        "message": { "type": "string", "minLength": 1 },
        "except": {
          "type": "array",
          "items": { "type": "string", "minLength": 1 }
        }
      }
    },
    "deny_import_policy": {
      "unevaluatedProperties": false,
      "allOf": [
        { "$ref": "#/$defs/base_policy" },
        {
          "type": "object",
          "required": ["type", "from", "to"],
          "properties": {
            "type": { "const": "deny_import" },
            "from": {
              "type": "array",
              "minItems": 1,
              "items": { "type": "string", "minLength": 1 }
            },
            "to": {
              "type": "array",
              "minItems": 1,
              "items": { "type": "string", "minLength": 1 }
            }
          }
        }
      ]
    },
    "deny_package_policy": {
      "unevaluatedProperties": false,
      "allOf": [
        { "$ref": "#/$defs/base_policy" },
        {
          "type": "object",
          "required": ["type", "scope", "packages"],
          "properties": {
            "type": { "const": "deny_package" },
            "scope": {
              "type": "array",
              "minItems": 1,
              "items": { "type": "string", "minLength": 1 }
            },
            "packages": {
              "type": "array",
              "minItems": 1,
              "items": { "type": "string", "minLength": 1 }
            }
          }
        }
      ]
    },
    "file_pattern_policy": {
      "unevaluatedProperties": false,
      "allOf": [
        { "$ref": "#/$defs/base_policy" },
        {
          "type": "object",
          "required": ["type", "scope", "pattern"],
          "properties": {
            "type": { "const": "file_pattern" },
            "scope": {
              "type": "array",
              "minItems": 1,
              "items": { "type": "string", "minLength": 1 }
            },
            "pattern": { "type": "string", "minLength": 1 }
          }
        }
      ]
    },
    "no_cycle_policy": {
      "unevaluatedProperties": false,
      "allOf": [
        { "$ref": "#/$defs/base_policy" },
        {
          "type": "object",
          "required": ["type", "scope"],
          "properties": {
            "type": { "const": "no_cycle" },
            "scope": {
              "type": "array",
              "minItems": 1,
              "items": { "type": "string", "minLength": 1 }
            }
          }
        }
      ]
    },
    "construction_policy": {
      "unevaluatedProperties": false,
      "allOf": [
        { "$ref": "#/$defs/base_policy" },
        {
          "type": "object",
          "required": ["type", "scope", "services"],
          "properties": {
            "type": { "const": "construction_policy" },
            "scope": {
              "type": "array",
              "minItems": 1,
              "items": { "type": "string", "minLength": 1 }
            },
            "services": {
              "type": "array",
              "minItems": 1,
              "items": { "type": "string", "minLength": 1 }
            },
            "allow_in": {
              "type": "array",
              "items": { "type": "string", "minLength": 1 }
            },
            "service_name_regex": { "type": "string", "minLength": 1 }
          }
        }
      ]
    },
    "policy": {
      "oneOf": [
        { "$ref": "#/$defs/deny_import_policy" },
        { "$ref": "#/$defs/deny_package_policy" },
        { "$ref": "#/$defs/file_pattern_policy" },
        { "$ref": "#/$defs/no_cycle_policy" },
        { "$ref": "#/$defs/construction_policy" }
      ]
    }
  }
}
`
