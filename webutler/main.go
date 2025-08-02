package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	mcpclient "github.com/mark3labs/mcp-go/client"
	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	tele "gopkg.in/telebot.v4"
)

type config struct {
	TelegramBotToken          string `env:"TELEGRAM_BOT_TOKEN"`
	TelegramApiId             string `env:"TELEGRAM_API_ID"`
	TelegramApiHash           string `env:"TELEGRAM_API_HASH"`
	OpenAIAPIKey              string `env:"OPENAI_API_KEY"`
	OpenAIAPIURL              string `env:"OPENAI_API_URL"`
	OpenAIModel               string `env:"OPENAI_MODEL"`
	GithubPersonalAccessToken string `env:"GITHUB_PERSONAL_ACCESS_TOKEN"`
	GithubMCPCommand          string `env:"GITHUB_MCP_COMMAND" default:"docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN ghcr.io/github/github-mcp-server"`
}

// Conversation stores messages for the single user
type Conversation struct {
	Messages []openai.ChatCompletionMessage
}

// Global conversation for single user
var conversation = &Conversation{
	Messages: []openai.ChatCompletionMessage{},
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("Error parsing environment variables: %+v\n", err)
		os.Exit(1)
	}

	// Setup MCP client for GitHub
	githubMCPCommand := strings.Split(cfg.GithubMCPCommand, " ")
	// For Docker, we don't need to pass the token in the env slice since it's already in the command
	var envVars []string
	if !strings.Contains(cfg.GithubMCPCommand, "docker") {
		envVars = []string{"GITHUB_PERSONAL_ACCESS_TOKEN=" + cfg.GithubPersonalAccessToken}
	}
	stdio := mcptransport.NewStdio(githubMCPCommand[0], envVars, githubMCPCommand[1:]...)
	mcpClient := mcpclient.NewClient(stdio)
	if err := mcpClient.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}
	defer mcpClient.Close()
	initResult, err := mcpClient.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION},
	})
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	log.Printf("Connected to server: %s v%s", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	log.Printf("Server capabilities: %+v", initResult.Capabilities)

	// Define limited set of tools for OpenAI
	openaiTools := []openai.Tool{
		{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name: "create_issue",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"assignees": {"type": "array", "items": {"type": "string"}},
						"body":      {"type": "string"},
						"labels":    {"type": "array", "items": {"type": "string"}},
						"milestone": {"type": "number"},
						"owner":     {"type": "string"},
						"repo":      {"type": "string"},
						"title":     {"type": "string"}
					},
					"required": ["owner", "repo", "title"]
				}`),
			},
		},
		{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name: "list_tags",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"owner":   {"type": "string"},
						"repo":    {"type": "string"},
						"page":    {"type": "number"},
						"perPage": {"type": "number"}
					},
					"required": ["owner", "repo", "page", "perPage"]
				}`),
			},
		},
	}

	// Setup OpenAI client
	openaiConfig := openai.DefaultConfig(cfg.OpenAIAPIKey)
	if cfg.OpenAIAPIURL != "" {
		openaiConfig.BaseURL = cfg.OpenAIAPIURL
	}
	openaiClient := openai.NewClientWithConfig(openaiConfig)

	bot, err := tele.NewBot(tele.Settings{
		Token:  cfg.TelegramBotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Handle /new command
	bot.Handle("/new", func(c tele.Context) error {
		conversation.Messages = []openai.ChatCompletionMessage{}
		return c.Send("New conversation started")
	})

	// Handle text messages (non-command messages)
	bot.Handle(tele.OnText, func(c tele.Context) error {
		messageText := c.Text()

		// Initialize conversation if it doesn't exist
		if conversation == nil {
			conversation = &Conversation{
				Messages: []openai.ChatCompletionMessage{},
			}
		}

		// Add user message to conversation
		conversation.Messages = append(conversation.Messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: messageText,
		})

		// Process with OpenAI
		response, err := openaiClient.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model:    cfg.OpenAIModel,
			Messages: conversation.Messages,
			Tools:    openaiTools,
		})
		if err != nil {
			return err
		}

		// Add assistant response to conversation
		conversation.Messages = append(conversation.Messages, response.Choices[0].Message)

		// Handle tool calls if present
		if response.Choices[0].FinishReason == openai.FinishReasonToolCalls {
			for _, toolCall := range response.Choices[0].Message.ToolCalls {
				argsMap := make(map[string]any)
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &argsMap); err != nil {
					return err
				}
				log.Printf("Tool call arguments: %+v", argsMap)
				toolCallResult, err := mcpClient.CallTool(context.Background(), mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      toolCall.Function.Name,
						Arguments: argsMap,
					},
				})
				if err != nil {
					return err
				}
				toolResultContent := toolCallResult.Content[0].(mcp.TextContent)
				conversation.Messages = append(conversation.Messages, openai.ChatCompletionMessage{
					Role:       "tool",
					Content:    toolResultContent.Text,
					ToolCallID: toolCall.ID,
				})
			}

			// Make the second API call with the complete conversation including tool calls and responses
			response, err = openaiClient.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
				Model:    cfg.OpenAIModel,
				Messages: conversation.Messages,
				Tools:    openaiTools,
			})
			if err != nil {
				return err
			}

			// Add final assistant response to conversation
			conversation.Messages = append(conversation.Messages, response.Choices[0].Message)
		}

		return c.Send(response.Choices[0].Message.Content)
	})

	bot.Start()
}
