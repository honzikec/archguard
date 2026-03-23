package pathutil

import (
	"testing"
)

func TestStripJSONComments(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"no comments",
			`{"a": 1}`,
			`{"a": 1}`,
		},
		{
			"line comment",
			`{"a": 1} // comment`,
			`{"a": 1} `,
		},
		{
			"block comment",
			`{"a": /* block */ 1}`,
			`{"a":  1}`,
		},
		{
			"comment inside string",
			`{"url": "http://example.com/api//v1"}`,
			`{"url": "http://example.com/api//v1"}`,
		},
		{
			"block comment inside string",
			`{"a": "hello /* world */"}`,
			`{"a": "hello /* world */"}`,
		},
		{
			"escaped quotes inside string",
			`{"a": "\"//not a comment\""}`,
			`{"a": "\"//not a comment\""}`,
		},
		{
			"multiline block comment",
			"{\n/*\nblock\n*/\"a\": 1\n}",
			"{\n\n\n\"a\": 1\n}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := string(stripJSONComments([]byte(tc.input)))
			if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestStripJSONTrailingCommas(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"no trailing",
			`{"a": 1, "b": 2}`,
			`{"a": 1, "b": 2}`,
		},
		{
			"trailing object comma",
			`{"a": 1, "b": 2,}`,
			`{"a": 1, "b": 2}`,
		},
		{
			"trailing array comma",
			`["a", "b",]`,
			`["a", "b"]`,
		},
		{
			"trailing comma with whitespace",
			`{"a": 1, 
			}`,
			`{"a": 1 
			}`,
		},
		{
			"comma inside string",
			`{"a": "hello, world"}`,
			`{"a": "hello, world"}`,
		},
		{
			"escaped quote before trailing comma",
			`["\"",]`,
			`["\""]`,
		},
		{
			"multiple trailing commas in layered structures",
			`{"a": [1, 2, ], "b": {"c": 3, }, }`,
			`{"a": [1, 2 ], "b": {"c": 3 } }`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := string(stripJSONTrailingCommas([]byte(tc.input)))
			if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}
