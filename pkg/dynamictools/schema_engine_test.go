package dynamictools

import (
	"context"
	"testing"
)

func TestInjectData_NilProps(t *testing.T) {
	root := &GeneratedComponent{
		ID:    "root",
		Type:  "text",
		Props: nil,
	}
	injectData(root, map[string]any{"k": "v"})
	if root.Props == nil || root.Props["k"] != "v" {
		t.Fatalf("expected props merged, got %#v", root.Props)
	}
}

func TestExecute_NilPropsInSchema(t *testing.T) {
	e := NewSchemaEngine(NewHostFunctions())
	tool := &DynamicTool{
		ChatSchema: GeneratedComponent{
			ID:    "root",
			Type:  "text",
			Props: nil,
		},
	}
	res, err := e.Execute(context.Background(), tool, map[string]any{"injected": 1})
	if err != nil {
		t.Fatal(err)
	}
	if res.Schema.Props["injected"] != 1 {
		t.Fatalf("expected injected key, got %#v", res.Schema.Props)
	}
}
