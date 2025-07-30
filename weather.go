package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	geoEndpoint     = "https://api.openweathermap.org/geo/1.0/zip"
	weatherEndpoint = "https://api.openweathermap.org/data/3.0/onecall"
	openAIModel     = openai.GPT3Dot5Turbo

	// National Weather Service API endpoints
	nwsPointsEndpoint = "https://api.weather.gov/points"
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

// NWS API response types
type NWSPointResponse struct {
	Properties struct {
		Forecast            string `json:"forecast"`
		ForecastHourly      string `json:"forecastHourly"`
		ObservationStations string `json:"observationStations"`
		RelativeLocation    struct {
			Properties struct {
				City string `json:"city"`
			} `json:"properties"`
		} `json:"relativeLocation"`
	} `json:"properties"`
}

type NWSForecastResponse struct {
	Properties struct {
		Periods []struct {
			StartTime        string  `json:"startTime"`
			EndTime          string  `json:"endTime"`
			Temperature      float64 `json:"temperature"`
			WindSpeed        string  `json:"windSpeed"`
			WindDirection    string  `json:"windDirection"`
			ShortForecast    string  `json:"shortForecast"`
			DetailedForecast string  `json:"detailedForecast"`
			IsDaytime        bool    `json:"isDaytime"`
		} `json:"periods"`
	} `json:"properties"`
}

type NWSStationsResponse struct {
	Features []struct {
		Properties struct {
			StationIdentifier string `json:"stationIdentifier"`
		} `json:"properties"`
	} `json:"features"`
}

type NWSObservationResponse struct {
	Properties struct {
		Temperature struct {
			Value float64 `json:"value"`
		} `json:"temperature"`
		WindSpeed struct {
			Value float64 `json:"value"`
		} `json:"windSpeed"`
		RelativeHumidity struct {
			Value float64 `json:"value"`
		} `json:"relativeHumidity"`
		HeatIndex struct {
			Value float64 `json:"value"`
		} `json:"heatIndex"`
		TextDescription string `json:"textDescription"`
	} `json:"properties"`
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

	allowedHosts := []string{"openweathermap.org", "api.weather.gov"}
	allowed := false
	for _, host := range allowedHosts {
		if strings.HasSuffix(parsedURL.Host, host) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("URL host not allowed: %s", parsedURL.Host)
	}

	return nil
}

