package config

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	// Test case 1: Environment variable is set
	os.Setenv("TEST_ENV_VAR", "test_value")
	value := GetEnv("TEST_ENV_VAR", "default_value")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', but got '%s'", value)
	}
	os.Unsetenv("TEST_ENV_VAR")

	// Test case 2: Environment variable is not set, should return default value
	value = GetEnv("NON_EXISTENT_VAR", "default_value")
	if value != "default_value" {
		t.Errorf("Expected 'default_value', but got '%s'", value)
	}
}
