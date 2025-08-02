# WebButler

A Telegram bot that creates GitHub issues using AI and MCP (Model Context Protocol) tools. Processes user requests through OpenAI models and automatically generates GitHub issues with the requested content.

## Features

- Telegram bot interface for natural language requests
- AI-powered issue creation using OpenAI models
- GitHub integration via MCP tools
- Conversation memory for context-aware interactions
- Support for issue creation with assignees, labels, and milestones

## Setup

1. Copy `env.example` to `.env` and configure:
   ```
   TELEGRAM_BOT_TOKEN=your_telegram_bot_token
   TELEGRAM_API_ID=your_telegram_api_id
   TELEGRAM_API_HASH=your_telegram_api_hash
   OPENAI_API_KEY=your_openai_api_key
   OPENAI_API_URL=your_openai_api_url
   OPENAI_MODEL=gpt-4
   GITHUB_PERSONAL_ACCESS_TOKEN=your_github_token
   GITHUB_MCP_COMMAND=docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN ghcr.io/github/github-mcp-server
   ```

2. Run the bot:
   ```bash
   go run main.go
   ```

## Usage

- Send any message to create a GitHub issue
- Use `/new` to start a fresh conversation
- The bot will process your request and create the appropriate GitHub issue 