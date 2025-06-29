# Jot

A Telegram bot that automatically saves messages to markdown files with YAML frontmatter for seamless integration with Obsidian via Syncthing.

## Features

- **Telegram bot**: Receives messages and saves them as markdown files
- **YAML frontmatter**: Structured metadata including source, sender, timestamps, and tags
- **Template-based**: Uses customizable markdown templates
- **Docker-ready**: Containerized for easy deployment
- **Admin-only**: Whitelist-based access control
- **Message editing**: Handles edited messages and updates files accordingly
- **Auto-cleanup**: Deletes messages from Telegram after saving

## Usage

### Environment Variables

```bash
TELEGRAM_BOT_TOKEN=your_bot_token_here
BOT_ADMIN_ID=your_telegram_user_id
INBOX_PATH=/path/to/save/messages
FILENAME_TEMPLATE=inbox_20060102_150405.md
```

### Docker

```bash
docker-compose up -d
```

### Local

```bash
go run main.go
```

## Message Format

Messages are saved as markdown files with YAML frontmatter:

```yaml
---
summary: |
  Your message content here
source: telegram
sender: your_username
aliases:
tags:
  - task
  - telegram
completed:
created: 2024-01-01T12:00:00Z
modified: 2024-01-01T12:00:00Z
---
```

## Workflow

1. Send message to Telegram bot
2. Bot saves message to `INBOX_PATH` with timestamped filename
3. Syncthing syncs files to your Obsidian vault
4. Message is automatically deleted from Telegram
5. Edit messages in Telegram to update the saved file

## Dependencies

- `telebot.v4` - Telegram bot framework
- Go 1.24.1+

## Integration

Designed to work with:
- **Obsidian**: For note management and viewing
- **Syncthing**: For file synchronization between devices


