package main

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	// Test with empty ZIP codes
	config := &Config{
		APIKey:       "test-key",
		OutputFormat: FormatText,
	}
	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for empty ZIP codes, got nil")
	}

	// Test with valid ZIP code
	config = &Config{
		APIKey:       "test-key",
		ZipCodes:     []string{"90210"},
		OutputFormat: FormatText,
	}
	err = ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	// Test with invalid ZIP code
	config = &Config{
		APIKey:       "test-key",
		ZipCodes:     []string{"invalid"},
		OutputFormat: FormatText,
	}
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid ZIP code, got nil")
	}

	// Test with empty API key (should now be valid with NWS fallback)
	config = &Config{
		APIKey:       "",
		ZipCodes:     []string{"90210"},
		OutputFormat: FormatText,
	}
	err = ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for empty API key (NWS fallback), got: %v", err)
	}
}

func TestIsValidZip(t *testing.T) {
	validZips := []string{"90210", "10001", "60601"}
	invalidZips := []string{"9021", "1000a", "abcde", "123456"}

	for _, zip := range validZips {
		if !isValidZip(zip) {
			t.Errorf("Expected %s to be a valid ZIP code", zip)
		}
	}

	for _, zip := range invalidZips {
		if isValidZip(zip) {
			t.Errorf("Expected %s to be an invalid ZIP code", zip)
		}
	}
}
