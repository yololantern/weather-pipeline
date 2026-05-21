package main

import (
	"strings"
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

func makeWeatherResponse() WeatherResponse {
	w := WeatherResponse{}
	w.Current.Temp = 20.0
	w.Current.FeelsLike = 18.0
	w.Current.Humidity = 60
	w.Current.WindSpeed = 5.0
	w.Current.Weather = []struct {
		Description string `json:"description"`
	}{{Description: "clear sky"}}
	w.Daily = []struct {
		Dt   int64 `json:"dt"`
		Temp struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		} `json:"temp"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
	}{
		{Dt: 1700000000, Weather: []struct {
			Description string `json:"description"`
		}{{Description: "sunny"}}},
	}
	w.Daily[0].Temp.Min = 15.0
	w.Daily[0].Temp.Max = 25.0
	return w
}

func TestBuildForecastTextImperial(t *testing.T) {
	w := makeWeatherResponse()
	result := buildForecastText("Beverly Hills", "90210", w, false)
	if !strings.Contains(result, "°F") {
		t.Errorf("imperial output should contain °F, got:\n%s", result)
	}
	if strings.Contains(result, "°C") {
		t.Errorf("imperial output must not contain °C, got:\n%s", result)
	}
	if !strings.Contains(result, "mph") {
		t.Errorf("imperial output should contain mph, got:\n%s", result)
	}
}

func TestBuildForecastTextMetric(t *testing.T) {
	w := makeWeatherResponse()
	result := buildForecastText("Beverly Hills", "90210", w, true)
	if !strings.Contains(result, "°C") {
		t.Errorf("metric output should contain °C, got:\n%s", result)
	}
	if strings.Contains(result, "°F") {
		t.Errorf("metric output must not contain °F, got:\n%s", result)
	}
	if !strings.Contains(result, "m/s") {
		t.Errorf("metric output should contain m/s, got:\n%s", result)
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
