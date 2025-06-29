# Notes Sync

A Go-based file watcher that monitors markdown files and syncs them to various storage backends with frontmatter parsing.

This is a second iteration of the note syncing approach, followed after [`jot`](../jot/).

## Features

- **File watching**: Monitors markdown files for changes using `fsnotify`
- **Frontmatter parsing**: Extracts YAML frontmatter from markdown files
- **Multiple storage backends**: 
  - In-memory storage
  - SQLite database
  - MongoDB
- **Glob pattern exclusions**: Skip files/directories using glob patterns
- **Real-time sync**: Automatically syncs changes as they happen

## Usage

1. **Configure** `config.yml`:
```yaml
path: "/path/to/your/notes"
storage_type: sqlite  # memory, sqlite, or mongodb
connection_uri: notes.db
clear_storage: true
exclude_patterns:
  - "*/.git"
  - "*/.obsidian"
  - "*/templates/**"
```

2. **Run**:
```bash
go run main.go
```

## Storage Options

- **Memory**: Fast, ephemeral storage for testing
- **SQLite**: Lightweight, file-based database
- **MongoDB**: Scalable, document-based storage

## Dependencies

- `fsnotify` - File system notifications
- `gobwas/glob` - Glob pattern matching
- `mattn/go-sqlite3` - SQLite driver
- `mongo-driver` - MongoDB driver
- `yaml.v3` - YAML parsing

## Testing

```bash
go test
```

Runs comprehensive tests for parsing, storage backends, and event handling. 