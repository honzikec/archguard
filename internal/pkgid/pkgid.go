package pkgid

import "strings"

// Canonical collapses package subpath imports to base package identifiers.
// Examples:
// - "react-dom/client" -> "react-dom"
// - "@scope/pkg/subpath" -> "@scope/pkg"
// - "node:fs/promises" -> "node:fs"
func Canonical(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Strip query/hash suffixes to support import-like specifiers in transpiled output.
	if idx := strings.IndexAny(raw, "?#"); idx >= 0 {
		raw = raw[:idx]
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.HasPrefix(raw, ".") || strings.HasPrefix(raw, "/") {
		return ""
	}

	prefix := ""
	body := raw
	if strings.HasPrefix(raw, "node:") {
		prefix = "node:"
		body = strings.TrimPrefix(raw, "node:")
	}

	if body == "" {
		return raw
	}

	parts := strings.Split(body, "/")
	if len(parts) == 0 {
		return raw
	}
	if strings.HasPrefix(parts[0], "@") {
		if len(parts) >= 2 {
			return prefix + parts[0] + "/" + parts[1]
		}
		return prefix + parts[0]
	}
	return prefix + parts[0]
}
