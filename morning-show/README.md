# Morning Show

A Go script that creates automated morning show content by reading unread messages from miniflu, summarizing them using Gemini AI, and generating audio using text-to-speech.

## Features

- **Miniflu Integration**: Reads unread messages from a miniflu instance
- **AI Summarization**: Uses Google's Gemini AI to create engaging morning show summaries
- **Text-to-Speech**: Generates audio output (currently placeholder implementation)
- **Configurable**: Environment-based configuration for easy deployment

## Prerequisites

- Go 1.19 or later
- A miniflu instance running and accessible
- Google Gemini API key

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
   # Miniflu API URL
   MINIFLU_URL=http://your-miniflu-instance:8080/api/messages/unread
   
   # Gemini API Key (required)
   GEMINI_API_KEY=your_actual_api_key_here
   
   # Output settings
   OUTPUT_FORMAT=wav
   OUTPUT_FILE=morning-show.wav
   ```

## Usage

### Basic Usage

```bash
# Set environment variables
export GEMINI_API_KEY="your_api_key_here"
export MINIFLU_URL="http://localhost:8080/api/messages/unread"

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

1. **Message Collection**: The script fetches unread messages from the configured miniflu instance
2. **AI Summarization**: Messages are sent to Gemini AI with a prompt to create an engaging morning show summary
3. **Audio Generation**: The summary is processed for text-to-speech (currently outputs a text file as placeholder)
4. **Output**: The final audio file is saved to the configured location

## API Requirements

### Miniflu API

The script expects miniflu to provide an endpoint that returns unread messages in the following format:

```json
{
  "messages": [
    {
      "id": "message_id",
      "content": "Message content",
      "timestamp": "2024-01-01T00:00:00Z",
      "read": false,
      "source": "source_name"
    }
  ],
  "total": 1
}
```

### Gemini API

- Requires a valid Gemini API key
- Uses the `gemini-1.5-flash` model for summarization
- Make sure your API key has access to the Generative AI API

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `MINIFLU_URL` | URL to miniflu unread messages endpoint | `http://localhost:8080/api/messages/unread` | No |
| `GEMINI_API_KEY` | Google Gemini API key | - | Yes |
| `OUTPUT_FORMAT` | Output audio format | `wav` | No |
| `OUTPUT_FILE` | Output filename | `morning-show.wav` | No |

## Current Limitations

- **TTS Implementation**: The current implementation outputs a text file instead of actual audio. To implement real TTS, you would need to:
  - Use Google Cloud Text-to-Speech API
  - Integrate with a different TTS service
  - Implement a custom TTS solution
- **Aoede Voice**: The specific "Aoede" voice mentioned in requirements is not yet implemented

## Future Enhancements

- Real TTS integration with Google Cloud Text-to-Speech
- Support for multiple voice options including Aoede
- Audio format conversion (WAV/MP3)
- Scheduling and automation features
- Enhanced error handling and retry logic
- Configuration file support (JSON/YAML)

## Troubleshooting

### Common Issues

1. **"GEMINI_API_KEY environment variable is required"**
   - Make sure you've set the `GEMINI_API_KEY` environment variable
   - Verify your API key is valid and has the necessary permissions

2. **"Failed to make request to miniflu"**
   - Check that your miniflu instance is running and accessible
   - Verify the `MINIFLU_URL` is correct
   - Ensure the miniflu API endpoint returns the expected JSON format

3. **"Failed to generate content"**
   - Check your Gemini API key and quota
   - Verify you have access to the Generative AI API

## License

This project is part of the larger workspace and follows the same licensing terms.