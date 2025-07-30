package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	geoEndpoint     = "https://api.openweathermap.org/geo/1.0/zip"
	weatherEndpoint = "https://api.openweathermap.org/data/3.0/onecall"
	openAIModel     = openai.GPT3Dot5Turbo
)

// WeatherData represents the processed weather data ready for pipeline
type WeatherData struct {
	LocationID   string    `json:"location_id"`
	LocationName string    `json:"location_name"`
	Timestamp    time.Time `json:"timestamp"`
	Temperature  float64   `json:"temperature"`
	FeelsLike    float64   `json:"feels_like"`
	TempMin      float64   `json:"temp_min"`
	TempMax      float64   `json:"temp_max"`
	Humidity     int       `json:"humidity"`
	WindSpeed    float64   `json:"wind_speed"`
	Condition    string    `json:"condition"`
	ForecastDays int       `json:"forecast_days"`
	Forecast     []struct {
		Date      time.Time `json:"date"`
		TempMin   float64   `json:"temp_min"`
		TempMax   float64   `json:"temp_max"`
		Condition string    `json:"condition"`
	} `json:"forecast"`
	Summary  string `json:"summary,omitempty"`
	IsMetric bool   `json:"is_metric"`
}

type GeoResponse struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name"`
}

type WeatherResponse struct {
	Current struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Humidity  int     `json:"humidity"`
		WindSpeed float64 `json:"wind_speed"`
		Weather   []struct {
			Description string `json:"description"`
		} `json:"weather"`
	} `json:"current"`
	Daily []struct {
		Dt   int64 `json:"dt"`
		Temp struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		} `json:"temp"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
	} `json:"daily"`
}

func isValidZip(zip string) bool {
	if len(zip) != 5 {
		return false
	}
	for _, r := range zip {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use HTTPS")
	}

	if !strings.HasSuffix(parsedURL.Host, "openweathermap.org") {
		return fmt.Errorf("URL host not allowed: %s", parsedURL.Host)
	}

	return nil
}

func getCoordinates(zip string, apiKey string) (float64, float64, string, error) {
	urlStr := fmt.Sprintf("%s?zip=%s,US&appid=%s", geoEndpoint, zip, apiKey)

	if err := validateURL(urlStr); err != nil {
		return 0, 0, "", fmt.Errorf("URL validation failed: %w", err)
	}

	resp, err := http.Get(urlStr) //nolint
	if err != nil {
		return 0, 0, "", fmt.Errorf("error getting geocode: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, "", fmt.Errorf("geocoding API error: status code %d", resp.StatusCode)
	}

	var geo GeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return 0, 0, "", fmt.Errorf("error decoding geocode response: %w", err)
	}
	return geo.Lat, geo.Lon, geo.Name, nil
}

func getWeather(lat, lon float64, apiKey string) (WeatherResponse, error) {
	units := "imperial"
	if false { // Default is imperial, change based on config in real implementation
		units = "metric"
	}
	urlStr := fmt.Sprintf("%s?lat=%f&lon=%f&exclude=minutely,hourly,alerts&units=%s&appid=%s", weatherEndpoint, lat, lon, units, apiKey)

	if err := validateURL(urlStr); err != nil {
		return WeatherResponse{}, fmt.Errorf("URL validation failed: %w", err)
	}

	resp, err := http.Get(urlStr) //nolint
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error fetching weather: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("weather API error: status code %d", resp.StatusCode)
	}

	var weather WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		return WeatherResponse{}, fmt.Errorf("error decoding weather response: %w", err)
	}
	return weather, nil
}

func buildForecastText(city, zip string, w WeatherResponse) string {
	unit := "°F"
	windUnit := "mph"
	if false { // Default is imperial, change based on config in real implementation
		unit = "°C"
		windUnit = "m/s"
	}
	result := fmt.Sprintf("Location: %s (ZIP: %s)\n", city, zip)
	result += fmt.Sprintf("Now: %.1f%s, feels like %.1f%s, %s\n",
		w.Current.Temp, unit, w.Current.FeelsLike, unit, w.Current.Weather[0].Description)
	result += fmt.Sprintf("Humidity: %d%%, Wind: %.1f %s\n", w.Current.Humidity, w.Current.WindSpeed, windUnit)
	result += "7-Day Forecast:\n"
	days := len(w.Daily)
	if days > 7 {
		days = 7
	}
	for i := 1; i < days; i++ {
		day := w.Daily[i]
		date := time.Unix(day.Dt, 0).Format("Mon Jan 2")
		result += fmt.Sprintf("%s: Min %.1f%s, Max %.1f%s, %s\n",
			date, day.Temp.Min, unit, day.Temp.Max, unit, day.Weather[0].Description)
	}
	return result
}

func summarizeForecast(data string) string {
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		return "Missing OPENAI_API_KEY environment variable"
	}

	client := openai.NewClient(openAIKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openAIModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful and friendly weather forecaster writing short reports.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Based on the following structured weather data, write a 3-5 sentence friendly and clear weather summary:\n\n" + data,
				},
			},
		},
	)

	if err != nil {
		return fmt.Sprintf("OpenAI API error: %v", err)
	}

	return resp.Choices[0].Message.Content
}
