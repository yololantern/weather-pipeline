# Weather Data Pipeline

A real-time weather data collection and processing pipeline built in Go, designed to ingest data from the OpenWeatherMap API and process it for analytics platforms.

## Features

- Collect real-time weather data from OpenWeatherMap API
- Fallback to National Weather Service API when no OpenWeatherMap API key is provided
- Process multiple locations in batch
- Schedule automatic data collection at configurable intervals
- Output in multiple formats (text, JSON, CSV, Kafka)
- AI-powered weather summary generation using OpenAI
- Flexible configuration through command-line flags
- Optional web-based GUI with Docker support

## Installation

### Prerequisites

- Go 1.21 or later
- OpenWeatherMap API key (optional, will use National Weather Service API as fallback)
- OpenAI API key (optional, for AI summaries)
- Docker and Docker Compose (optional, for running the GUI)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yololantern/weather-pipeline.git
cd weather-pipeline

# Build the application
go build -o weathercli

# Run the application
./weathercli <ZIP_CODE>
```

## Usage Examples

### Basic Usage

```bash
# Get weather for a single location
./weathercli 90210

# Use metric units (Celsius)
./weathercli -metric 90210

# Get weather for multiple locations
./weathercli -zip-codes=90210,10001,60601
```

### Data Collection & Output

```bash
# Output data in JSON format
./weathercli -format=json -zip-codes=90210,10001

# Save output to a file
./weathercli -format=json -output=weather.json -zip-codes=90210,10001

# Output as CSV for data analysis
./weathercli -format=csv -output=weather.csv -zip-codes=90210,10001,60601,02108
```

### Scheduled Collection

```bash
# Collect data every hour
./weathercli -interval=3600 -zip-codes=90210,10001

# Continuous monitoring with verbose logging
./weathercli -interval=1800 -verbose -zip-codes=90210,10001,60601
```

### Data Pipeline Integration

```bash
# Stream data to Kafka
./weathercli -format=kafka -kafka-broker=localhost:9092 -kafka-topic=weather-data -zip-codes=90210,10001

# Export data for NiFi ingestion
./weathercli -format=json -output=/data/nifi/input/weather.json -zip-codes=90210,10001,60601
```

## Web-based GUI

The application includes an optional web-based GUI that can be run using Docker.

### Running the GUI with Docker

```bash
# Build and start the GUI
docker-compose up -d

# Access the GUI in your browser
# http://localhost:8080
```

### GUI Features

- User-friendly interface for weather data
- Support for multiple ZIP codes
- Optional API key input
- Toggle between metric and imperial units
- Responsive design for desktop and mobile
- Displays current conditions and 7-day forecast
- Shows AI-generated summary when available

### Docker Environment Variables

You can configure the Docker container by editing the `docker-compose.yml` file:

```yaml
environment:
  - PORT=8080
  # Uncomment and set your OpenWeatherMap API key if you have one
  # - OWM_API_KEY=your_api_key_here
  # Uncomment and set your OpenAI API key if you want AI-generated summaries
  # - OPENAI_API_KEY=your_openai_api_key_here
```

## Environment Variables

The application recognizes the following environment variables:

- `OWM_API_KEY`: Your OpenWeatherMap API key (optional)
- `OPENAI_API_KEY`: Your OpenAI API key (for AI-generated summaries)

Example:
```bash
export OWM_API_KEY=your_openweathermap_api_key
export OPENAI_API_KEY=your_openai_api_key
./weathercli 90210
```

## Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `-api-key` | OpenWeatherMap API key | From `OWM_API_KEY` env var |
| `-zip-codes` | Comma-separated list of ZIP codes | - |
| `-format` | Output format: text, json, csv, kafka | text |
| `-output` | Output file path | stdout |
| `-metric` | Use metric units (Celsius) | false |
| `-kafka-broker` | Kafka broker address | localhost:9092 |
| `-kafka-topic` | Kafka topic for output | weather-data |
| `-interval` | Polling interval in seconds | 0 (run once) |
| `-verbose` | Enable verbose logging | false |

## Data Pipeline Architecture

This application forms part of a larger data pipeline:

1. **Data Collection**: Collect weather data from OpenWeatherMap API or National Weather Service API
2. **Preprocessing**: Clean and transform the data into a standard format
3. **Stream Processing**: Stream data through Kafka for real-time analysis
4. **Batch Processing**: Generate CSV/JSON files for batch analysis
5. **Data Warehouse**: Store processed data in a warehouse for long-term analytics
6. **Visualization**: Generate insights on weather patterns and trends

## Output Formats

### Text Format
Human-readable output with current conditions and forecast.

### JSON Format
Structured data suitable for API responses or file storage.

### CSV Format
Tabular data format ideal for spreadsheet analysis or data warehouse loading.

### Kafka Format
Streams data to a Kafka topic for real-time processing.

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.