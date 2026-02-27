package types

import (
	"encoding/json"
	"testing"

	"lcp.io/lcp/lib/runtime"
)

func TestNamespaceJSON(t *testing.T) {
	ns := &Namespace{
		TypeMeta:   runtime.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: ObjectMeta{Name: "my-team"},
		Spec: NamespaceSpec{
			OwnerID:    "1",
			Visibility: "private",
		},
	}
	data, err := json.Marshal(ns)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Namespace
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Kind != "Namespace" || got.Spec.OwnerID != "1" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestNamespaceImplementsObject(t *testing.T) {
	var _ runtime.Object = &Namespace{}
	var _ runtime.Object = &NamespaceMember{}
}
