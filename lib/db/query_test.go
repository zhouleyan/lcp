package db

import (
	"testing"
)

func TestBuildWhereClause_Empty(t *testing.T) {
	spec := ListSpec{
		Fields: map[string]Field{
			"status": {Column: "u.status", Op: Eq},
		},
	}
	clause, args := BuildWhereClause(nil, spec, 1)
	if clause != "" {
		t.Errorf("expected empty clause, got %q", clause)
	}
	if args != nil {
		t.Errorf("expected nil args, got %v", args)
	}
}

func TestBuildWhereClause_EqFilter(t *testing.T) {
	spec := ListSpec{
		Fields: map[string]Field{
			"status": {Column: "u.status", Op: Eq},
		},
	}
	clause, args := BuildWhereClause(map[string]any{"status": "active"}, spec, 1)
	if clause != " WHERE u.status = $1" {
		t.Errorf("unexpected clause: %q", clause)
	}
	if len(args) != 1 || args[0] != "active" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestBuildWhereClause_LikeFilter(t *testing.T) {
	spec := ListSpec{
		Fields: map[string]Field{
			"username": {Column: "u.username", Op: Like},
		},
	}
	clause, args := BuildWhereClause(map[string]any{"username": "ali%ce"}, spec, 1)
	if clause != " WHERE u.username ILIKE '%' || $1 || '%'" {
		t.Errorf("unexpected clause: %q", clause)
	}
	// Should escape the % in the value
	if len(args) != 1 || args[0] != `ali\%ce` {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestBuildWhereClause_UnknownFilter(t *testing.T) {
	spec := ListSpec{
		Fields: map[string]Field{
			"status": {Column: "u.status", Op: Eq},
		},
	}
	clause, args := BuildWhereClause(map[string]any{"unknown": "value"}, spec, 1)
	if clause != "" {
		t.Errorf("expected empty clause for unknown filter, got %q", clause)
	}
	if args != nil {
		t.Errorf("expected nil args, got %v", args)
	}
}

func TestBuildWhereClause_ArgStart(t *testing.T) {
	spec := ListSpec{
		Fields: map[string]Field{
			"status": {Column: "u.status", Op: Eq},
		},
	}
	clause, args := BuildWhereClause(map[string]any{"status": "active"}, spec, 3)
	if clause != " WHERE u.status = $3" {
		t.Errorf("unexpected clause: %q", clause)
	}
	if len(args) != 1 {
		t.Errorf("unexpected args count: %d", len(args))
	}
}

func TestBuildOrderBy_KnownField(t *testing.T) {
	spec := ListSpec{
		Fields: map[string]Field{
			"username": {Column: "u.username", Op: Like},
		},
		DefaultSort: "u.created_at",
	}
	result := BuildOrderBy("username", "asc", spec)
	if result != " ORDER BY u.username ASC" {
		t.Errorf("unexpected order: %q", result)
	}
}

func TestBuildOrderBy_DefaultSort(t *testing.T) {
	spec := ListSpec{
		Fields: map[string]Field{
			"username": {Column: "u.username", Op: Like},
		},
		DefaultSort: "u.created_at",
	}
	result := BuildOrderBy("", "", spec)
	if result != " ORDER BY u.created_at DESC" {
		t.Errorf("unexpected order: %q", result)
	}
}

func TestBuildOrderBy_UnknownField(t *testing.T) {
	spec := ListSpec{
		Fields: map[string]Field{
			"username": {Column: "u.username", Op: Like},
		},
		DefaultSort: "u.created_at",
	}
	result := BuildOrderBy("injected_column", "asc", spec)
	if result != " ORDER BY u.created_at ASC" {
		t.Errorf("unexpected order: %q", result)
	}
}
