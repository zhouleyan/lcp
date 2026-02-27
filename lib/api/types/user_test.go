package types

import (
	"encoding/json"
	"testing"

	"lcp.io/lcp/lib/runtime"
)

func TestUserJSON(t *testing.T) {
	user := &User{
		TypeMeta:   runtime.TypeMeta{APIVersion: "v1", Kind: "User"},
		ObjectMeta: ObjectMeta{Name: "alice"},
		Spec: UserSpec{
			Username: "alice",
			Email:    "alice@example.com",
			Phone:    "+8613800138000",
		},
	}
	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got User
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Kind != "User" || got.Spec.Username != "alice" || got.Spec.Email != "alice@example.com" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestUserImplementsObject(t *testing.T) {
	var _ runtime.Object = &User{}
}

func TestUserListImplementsObject(t *testing.T) {
	var _ runtime.Object = &UserList{}
}
