package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// OutputFormat defines the format for data output
type OutputFormat string

const (
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatText  OutputFormat = "text"
	FormatKafka OutputFormat = "kafka"
)

// Config holds application configuration
type Config struct {
	APIKey       string
	ZipCodes     []string
	OutputFormat OutputFormat
	OutputPath   string
	IsMetric     bool
	KafkaBroker  string
	KafkaTopic   string
	Interval     time.Duration
	Verbose      bool
}

// ParseFlags parses command line flags and returns a Config
func ParseFlags() *Config {
	config := &Config{}

	// Define flags
	flag.StringVar(&config.APIKey, "api-key", os.Getenv("OWM_API_KEY"), "OpenWeatherMap API key")
	zipCodesStr := flag.String("zip-codes", "", "Comma-separated list of ZIP codes")
	format := flag.String("format", "text", "Output format: text, json, csv, kafka")
	flag.StringVar(&config.OutputPath, "output", "", "Output file path (stdout if empty)")
	flag.BoolVar(&config.IsMetric, "metric", false, "Use metric units (Celsius)")
	flag.StringVar(&config.KafkaBroker, "kafka-broker", "localhost:9092", "Kafka broker address")
	flag.StringVar(&config.KafkaTopic, "kafka-topic", "weather-data", "Kafka topic for output")
	interval := flag.Int("interval", 0, "Polling interval in seconds (0 for one-time run)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	// Parse flags
	flag.Parse()

	// Process ZIP codes
	if *zipCodesStr != "" {
		config.ZipCodes = strings.Split(*zipCodesStr, ",")
	} else {
		// Check for positional argument
		if flag.NArg() > 0 {
			config.ZipCodes = []string{flag.Arg(0)}
		}
	}

	// Set output format
	config.OutputFormat = OutputFormat(*format)

	// Set interval
	if *interval > 0 {
		config.Interval = time.Duration(*interval) * time.Second
	}

	return config
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("OpenWeatherMap API key is required (use -api-key flag or OWM_API_KEY env var)")
	}

	if len(config.ZipCodes) == 0 {
		return fmt.Errorf("at least one ZIP code is required")
	}

	for _, zip := range config.ZipCodes {
		if !isValidZip(zip) {
			return fmt.Errorf("invalid ZIP code format: %s", zip)
		}
	}

	switch config.OutputFormat {
	case FormatJSON, FormatCSV, FormatText, FormatKafka:
		// Valid format
	default:
		return fmt.Errorf("invalid output format: %s", config.OutputFormat)
	}

	if config.OutputFormat == FormatKafka && config.KafkaBroker == "" {
		return fmt.Errorf("kafka broker is required when using kafka output format")
	}

	return nil
}

// ProcessLocations processes all locations in the configuration
func ProcessLocations(config *Config) {
	var weatherDataList []WeatherData

	for _, zip := range config.ZipCodes {
		if config.Verbose {
			log.Printf("Processing ZIP code: %s", zip)
		}

		// Get weather data
		weatherData, err := GetLocationWeather(zip, config)
		if err != nil {
			log.Printf("Error processing %s: %v", zip, err)
			continue
		}

		weatherDataList = append(weatherDataList, weatherData)

		// Output data immediately if not collecting for batch output
		if config.OutputFormat != FormatJSON && config.OutputFormat != FormatCSV {
			OutputWeatherData(weatherData, config)
		}
	}

	// Batch output for formats that make sense in batch
	if len(weatherDataList) > 0 && (config.OutputFormat == FormatJSON || config.OutputFormat == FormatCSV) {
		OutputWeatherDataBatch(weatherDataList, config)
	}
}

// GetLocationWeather retrieves and processes weather data for a location
func GetLocationWeather(zip string, config *Config) (WeatherData, error) {
	var weatherData WeatherData

	// Get coordinates
	lat, lon, city, err := getCoordinates(zip, config.APIKey)
	if err != nil {
		return weatherData, fmt.Errorf("failed to get coordinates: %w", err)
	}

	// Get weather
	weather, err := getWeather(lat, lon, config.APIKey)
	if err != nil {
		return weatherData, fmt.Errorf("failed to get weather: %w", err)
	}

	// Process data into standardized format
	weatherData = WeatherData{
		LocationID:   zip,
		LocationName: city,
		Timestamp:    time.Now(),
		Temperature:  weather.Current.Temp,
		FeelsLike:    weather.Current.FeelsLike,
		Humidity:     weather.Current.Humidity,
		WindSpeed:    weather.Current.WindSpeed,
		IsMetric:     config.IsMetric,
	}

	if len(weather.Current.Weather) > 0 {
		weatherData.Condition = weather.Current.Weather[0].Description
	}

	// Process forecast data
	forecastDays := len(weather.Daily)
	if forecastDays > 7 {
		forecastDays = 7
	}

	weatherData.ForecastDays = forecastDays - 1 // Excluding today
	weatherData.Forecast = make([]struct {
		Date      time.Time `json:"date"`
		TempMin   float64   `json:"temp_min"`
		TempMax   float64   `json:"temp_max"`
		Condition string    `json:"condition"`
	}, forecastDays-1)

	for i := 1; i < forecastDays; i++ {
		day := weather.Daily[i]
		weatherData.Forecast[i-1] = struct {
			Date      time.Time `json:"date"`
			TempMin   float64   `json:"temp_min"`
			TempMax   float64   `json:"temp_max"`
			Condition string    `json:"condition"`
		}{
			Date:      time.Unix(day.Dt, 0),
			TempMin:   day.Temp.Min,
			TempMax:   day.Temp.Max,
			Condition: day.Weather[0].Description,
		}
	}

	// Generate summary if needed for specific output formats
	if config.OutputFormat == FormatText {
		forecastText := buildForecastText(city, zip, weather)
		if config.Verbose {
			log.Println("Generating AI summary")
		}
		weatherData.Summary = summarizeForecast(forecastText)
	}

	return weatherData, nil
}

