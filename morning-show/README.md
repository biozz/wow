# Morning Show

A Go script that creates automated morning show content by reading unread entries from Miniflux, summarizing them using Gemini AI, and generating audio using text-to-speech.

## Features

- **Miniflux Integration**: Reads unread RSS feed entries from a Miniflux instance
- **AI Summarization**: Uses Google's Gemini AI to create engaging morning show summaries
- **Real Text-to-Speech**: Generates actual audio using Gemini TTS or Google Cloud TTS
- **Multiple TTS Options**: Support for both Gemini TTS (with Kore voice) and Google Cloud TTS
- **Configurable**: Environment-based configuration for easy deployment

## Prerequisites

- Go 1.19 or later
- A Miniflux instance running and accessible
- Google Gemini API key
- Google Cloud Project (for Gemini TTS)
- gcloud CLI installed and authenticated (for Gemini TTS)

## Installation

1. Clone or navigate to the morning-show directory
2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Configuration

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your configuration:
   ```bash
   # Miniflux API URL
   MINIFLUX_URL=http://your-miniflux-instance:8080/v1/entries?status=unread&direction=desc
   
   # Gemini API Key (required)
   GEMINI_API_KEY=your_actual_api_key_here
   
   # Google Cloud Project ID (required for Gemini TTS)
   PROJECT_ID=your_project_id_here
   
   # TTS Configuration
   TTS_SERVICE=gemini
   VOICE_NAME=Kore
   LANGUAGE_CODE=en-us
   TTS_PROMPT=Say the following in a curious and engaging way for a morning show
   
   # Output settings
   OUTPUT_FORMAT=wav
   OUTPUT_FILE=morning-show.wav
   ```

## Usage

### Basic Usage

```bash
# Set environment variables
export GEMINI_API_KEY="your_api_key_here"
export PROJECT_ID="your_project_id_here"
export MINIFLUX_URL="http://localhost:8080/v1/entries?status=unread&direction=desc"

# For Gemini TTS, make sure you're authenticated with gcloud
gcloud auth application-default login

# Run the morning show generator
go run main.go
```

### Using Environment File

```bash
# Load environment variables from .env file
source .env
go run main.go
```

### Build and Run

```bash
# Build the binary
go build -o morning-show main.go

# Run the binary
./morning-show
```

## How It Works

1. **Entry Collection**: The script fetches unread RSS feed entries from the configured Miniflux instance
2. **AI Summarization**: Entries are sent to Gemini AI with a prompt to create an engaging morning show summary
3. **Audio Generation**: The summary is processed using either:
   - **Gemini TTS**: Uses the latest Gemini TTS models with advanced voice synthesis
   - **Google Cloud TTS**: Uses the standard Google Cloud Text-to-Speech API
4. **Output**: The final audio file (WAV format) is saved to the configured location

## API Requirements

### Miniflux API

The script expects Miniflux to provide an endpoint that returns unread entries in the following format:

```json
[
  {
    "id": 123,
    "title": "Article Title",
    "url": "https://example.com/article",
    "content": "Article content...",
    "summary": "Article summary...",
    "published_at": "2024-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "status": "unread",
    "feed": {
      "id": 1,
      "title": "Feed Title",
      "site_url": "https://example.com",
      "feed_url": "https://example.com/feed.xml"
    }
  }
]
```

### Gemini API

- Requires a valid Gemini API key
- Uses the `gemini-1.5-flash` model for summarization
- Make sure your API key has access to the Generative AI API

### Google Cloud TTS

- **For Gemini TTS**: Requires a Google Cloud Project ID and gcloud CLI authentication
- **For Google Cloud TTS**: Uses the standard Text-to-Speech API
- Available models for Gemini TTS: `gemini-2.5-flash-preview-tts`, `gemini-2.5-pro-preview-tts`
- Available voices: Kore (and others depending on the model)

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `MINIFLUX_URL` | URL to Miniflux unread entries endpoint | `http://localhost:8080/v1/entries?status=unread&direction=desc` | No |
| `GEMINI_API_KEY` | Google Gemini API key | - | Yes |
| `PROJECT_ID` | Google Cloud Project ID (for Gemini TTS) | - | Yes (for Gemini TTS) |
| `TTS_SERVICE` | TTS service to use (`gemini` or `google`) | `gemini` | No |
| `VOICE_NAME` | Voice name for TTS | `Kore` | No |
| `LANGUAGE_CODE` | Language code for TTS | `en-us` | No |
| `TTS_PROMPT` | Prompt for TTS voice style | `Say the following in a curious and engaging way for a morning show` | No |
| `OUTPUT_FORMAT` | Output audio format | `wav` | No |
| `OUTPUT_FILE` | Output filename | `morning-show.wav` | No |

## Current Limitations

- **Aoede Voice**: The specific "Aoede" voice mentioned in requirements is not yet implemented (currently using Kore voice)
- **Audio Format**: Currently outputs WAV format only (MP3 support can be added)
- **gcloud Dependency**: Gemini TTS requires gcloud CLI to be installed and authenticated

## Future Enhancements

- Support for Aoede voice and other advanced voice options
- Audio format conversion (MP3 support)
- Scheduling and automation features
- Enhanced error handling and retry logic
- Configuration file support (JSON/YAML)
- Fallback TTS options if primary service fails

## Troubleshooting

### Common Issues

1. **"GEMINI_API_KEY environment variable is required"**
   - Make sure you've set the `GEMINI_API_KEY` environment variable
   - Verify your API key is valid and has the necessary permissions

2. **"Failed to make request to Miniflux"**
   - Check that your Miniflux instance is running and accessible
   - Verify the `MINIFLUX_URL` is correct
   - Ensure the Miniflux API endpoint returns the expected JSON format

3. **"Failed to generate content"**
   - Check your Gemini API key and quota
   - Verify you have access to the Generative AI API

4. **"PROJECT_ID environment variable is required for Gemini TTS"**
   - Set the `PROJECT_ID` environment variable with your Google Cloud project ID
   - Make sure you have access to the project

5. **"Failed to get access token from gcloud"**
   - Install gcloud CLI: https://cloud.google.com/sdk/docs/install
   - Run `gcloud auth application-default login`
   - Make sure you're authenticated with the correct project

## License

This project is part of the larger workspace and follows the same licensing terms.