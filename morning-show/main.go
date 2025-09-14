package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"google.golang.org/api/texttospeech/v1"
)

// Config holds configuration for the morning show
type Config struct {
	MinifluxURL     string
	GeminiAPIKey    string
	ProjectID       string
	OutputFormat    string // "wav" or "mp3"
	OutputFile      string
	TTSService      string // "gemini" or "google"
	VoiceName       string
	LanguageCode    string
	TTSPrompt       string
}

// MinifluxEntry represents an entry from Miniflux
type MinifluxEntry struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Content     string    `json:"content"`
	Summary     string    `json:"summary"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
	Feed        struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		SiteURL  string `json:"site_url"`
		FeedURL  string `json:"feed_url"`
	} `json:"feed"`
}

// MinifluxResponse represents the response from Miniflux API
type MinifluxResponse []MinifluxEntry

// GeminiTTSRequest represents the request to Gemini TTS API
type GeminiTTSRequest struct {
	Input struct {
		Prompt string `json:"prompt"`
		Text   string `json:"text"`
	} `json:"input"`
	Voice struct {
		LanguageCode string `json:"languageCode"`
		Name         string `json:"name"`
		ModelName    string `json:"model_name"`
	} `json:"voice"`
	AudioConfig struct {
		AudioEncoding string `json:"audioEncoding"`
	} `json:"audioConfig"`
}

// GeminiTTSResponse represents the response from Gemini TTS API
type GeminiTTSResponse struct {
	AudioContent string `json:"audioContent"`
}

func main() {
	// Load configuration from environment variables
	config := &Config{
		MinifluxURL:   getEnv("MINIFLUX_URL", "http://localhost:8080/v1/entries?status=unread&direction=desc"),
		GeminiAPIKey:  getEnv("GEMINI_API_KEY", ""),
		ProjectID:     getEnv("PROJECT_ID", ""),
		OutputFormat:  getEnv("OUTPUT_FORMAT", "wav"),
		OutputFile:    getEnv("OUTPUT_FILE", "morning-show.wav"),
		TTSService:    getEnv("TTS_SERVICE", "gemini"),
		VoiceName:     getEnv("VOICE_NAME", "Kore"),
		LanguageCode:  getEnv("LANGUAGE_CODE", "en-us"),
		TTSPrompt:     getEnv("TTS_PROMPT", "Say the following in a curious and engaging way for a morning show"),
	}

	if config.GeminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	if config.TTSService == "gemini" && config.ProjectID == "" {
		log.Fatal("PROJECT_ID environment variable is required for Gemini TTS")
	}

	log.Println("Starting morning show generation...")

	// Step 1: Read unread entries from Miniflux
	entries, err := readMinifluxEntries(config.MinifluxURL)
	if err != nil {
		log.Fatalf("Failed to read Miniflux entries: %v", err)
	}

	if len(entries) == 0 {
		log.Println("No unread entries found. Exiting.")
		return
	}

	log.Printf("Found %d unread entries", len(entries))

	// Step 2: Summarize entries using Gemini
	summary, err := summarizeEntries(entries, config.GeminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to summarize entries: %v", err)
	}

	log.Println("Entries summarized successfully")

	// Step 3: Generate audio using Gemini TTS
	err = generateAudio(summary, config)
	if err != nil {
		log.Fatalf("Failed to generate audio: %v", err)
	}

	log.Printf("Morning show audio generated successfully: %s", config.OutputFile)
}

// readMinifluxEntries fetches unread entries from Miniflux
func readMinifluxEntries(url string) ([]MinifluxEntry, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Miniflux: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Miniflux API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response MinifluxResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

// summarizeEntries uses Gemini to summarize the entries
func summarizeEntries(entries []MinifluxEntry, apiKey string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")

	// Prepare the prompt with all entries
	var prompt strings.Builder
	prompt.WriteString("You are creating a summary for a morning show. Please summarize the following unread RSS feed entries in a conversational, engaging way that would be suitable for a morning show format. Keep it concise but informative, and make it sound natural when read aloud.\n\n")
	prompt.WriteString("Feed Entries:\n")

	for i, entry := range entries {
		// Use summary if available, otherwise use content (truncated)
		content := entry.Summary
		if content == "" {
			content = entry.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
		}
		
		prompt.WriteString(fmt.Sprintf("%d. [%s] %s - %s\n", i+1, entry.Feed.Title, entry.Title, content))
	}

	prompt.WriteString("\nPlease provide a summary that flows well as a morning show segment.")

	resp, err := model.GenerateContent(ctx, genai.Text(prompt.String()))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	summary := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			summary += string(text)
		}
	}

	return summary, nil
}

// generateAudio uses TTS to generate audio from the summary
func generateAudio(text string, config *Config) error {
	log.Printf("Generating audio for text: %s", text)
	
	switch config.TTSService {
	case "gemini":
		return generateGeminiTTS(text, config)
	case "google":
		return generateGoogleTTS(text, config)
	default:
		return fmt.Errorf("unsupported TTS service: %s", config.TTSService)
	}
}

// generateGeminiTTS uses Gemini TTS API to generate audio
func generateGeminiTTS(text string, config *Config) error {
	// Get access token using gcloud
	accessToken, err := getGCloudAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Prepare the request
	request := GeminiTTSRequest{
		Input: struct {
			Prompt string `json:"prompt"`
			Text   string `json:"text"`
		}{
			Prompt: config.TTSPrompt,
			Text:   text,
		},
		Voice: struct {
			LanguageCode string `json:"languageCode"`
			Name         string `json:"name"`
			ModelName    string `json:"model_name"`
		}{
			LanguageCode: config.LanguageCode,
			Name:         config.VoiceName,
			ModelName:    "gemini-2.5-flash-preview-tts",
		},
		AudioConfig: struct {
			AudioEncoding string `json:"audioEncoding"`
		}{
			AudioEncoding: "LINEAR16",
		},
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request to Gemini TTS API
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", "https://texttospeech.googleapis.com/v1/text:synthesize", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-goog-user-project", config.ProjectID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TTS API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ttsResponse GeminiTTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ttsResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Decode base64 audio content
	audioData, err := base64.StdEncoding.DecodeString(ttsResponse.AudioContent)
	if err != nil {
		return fmt.Errorf("failed to decode audio content: %w", err)
	}

	// Write audio file
	file, err := os.Create(config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(audioData)
	if err != nil {
		return fmt.Errorf("failed to write audio data: %w", err)
	}

	log.Printf("Audio generated successfully: %s", config.OutputFile)
	return nil
}

// generateGoogleTTS uses Google Cloud Text-to-Speech API
func generateGoogleTTS(text string, config *Config) error {
	ctx := context.Background()
	
	// Create TTS client
	client, err := texttospeech.NewService(ctx, option.WithAPIKey(config.GeminiAPIKey))
	if err != nil {
		return fmt.Errorf("failed to create TTS client: %w", err)
	}

	// Prepare the synthesis request
	req := &texttospeech.SynthesizeSpeechRequest{
		Input: &texttospeech.SynthesisInput{
			Text: text,
		},
		Voice: &texttospeech.VoiceSelectionParams{
			LanguageCode: config.LanguageCode,
			Name:         config.VoiceName,
		},
		AudioConfig: &texttospeech.AudioConfig{
			AudioEncoding: "LINEAR16",
		},
	}

	// Perform the synthesis
	resp, err := client.Text.Synthesize(req).Do()
	if err != nil {
		return fmt.Errorf("failed to synthesize speech: %w", err)
	}

	// Write audio file
	file, err := os.Create(config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Decode base64 audio content for Google TTS
	audioData, err := base64.StdEncoding.DecodeString(resp.AudioContent)
	if err != nil {
		return fmt.Errorf("failed to decode audio content: %w", err)
	}

	_, err = file.Write(audioData)
	if err != nil {
		return fmt.Errorf("failed to write audio data: %w", err)
	}

	log.Printf("Audio generated successfully: %s", config.OutputFile)
	return nil
}

// getGCloudAccessToken gets an access token using gcloud CLI
func getGCloudAccessToken() (string, error) {
	// This is a simplified implementation
	// In a production environment, you might want to use the Google Cloud Go client libraries
	// or implement proper OAuth2 flow
	
	// For now, we'll assume the user has run `gcloud auth application-default login`
	// and we can use the default credentials
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get access token from gcloud: %w", err)
	}
	
	return strings.TrimSpace(string(output)), nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}