// OutputWeatherData outputs a single weather data record
func OutputWeatherData(data WeatherData, config *Config) {
	switch config.OutputFormat {
	case FormatText:
		OutputTextFormat(data, config)
	case FormatJSON:
		// Single JSON records handled in batch
	case FormatCSV:
		// CSV records handled in batch
	case FormatKafka:
		SendToKafka(data, config)
	}
}

// OutputWeatherDataBatch outputs a batch of weather data records
func OutputWeatherDataBatch(dataList []WeatherData, config *Config) {
	switch config.OutputFormat {
	case FormatJSON:
		OutputJSONFormat(dataList, config)
	case FormatCSV:
		OutputCSVFormat(dataList, config)
	}
}

// OutputTextFormat outputs weather data in human-readable text format
func OutputTextFormat(data WeatherData, config *Config) {
	unit := "°F"
	windUnit := "mph"
	if config.IsMetric {
		unit = "°C"
		windUnit = "m/s"
	}

	fmt.Printf("\n📍 Weather for %s (ZIP: %s)\n", data.LocationName, data.LocationID)
	fmt.Println("-----------------------------------")
	fmt.Printf("Now: %.1f%s, feels like %.1f%s, %s\n",
		data.Temperature, unit, data.FeelsLike, unit, data.Condition)
	fmt.Printf("Humidity: %d%%, Wind: %.1f %s\n", data.Humidity, data.WindSpeed, windUnit)

	fmt.Println("\n📆 Forecast:")
	for _, day := range data.Forecast {
		date := day.Date.Format("Mon Jan 2")
		fmt.Printf("%s: Min %.1f%s, Max %.1f%s, %s\n",
			date, day.TempMin, unit, day.TempMax, unit, day.Condition)
	}

	if data.Summary != "" {
		fmt.Println("\n📝 AI-Generated Forecast:")
		fmt.Println(data.Summary)
	}
}

// OutputJSONFormat outputs weather data in JSON format
func OutputJSONFormat(dataList []WeatherData, config *Config) {
	var output *os.File
	var err error

	if config.OutputPath == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(config.OutputPath)
		if err != nil {
			log.Printf("Error creating output file: %v", err)
			return
		}
		defer output.Close()
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")

	if len(dataList) == 1 {
		// Single record
		if err := encoder.Encode(dataList[0]); err != nil {
			log.Printf("Error encoding JSON: %v", err)
		}
	} else {
		// Multiple records
		if err := encoder.Encode(dataList); err != nil {
			log.Printf("Error encoding JSON: %v", err)
		}
	}
}

// OutputCSVFormat outputs weather data in CSV format
func OutputCSVFormat(dataList []WeatherData, config *Config) {
	var output *os.File
	var err error

	if config.OutputPath == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(config.OutputPath)
		if err != nil {
			log.Printf("Error creating output file: %v", err)
			return
		}
		defer output.Close()
	}

	writer := csv.NewWriter(output)
	defer writer.Flush()

	// Write header
	header := []string{
		"location_id", "location_name", "timestamp", "temperature",
		"feels_like", "humidity", "wind_speed", "condition", "is_metric",
	}
	if err := writer.Write(header); err != nil {
		log.Printf("Error writing CSV header: %v", err)
		return
	}

	// Write data rows
	for _, data := range dataList {
		row := []string{
			data.LocationID,
			data.LocationName,
			data.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%.1f", data.Temperature),
			fmt.Sprintf("%.1f", data.FeelsLike),
			fmt.Sprintf("%d", data.Humidity),
			fmt.Sprintf("%.1f", data.WindSpeed),
			data.Condition,
			fmt.Sprintf("%t", data.IsMetric),
		}

		if err := writer.Write(row); err != nil {
			log.Printf("Error writing CSV row: %v", err)
		}
	}
}

// SendToKafka sends weather data to Kafka
func SendToKafka(data WeatherData, config *Config) {
	// Note: This is a placeholder for Kafka integration
	// In a real implementation, you would:
	// 1. Import the Kafka client library
	// 2. Establish a connection to the Kafka broker
	// 3. Serialize the weather data to JSON
	// 4. Send the data to the specified topic

	if config.Verbose {
		log.Printf("Would send data for %s to Kafka topic %s at broker %s",
			data.LocationID, config.KafkaTopic, config.KafkaBroker)
	}

	// For now, just indicate what would happen
	log.Printf("Kafka integration not implemented - data for %s would be sent to %s",
		data.LocationID, config.KafkaTopic)
}
