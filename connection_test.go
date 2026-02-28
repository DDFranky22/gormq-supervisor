package main

import (
	"os"
	"testing"
)

func TestReplaceEnvVar_WithValidEnvVar(t *testing.T) {
	// Set up a test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result := replaceEnvVar("${TEST_VAR}")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}
}

func TestReplaceEnvVar_WithUnsetEnvVar(t *testing.T) {
	// Ensure the variable is not set
	os.Unsetenv("UNSET_VAR")

	result := replaceEnvVar("${UNSET_VAR}")
	if result != "" {
		t.Errorf("Expected empty string for unset variable, got '%s'", result)
	}
}

func TestReplaceEnvVar_WithPlainString(t *testing.T) {
	result := replaceEnvVar("plain_string")
	if result != "plain_string" {
		t.Errorf("Expected 'plain_string', got '%s'", result)
	}
}

func TestReplaceEnvVar_WithEmptyString(t *testing.T) {
	result := replaceEnvVar("")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestReplaceEnvVar_WithPartialFormat(t *testing.T) {
	// Test with only prefix
	result := replaceEnvVar("${INCOMPLETE")
	if result != "${INCOMPLETE" {
		t.Errorf("Expected '${INCOMPLETE', got '%s'", result)
	}

	// Test with only suffix
	result = replaceEnvVar("INCOMPLETE}")
	if result != "INCOMPLETE}" {
		t.Errorf("Expected 'INCOMPLETE}', got '%s'", result)
	}
}

func TestReplaceEnvVar_WithNestedBraces(t *testing.T) {
	os.Setenv("NESTED", "value")
	defer os.Unsetenv("NESTED")

	// Input "${${NESTED}}" is processed as:
	// 1. HasPrefix("${") and HasSuffix("}") → true
	// 2. TrimRight("${${NESTED}}", "}") → "${${NESTED}"
	// 3. TrimLeft("${${NESTED}", "${") → "NESTED" (removes all $, { chars from left)
	// 4. os.Getenv("NESTED") → "value"
	result := replaceEnvVar("${${NESTED}}")
	if result != "value" {
		t.Errorf("Expected 'value' for nested format (trims to 'NESTED'), got '%s'", result)
	}
}

func TestConnectionConfig_ReplaceEnvVariables(t *testing.T) {
	os.Setenv("TEST_USER", "myuser")
	os.Setenv("TEST_PASS", "mypassword")
	defer os.Unsetenv("TEST_USER")
	defer os.Unsetenv("TEST_PASS")

	config := &ConnectionConfig{
		Name:     "test_connection",
		Endpoint: "http://localhost:15672",
		Username: "${TEST_USER}",
		Password: "${TEST_PASS}",
		Vhost:    "/",
	}

	result := config.replaceEnvVariables()

	if result.Username != "myuser" {
		t.Errorf("Expected username 'myuser', got '%s'", result.Username)
	}
	if result.Password != "mypassword" {
		t.Errorf("Expected password 'mypassword', got '%s'", result.Password)
	}
	// Other fields should remain unchanged
	if result.Name != "test_connection" {
		t.Errorf("Expected name 'test_connection', got '%s'", result.Name)
	}
	if result.Endpoint != "http://localhost:15672" {
		t.Errorf("Expected endpoint 'http://localhost:15672', got '%s'", result.Endpoint)
	}
}

func TestConnectionConfig_ReplaceEnvVariables_WithPlainValues(t *testing.T) {
	config := &ConnectionConfig{
		Name:     "test_connection",
		Endpoint: "http://localhost:15672",
		Username: "plainuser",
		Password: "plainpass",
		Vhost:    "/",
	}

	result := config.replaceEnvVariables()

	if result.Username != "plainuser" {
		t.Errorf("Expected username 'plainuser', got '%s'", result.Username)
	}
	if result.Password != "plainpass" {
		t.Errorf("Expected password 'plainpass', got '%s'", result.Password)
	}
}

func TestConnectionConfig_ReplaceEnvVariables_ReturnsSamePointer(t *testing.T) {
	config := &ConnectionConfig{
		Name:     "test",
		Username: "user",
		Password: "pass",
	}

	result := config.replaceEnvVariables()

	if result != config {
		t.Error("Expected replaceEnvVariables to return the same pointer")
	}
}
