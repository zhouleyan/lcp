package modules

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestModuleDebug_SimpleMsg(t *testing.T) {
	opts := ExecOptions{
		Args:     map[string]any{"msg": "hello world"},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	stdout, stderr, err := ModuleDebug(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "hello world" {
		t.Errorf("expected stdout 'hello world', got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
}

func TestModuleDebug_TemplateMsg(t *testing.T) {
	opts := ExecOptions{
		Args:     map[string]any{"msg": "{{ .name }}"},
		Host:     "testhost",
		Variable: newTestVariableWithVars("testhost", map[string]any{"name": "test"}),
	}

	stdout, _, err := ModuleDebug(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "test" {
		t.Errorf("expected stdout 'test', got %q", stdout)
	}
}

func TestModuleDebug_MapMsg(t *testing.T) {
	msgMap := map[string]any{
		"key1": "value1",
		"key2": float64(42),
	}
	opts := ExecOptions{
		Args:     map[string]any{"msg": msgMap},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	stdout, _, err := ModuleDebug(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "key1") || !strings.Contains(stdout, "value1") {
		t.Errorf("expected JSON output containing key1/value1, got %q", stdout)
	}
	if !strings.Contains(stdout, "key2") || !strings.Contains(stdout, "42") {
		t.Errorf("expected JSON output containing key2/42, got %q", stdout)
	}
}

func TestModuleDebug_LogOutput(t *testing.T) {
	var buf bytes.Buffer
	opts := ExecOptions{
		Args:      map[string]any{"msg": "log this"},
		Host:      "testhost",
		Variable:  newTestVariable("testhost"),
		LogOutput: &buf,
	}

	stdout, _, err := ModuleDebug(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "log this" {
		t.Errorf("expected stdout 'log this', got %q", stdout)
	}
	if !strings.Contains(buf.String(), "DEBUG: log this") {
		t.Errorf("expected log output to contain 'DEBUG: log this', got %q", buf.String())
	}
}

func TestModuleDebug_NoMsg(t *testing.T) {
	opts := ExecOptions{
		Args:     map[string]any{},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	_, _, err := ModuleDebug(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for missing msg argument, got nil")
	}
	if !strings.Contains(err.Error(), "msg argument required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestModuleDebug_NumberMsg(t *testing.T) {
	opts := ExecOptions{
		Args:     map[string]any{"msg": 42},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	stdout, _, err := ModuleDebug(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "42" {
		t.Errorf("expected stdout '42', got %q", stdout)
	}
}

func TestModuleDebug_SliceMsg(t *testing.T) {
	opts := ExecOptions{
		Args:     map[string]any{"msg": []string{"a", "b", "c"}},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	stdout, _, err := ModuleDebug(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "a") || !strings.Contains(stdout, "b") || !strings.Contains(stdout, "c") {
		t.Errorf("expected JSON array output, got %q", stdout)
	}
}

func TestModuleDebug_Registered(t *testing.T) {
	fn := FindModule("debug")
	if fn == nil {
		t.Fatal("expected 'debug' module to be registered, got nil")
	}
}

// ======================================================================
// Assert module tests
// ======================================================================

func TestModuleAssert_Pass(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"that": []any{"{{ eq 1 1 }}"},
		},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	stdout, _, err := ModuleAssert(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "all assertions passed" {
		t.Errorf("expected 'all assertions passed', got %q", stdout)
	}
}

func TestModuleAssert_PassWithSuccessMsg(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"that":        []any{"{{ eq 1 1 }}"},
			"success_msg": "great success",
		},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	stdout, _, err := ModuleAssert(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "great success" {
		t.Errorf("expected 'great success', got %q", stdout)
	}
}

func TestModuleAssert_Fail(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"that":     []any{"{{ eq 1 2 }}"},
			"fail_msg": "values are not equal",
		},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	_, _, err := ModuleAssert(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for failed assertion, got nil")
	}
	if !strings.Contains(err.Error(), "values are not equal") {
		t.Errorf("expected error to contain 'values are not equal', got %v", err)
	}
}

func TestModuleAssert_FailDefaultMsg(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"that": []any{"{{ eq 1 2 }}"},
		},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	_, _, err := ModuleAssert(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for failed assertion, got nil")
	}
	if !strings.Contains(err.Error(), "assertion failed") {
		t.Errorf("expected error to contain 'assertion failed', got %v", err)
	}
}

func TestModuleAssert_NoThat(t *testing.T) {
	opts := ExecOptions{
		Args:     map[string]any{},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	_, _, err := ModuleAssert(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for missing 'that' argument, got nil")
	}
	if !strings.Contains(err.Error(), "'that' argument required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestModuleAssert_MultipleConditions(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"that": []any{
				"{{ eq 1 1 }}",
				"{{ ne 1 2 }}",
			},
		},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	stdout, _, err := ModuleAssert(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "all assertions passed" {
		t.Errorf("expected 'all assertions passed', got %q", stdout)
	}
}

func TestModuleAssert_MultipleConditionsOneFails(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"that": []any{
				"{{ eq 1 1 }}",
				"{{ eq 1 2 }}",
			},
			"fail_msg": "second condition failed",
		},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	_, _, err := ModuleAssert(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when one condition fails, got nil")
	}
	if !strings.Contains(err.Error(), "second condition failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestModuleAssert_SingleStringCondition(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"that": "{{ eq 1 1 }}",
		},
		Host:     "testhost",
		Variable: newTestVariable("testhost"),
	}

	stdout, _, err := ModuleAssert(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "all assertions passed" {
		t.Errorf("expected 'all assertions passed', got %q", stdout)
	}
}

func TestModuleAssert_WithVariables(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"that": []any{"{{ eq .status \"active\" }}"},
		},
		Host:     "testhost",
		Variable: newTestVariableWithVars("testhost", map[string]any{"status": "active"}),
	}

	stdout, _, err := ModuleAssert(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "all assertions passed" {
		t.Errorf("expected 'all assertions passed', got %q", stdout)
	}
}

func TestModuleAssert_Registered(t *testing.T) {
	fn := FindModule("assert")
	if fn == nil {
		t.Fatal("expected 'assert' module to be registered, got nil")
	}
}

// ======================================================================
// toStringSlice tests
// ======================================================================

func TestToStringSlice_AnySlice(t *testing.T) {
	input := []any{"a", "b", "c"}
	result := toStringSlice(input)
	if len(result) != 3 || result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("expected [a b c], got %v", result)
	}
}

func TestToStringSlice_StringSlice(t *testing.T) {
	input := []string{"x", "y"}
	result := toStringSlice(input)
	if len(result) != 2 || result[0] != "x" || result[1] != "y" {
		t.Errorf("expected [x y], got %v", result)
	}
}

func TestToStringSlice_SingleString(t *testing.T) {
	result := toStringSlice("hello")
	if len(result) != 1 || result[0] != "hello" {
		t.Errorf("expected [hello], got %v", result)
	}
}

func TestToStringSlice_Nil(t *testing.T) {
	result := toStringSlice(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestToStringSlice_NonStringItems(t *testing.T) {
	// Non-string items in []any should be skipped.
	input := []any{"a", 42, "b"}
	result := toStringSlice(input)
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Errorf("expected [a b], got %v", result)
	}
}
