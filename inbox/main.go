package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

// Message represents a queue message
type Message struct {
	ID         int64     `bun:",pk,autoincrement" json:"id"`
	Text       string    `bun:",notnull" json:"text"`
	State      string    `bun:",notnull" json:"-"`
	CreatedAt  time.Time `bun:"created_at,nullzero,notnull,default:CURRENT_TIMESTAMP" json:"timestamp"`
	ArchivedAt time.Time `bun:"archived_at,nullzero" json:"-"`
}

// PostMessageRequest represents the request body for POST /v1/messages
type PostMessageRequest struct {
	Text string `json:"text"`
}

// AddMessageRequest represents the request body for POST /v1/messages/add
type AddMessageRequest struct {
	Text string `json:"text"`
}

// Config holds application configuration
type Config struct {
	ListenAddr string
	DBPath     string
	AuthToken  string
}

// Server holds the application state
type Server struct {
	db     *bun.DB
	config Config
}

// NewServer creates a new server instance
func NewServer(config Config) (*Server, error) {
	log.Printf("Initializing server...")

	// Open SQLite database
	sqldb, err := sql.Open(sqliteshim.ShimName, config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	log.Printf("Database opened: %s", config.DBPath)

	// Create Bun DB
	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Printf("Database migrations completed")

	return &Server{
		db:     db,
		config: config,
	}, nil
}

// runMigrations executes the SQL migration files
func runMigrations(db *bun.DB) error {
	migrationSQL := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS messages (
	  id           INTEGER PRIMARY KEY AUTOINCREMENT,
	  text         TEXT NOT NULL,
	  state        TEXT NOT NULL CHECK (state IN ('new','archived')) DEFAULT 'new',
	  created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	  archived_at  DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_messages_state_created ON messages(state, created_at, id);
	`

	_, err := db.Exec(migrationSQL)
	return err
}

// authMiddleware validates the token URL parameter
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			log.Printf("Token parameter required")
			http.Error(w, "Token parameter required", http.StatusUnauthorized)
			return
		}

		if token != s.config.AuthToken {
			log.Printf("Invalid token: %s", token)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// handlePostMessage handles POST /v1/messages
func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PostMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	message := &Message{
		Text:  req.Text,
		State: "new",
	}

	_, err := s.db.NewInsert().Model(message).Exec(r.Context())
	if err != nil {
		log.Printf("Failed to insert message: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}

// handleGetMessages handles GET /v1/messages
func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 1
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	messages, err := s.fetchAndArchive(r.Context(), limit)
	if err != nil {
		log.Printf("Failed to fetch and archive messages: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// fetchAndArchive atomically fetches and archives messages
func (s *Server) fetchAndArchive(ctx context.Context, limit int) ([]Message, error) {
	var messages []Message

	err := s.db.RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewRaw(`
			WITH picked AS (
			  SELECT id FROM messages
			  WHERE state = 'new'
			  ORDER BY created_at ASC, id ASC
			  LIMIT ?
			)
			UPDATE messages
			SET state = 'archived', archived_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
			WHERE id IN (SELECT id FROM picked)
			RETURNING id, created_at, text
		`, limit).Scan(ctx, &messages)
	})

	return messages, err
}

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple health check - just verify DB is reachable
	if err := s.db.Ping(); err != nil {
		log.Printf("Health check failed: %v", err)
		http.Error(w, "Database unreachable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleAddMessage handles GET/POST /v1/messages/add
func (s *Server) handleAddMessage(w http.ResponseWriter, r *http.Request) {
	var text string

	switch r.Method {
	case http.MethodGet:
		text = r.URL.Query().Get("text")
		if text == "" {
			http.Error(w, "Text parameter is required", http.StatusBadRequest)
			return
		}
	case http.MethodPost:
		var req AddMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		text = req.Text
		if text == "" {
			http.Error(w, "Text is required", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	message := &Message{
		Text:  text,
		State: "new",
	}

	_, err := s.db.NewInsert().Model(message).Exec(r.Context())
	if err != nil {
		log.Printf("Failed to insert message: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}

// handleMessages routes requests to the appropriate handler
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handlePostMessage(w, r)
	case http.MethodGet:
		s.handleGetMessages(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/messages", s.loggingMiddleware(s.authMiddleware(s.handleMessages)))
	mux.HandleFunc("/v1/messages/add", s.loggingMiddleware(s.authMiddleware(s.handleAddMessage)))
	mux.HandleFunc("/health", s.loggingMiddleware(s.handleHealth))

	return mux
}

// loggingMiddleware logs each request
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Request: %s %s", r.Method, r.URL.Path)

		next(w, r)

		log.Printf("Response: %s %s - %v", r.Method, r.URL.Path, time.Since(start))
	}
}

// getConfig loads configuration from environment variables
func getConfig() Config {
	config := Config{
		ListenAddr: ":8080",
		DBPath:     "./inbox.db",
	}

	if addr := os.Getenv("LISTEN_ADDR"); addr != "" {
		config.ListenAddr = addr
	}
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		config.DBPath = dbPath
	}
	if token := os.Getenv("AUTH_TOKEN"); token != "" {
		config.AuthToken = token
	}

	return config
}

func main() {
	config := getConfig()

	if config.AuthToken == "" {
		log.Fatal("AUTH_TOKEN environment variable is required")
	}

	log.Printf("Starting inbox server with config: listen=%s, db=%s",
		config.ListenAddr, config.DBPath)

	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.db.Close()

	mux := server.setupRoutes()

	log.Printf("Server ready, listening on %s", config.ListenAddr)
	log.Fatal(http.ListenAndServe(config.ListenAddr, mux))
}
