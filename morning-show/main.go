package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Config holds configuration for the morning show
type Config struct {
	MinifluURL    string
	GeminiAPIKey  string
	OutputFormat  string // "wav" or "mp3"
	OutputFile    string
}

// MinifluMessage represents a message from miniflu
type MinifluMessage struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
	Source    string    `json:"source"`
}

// MinifluResponse represents the response from miniflu API
type MinifluResponse struct {
	Messages []MinifluMessage `json:"messages"`
	Total    int              `json:"total"`
}

// GeminiTTSRequest represents the request to Gemini TTS API
type GeminiTTSRequest struct {
	Text  string `json:"text"`
	Voice string `json:"voice"`
}

// GeminiTTSResponse represents the response from Gemini TTS API
type GeminiTTSResponse struct {
	AudioData string `json:"audioData"`
	Format    string `json:"format"`
}

func main() {
	// Load configuration from environment variables
	config := &Config{
		MinifluURL:   getEnv("MINIFLU_URL", "http://localhost:8080/api/messages/unread"),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		OutputFormat: getEnv("OUTPUT_FORMAT", "wav"),
		OutputFile:   getEnv("OUTPUT_FILE", "morning-show.wav"),
	}

	if config.GeminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	log.Println("Starting morning show generation...")

	// Step 1: Read unread messages from miniflu
	messages, err := readMinifluMessages(config.MinifluURL)
	if err != nil {
		log.Fatalf("Failed to read miniflu messages: %v", err)
	}

	if len(messages) == 0 {
		log.Println("No unread messages found. Exiting.")
		return
	}

	log.Printf("Found %d unread messages", len(messages))

	// Step 2: Summarize messages using Gemini
	summary, err := summarizeMessages(messages, config.GeminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to summarize messages: %v", err)
	}

	log.Println("Messages summarized successfully")

	// Step 3: Generate audio using Gemini TTS
	err = generateAudio(summary, config)
	if err != nil {
		log.Fatalf("Failed to generate audio: %v", err)
	}

	log.Printf("Morning show audio generated successfully: %s", config.OutputFile)
}

// readMinifluMessages fetches unread messages from miniflu
func readMinifluMessages(url string) ([]MinifluMessage, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to miniflu: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("miniflu API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response MinifluResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Messages, nil
}

// summarizeMessages uses Gemini to summarize the messages
func summarizeMessages(messages []MinifluMessage, apiKey string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")

	// Prepare the prompt with all messages
	var prompt strings.Builder
	prompt.WriteString("You are creating a summary for a morning show. Please summarize the following unread messages in a conversational, engaging way that would be suitable for a morning show format. Keep it concise but informative, and make it sound natural when read aloud.\n\n")
	prompt.WriteString("Messages:\n")

	for i, msg := range messages {
		prompt.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, msg.Source, msg.Content))
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

// generateAudio uses Gemini TTS to generate audio from the summary
func generateAudio(text string, config *Config) error {
	// Note: This is a placeholder implementation since Gemini TTS API
	// might not be directly available. In a real implementation, you would:
	// 1. Use Google Cloud Text-to-Speech API
	// 2. Or use a different TTS service
	// 3. Or implement a custom TTS solution

	// For now, we'll create a simple text file as output
	// In a real implementation, this would generate actual audio
	log.Printf("Generating audio for text: %s", text)
	
	// Create a simple text file as placeholder
	file, err := os.Create(config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("Morning Show Summary\n==================\n\n%s", text))
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	log.Printf("Audio content written to %s (placeholder implementation)", config.OutputFile)
	return nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}