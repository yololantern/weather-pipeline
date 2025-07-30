package main

import (
	"log"
	"time"
)

func main() {
	// Parse command line flags
	config := ParseFlags()

	// Validate config
	if err := ValidateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Process locations (either once or on interval)
	if config.Interval > 0 {
		// Run continuously with interval
		ticker := time.NewTicker(config.Interval)
		defer ticker.Stop()

		log.Printf("Starting weather data pipeline. Fetching data every %v", config.Interval)

		// Run once immediately
		ProcessLocations(config)

		// Then on ticker interval
		for range ticker.C {
			ProcessLocations(config)
		}
	} else {
		// Run once
		ProcessLocations(config)
	}
}