func getCoordinates(zip string, apiKey string) (float64, float64, string, error) {
	// If no API key is provided, use the Census geocoding API to get coordinates
	if apiKey == "" {
		return getNWSCoordinates(zip)
	}

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

// getNWSCoordinates uses a simple approximation for ZIP code to coordinates
// In a production app, you would use a proper geocoding service
func getNWSCoordinates(zip string) (float64, float64, string, error) {
	// This is a simplified approach - in production, use a proper geocoding service
	// For this example, we'll use a hardcoded mapping for a few ZIP codes
	zipCoords := map[string]struct {
		lat  float64
		lon  float64
		city string
	}{
		"90210": {34.0901, -118.4065, "Beverly Hills"},
		"10001": {40.7501, -73.9996, "New York"},
		"60601": {41.8841, -87.6277, "Chicago"},
		"02108": {42.3581, -71.0636, "Boston"},
		"94102": {37.7794, -122.4184, "San Francisco"},
		"98101": {47.6097, -122.3331, "Seattle"},
		"33101": {25.7743, -80.1937, "Miami"},
		"75201": {32.7795, -96.8022, "Dallas"},
		"77001": {29.7604, -95.3698, "Houston"},
		"85001": {33.4484, -112.0740, "Phoenix"},
	}

	if coords, ok := zipCoords[zip]; ok {
		return coords.lat, coords.lon, coords.city, nil
	}

	// For unknown ZIP codes, use a default location (NYC)
	return 40.7128, -74.0060, "New York", nil
}

func getWeather(lat, lon float64, apiKey string) (WeatherResponse, error) {
	// If no API key is provided, use the National Weather Service API
	if apiKey == "" {
		return getNWSWeather(lat, lon)
	}

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

// getNWSWeather fetches weather data from the National Weather Service API
func getNWSWeather(lat, lon float64) (WeatherResponse, error) {
	// Step 1: Get the forecast points URL
	pointsURL := fmt.Sprintf("%s/%.4f,%.4f", nwsPointsEndpoint, lat, lon)

	if err := validateURL(pointsURL); err != nil {
		return WeatherResponse{}, fmt.Errorf("URL validation failed: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", pointsURL, nil)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	// NWS API requires a User-Agent header
	req.Header.Set("User-Agent", "WeatherPipeline/1.0 (https://github.com/user/weather-pipeline)")

	pointsResp, err := client.Do(req)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error fetching NWS points: %w", err)
	}
	defer pointsResp.Body.Close()

	if pointsResp.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("NWS API error: status code %d", pointsResp.StatusCode)
	}

	var pointsData NWSPointResponse
	if err := json.NewDecoder(pointsResp.Body).Decode(&pointsData); err != nil {
		return WeatherResponse{}, fmt.Errorf("error decoding NWS points response: %w", err)
	}

	// Step 2: Get the forecast data
	forecastURL := pointsData.Properties.Forecast
	req, err = http.NewRequest("GET", forecastURL, nil)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error creating forecast request: %w", err)
	}
	req.Header.Set("User-Agent", "WeatherPipeline/1.0 (https://github.com/user/weather-pipeline)")

	forecastResp, err := client.Do(req)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error fetching NWS forecast: %w", err)
	}
	defer forecastResp.Body.Close()

	if forecastResp.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("NWS forecast API error: status code %d", forecastResp.StatusCode)
	}

	var forecastData NWSForecastResponse
	if err := json.NewDecoder(forecastResp.Body).Decode(&forecastData); err != nil {
		return WeatherResponse{}, fmt.Errorf("error decoding NWS forecast response: %w", err)
	}

	// Step 3: Get observation station
	stationsURL := pointsData.Properties.ObservationStations
	req, err = http.NewRequest("GET", stationsURL, nil)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error creating stations request: %w", err)
	}
	req.Header.Set("User-Agent", "WeatherPipeline/1.0 (https://github.com/user/weather-pipeline)")

	stationsResp, err := client.Do(req)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error fetching NWS stations: %w", err)
	}
	defer stationsResp.Body.Close()

	if stationsResp.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("NWS stations API error: status code %d", stationsResp.StatusCode)
	}

	var stationsData NWSStationsResponse
	if err := json.NewDecoder(stationsResp.Body).Decode(&stationsData); err != nil {
		return WeatherResponse{}, fmt.Errorf("error decoding NWS stations response: %w", err)
	}

	if len(stationsData.Features) == 0 {
		return WeatherResponse{}, fmt.Errorf("no observation stations found")
	}

	// Step 4: Get current observations
	stationID := stationsData.Features[0].Properties.StationIdentifier
	observationURL := fmt.Sprintf("https://api.weather.gov/stations/%s/observations/latest", stationID)
	req, err = http.NewRequest("GET", observationURL, nil)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error creating observation request: %w", err)
	}
	req.Header.Set("User-Agent", "WeatherPipeline/1.0 (https://github.com/user/weather-pipeline)")

	obsResp, err := client.Do(req)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error fetching NWS observations: %w", err)
	}
	defer obsResp.Body.Close()

	if obsResp.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("NWS observations API error: status code %d", obsResp.StatusCode)
	}

	var obsData NWSObservationResponse
	if err := json.NewDecoder(obsResp.Body).Decode(&obsData); err != nil {
		return WeatherResponse{}, fmt.Errorf("error decoding NWS observation response: %w", err)
	}

	// Convert NWS data to our standard WeatherResponse format
	weather := WeatherResponse{}

	// Current conditions
	weather.Current.Temp = celsiusToFahrenheit(obsData.Properties.Temperature.Value)

	// Use heat index if available, otherwise use temperature
	feelsLike := obsData.Properties.Temperature.Value
	if obsData.Properties.HeatIndex.Value != 0 {
		feelsLike = obsData.Properties.HeatIndex.Value
	}
	weather.Current.FeelsLike = celsiusToFahrenheit(feelsLike)

	// Convert m/s to mph for wind speed
	weather.Current.WindSpeed = obsData.Properties.WindSpeed.Value * 2.237

	// Convert relative humidity from percentage (0-100) to integer
	weather.Current.Humidity = int(obsData.Properties.RelativeHumidity.Value)

	// Set weather description
	weather.Current.Weather = []struct {
		Description string `json:"description"`
	}{{Description: obsData.Properties.TextDescription}}

	// Process forecast data
	weather.Daily = make([]struct {
		Dt   int64 `json:"dt"`
		Temp struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		} `json:"temp"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
	}, 0)

	// Group forecast periods by day (NWS provides 12-hour periods)
	dayMap := make(map[string]struct {
		date        time.Time
		minTemp     float64
		maxTemp     float64
		description string
	})

	for _, period := range forecastData.Properties.Periods {
		startTime, err := time.Parse(time.RFC3339, period.StartTime)
		if err != nil {
			continue
		}

		// Use date as key to group by day
		dateKey := startTime.Format("2006-01-02")

		day, exists := dayMap[dateKey]
		if !exists {
			day = struct {
				date        time.Time
				minTemp     float64
				maxTemp     float64
				description string
			}{
				date:        startTime,
				minTemp:     period.Temperature,
				maxTemp:     period.Temperature,
				description: period.ShortForecast,
			}
		} else {
			// For daytime periods, use as max temp
			if period.IsDaytime && period.Temperature > day.maxTemp {
				day.maxTemp = period.Temperature
				day.description = period.ShortForecast
			}

			// For nighttime periods, use as min temp
			if !period.IsDaytime && period.Temperature < day.minTemp {
				day.minTemp = period.Temperature
			}
		}

		dayMap[dateKey] = day
	}

	// Convert map to sorted array
	days := make([]string, 0, len(dayMap))
	for day := range dayMap {
		days = append(days, day)
	}

	// Sort days chronologically
	sort.Strings(days)

	// Add today as the first day
	today := time.Now().Format("2006-01-02")
	if dayData, ok := dayMap[today]; ok {
		dailyData := struct {
			Dt   int64 `json:"dt"`
			Temp struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
			} `json:"temp"`
			Weather []struct {
				Description string `json:"description"`
			} `json:"weather"`
		}{
			Dt: dayData.date.Unix(),
			Weather: []struct {
				Description string `json:"description"`
			}{{Description: dayData.description}},
		}
		dailyData.Temp.Min = dayData.minTemp
		dailyData.Temp.Max = dayData.maxTemp

		weather.Daily = append(weather.Daily, dailyData)
	}

	// Add forecast days
	for _, day := range days {
		if day == today {
			continue // Skip today as it's already added
		}

		dayData := dayMap[day]
		dailyData := struct {
			Dt   int64 `json:"dt"`
			Temp struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
			} `json:"temp"`
			Weather []struct {
				Description string `json:"description"`
			} `json:"weather"`
		}{
			Dt: dayData.date.Unix(),
			Weather: []struct {
				Description string `json:"description"`
			}{{Description: dayData.description}},
		}
		dailyData.Temp.Min = dayData.minTemp
		dailyData.Temp.Max = dayData.maxTemp

		weather.Daily = append(weather.Daily, dailyData)
	}

	return weather, nil
}

// celsiusToFahrenheit converts temperature from Celsius to Fahrenheit
func celsiusToFahrenheit(celsius float64) float64 {
	return celsius*9/5 + 32
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
