package api

import (
	"encoding/json"
	"testing"
)

func TestExtractBalancedJSONObject(t *testing.T) {
	inner := `{"a": "brace } in string", "b": {"c": 1}}`
	wrapped := "prefix " + inner + " suffix"
	got, ok := extractBalancedJSONObject(wrapped)
	if !ok {
		t.Fatal("expected ok")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, got)
	}
	if m["a"] != "brace } in string" {
		t.Fatalf("unexpected a: %v", m["a"])
	}
}

func TestExtractJSON_CodeFence(t *testing.T) {
	s := "```json\n{\"x\":1}\n```"
	got := extractJSON(s)
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil || m["x"] != float64(1) {
		t.Fatalf("got %q err %v", got, err)
	}
}
