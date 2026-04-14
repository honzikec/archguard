package pathutil

import "testing"

func TestUnmarshalJSONCSupportsCommentsAndTrailingCommas(t *testing.T) {
	var cfg tsConfigFile
	err := unmarshalJSONC([]byte(`/* generated */
{
  // tsconfig can include comments and trailing commas
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*",],
    },
  },
}`), &cfg)
	if err != nil {
		t.Fatalf("expected JSONC to decode, got error: %v", err)
	}
	if cfg.CompilerOptions.BaseURL != "." {
		t.Fatalf("unexpected baseUrl: %s", cfg.CompilerOptions.BaseURL)
	}
	if got := cfg.CompilerOptions.Paths["@/*"][0]; got != "src/*" {
		t.Fatalf("unexpected path target: %s", got)
	}
}

func TestUnmarshalJSONCPreservesCommentLikeStrings(t *testing.T) {
	var decoded struct {
		URL string `json:"url"`
	}
	err := unmarshalJSONC([]byte(`{
  "url": "http://example.com/api//v1",
}`), &decoded)
	if err != nil {
		t.Fatalf("expected JSONC to decode, got error: %v", err)
	}
	if decoded.URL != "http://example.com/api//v1" {
		t.Fatalf("unexpected URL: %s", decoded.URL)
	}
}
