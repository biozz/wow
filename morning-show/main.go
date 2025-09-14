package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"google.golang.org/genai"
	mflux "miniflux.app/v2/client"
)

type Config struct {
	MinifluxURL   string `env:"MINIFLUX_URL"`
	MinifluxToken string `env:"MINIFLUX_TOKEN"`
	GeminiAPIKey  string `env:"GEMINI_API_KEY"`
}

func main() {

	config := Config{}
	if err := env.Parse(&config); err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	mfluxClient := mflux.NewClient(config.MinifluxURL, config.MinifluxToken)

	// Step 1: Read unread entries from Miniflux
	entries, err := mfluxClient.Entries(&mflux.Filter{
		Status: mflux.EntryStatusUnread,
	})
	if err != nil {
		log.Fatalf("Failed to read Miniflux entries: %v", err)
	}
	if entries.Total == 0 {
		log.Println("No unread entries found. Exiting.")
		return
	}
	log.Printf("Found %d unread entries", entries.Total)

	// Step 2: Summarize entries using Gemini
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  config.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}

	// Read the prompt template from markdown file
	promptTemplate, err := os.ReadFile("summary-prompt.md")
	if err != nil {
		log.Fatalf("Failed to read summary-prompt.md: %v", err)
	}

	var prompt strings.Builder
	// Add current date and day as the first line
	currentTime := time.Now()
	dayOfWeek := currentTime.Format("Monday")
	date := currentTime.Format("January 2, 2006")
	prompt.WriteString(fmt.Sprintf("Today is %s, %s.\n\n", dayOfWeek, date))
	prompt.WriteString(fmt.Sprintf("Number of entries: %d.\n\n", entries.Total))
	prompt.WriteString(string(promptTemplate))

	for i, entry := range entries.Entries {
		content := entry.Content
		// Truncate the content to 200 characters
		if len(entry.Content) > 200 {
			content = entry.Content[:200] + "..."
		}
		prompt.WriteString(fmt.Sprintf("%d. [%s] %s - %s\n", i+1, entry.Feed.Title, entry.Title, content))
	}

	sumaryParts := []*genai.Part{
		{Text: prompt.String()},
	}

	result, err := genaiClient.Models.GenerateContent(context.Background(), "gemini-2.5-flash-lite", []*genai.Content{{Parts: sumaryParts}}, nil)
	if err != nil {
		log.Fatalf("Failed to generate content: %v", err)
	}

	log.Println(result.Text())

	log.Println("Entries summarized successfully")

	showParts := []*genai.Part{
		{Text: result.Text()},
	}
	showResult, err := genaiClient.Models.GenerateContent(
		context.Background(),
		"gemini-2.5-flash-preview-tts",
		[]*genai.Content{{Parts: showParts}}, // Content to be spoken
		&genai.GenerateContentConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: &genai.SpeechConfig{
				VoiceConfig: &genai.VoiceConfig{
					PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
						VoiceName: "Aoede",
					},
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to generate content: %v", err)
	}

	// Save as WAV file with timestamp in the name
	timestamp := time.Now().Format("20060102150405")
	fileName := fmt.Sprintf("morning-show-%s.wav", timestamp)
	err = writeWAVFile(fileName, showResult.Candidates[0].Content.Parts[0].InlineData.Data, 24000, 1, 16)
	if err != nil {
		log.Fatalf("Failed to write WAV file: %v", err)
	}

	log.Printf("Morning show audio generated successfully: %s", fileName)
}

// writeWAVFile writes PCM audio data to a WAV file with the specified format
// sampleRate: samples per second (e.g., 24000)
// channels: number of audio channels (1 for mono, 2 for stereo)
// bitsPerSample: bits per sample (16 for 16-bit audio)
func writeWAVFile(filename string, audioData []byte, sampleRate, channels, bitsPerSample int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Calculate derived values
	bytesPerSample := bitsPerSample / 8
	blockAlign := channels * bytesPerSample
	byteRate := sampleRate * blockAlign
	dataSize := len(audioData)
	fileSize := 36 + dataSize

	// Write WAV header
	// RIFF header
	file.WriteString("RIFF")
	binary.Write(file, binary.LittleEndian, uint32(fileSize))
	file.WriteString("WAVE")

	// fmt chunk
	file.WriteString("fmt ")
	binary.Write(file, binary.LittleEndian, uint32(16)) // fmt chunk size
	binary.Write(file, binary.LittleEndian, uint16(1))  // audio format (1 = PCM)
	binary.Write(file, binary.LittleEndian, uint16(channels))
	binary.Write(file, binary.LittleEndian, uint32(sampleRate))
	binary.Write(file, binary.LittleEndian, uint32(byteRate))
	binary.Write(file, binary.LittleEndian, uint16(blockAlign))
	binary.Write(file, binary.LittleEndian, uint16(bitsPerSample))

	// data chunk
	file.WriteString("data")
	binary.Write(file, binary.LittleEndian, uint32(dataSize))
	file.Write(audioData)

	return nil
}
