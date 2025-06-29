package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gobwas/glob"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v3"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	config, err := loadConfig("config.yml")
	if err != nil {
		log.Fatal(err)
	}
	storage, err := NewStorage(config.StorageType, config.Conn)
	if err != nil {
		log.Fatal(err)
	}
	defer storage.Close()
	if config.ClearStorage {
		if err := storage.Clear(); err != nil {
			log.Printf("Warning: Failed to clear storage: %v", err)
		}
	}
	// Init may create indices, depending on the storage type
	if err := storage.Init(); err != nil {
		log.Fatal(err)
	}
	parser := NewParser(config)
	watcher := NewWatcher(config, parser, storage)
	scanner := NewScanner(config, watcher, parser, storage)
	err = scanner.Scan()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Scan completed")
	if err := storage.Watch(); err != nil {
		log.Fatal(err)
	}
	watcher.Watch()
}

type Config struct {
	Path            string   `yaml:"path"`
	StorageType     string   `yaml:"storage_type"`
	Conn            string   `yaml:"connection_uri"`
	ClearStorage    bool     `yaml:"clear_storage"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

func loadConfig(configPath string) (*Config, error) {
	config := &Config{
		Path:            ".",
		StorageType:     "memory",
		Conn:            "",
		ExcludePatterns: []string{},
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}
	return config, nil
}

type Scanner interface {
	Scan()
}

func NewScanner(config *Config, watcher Watcher, parser Parser, storage Storage) *DefaultScanner {
	patterns := make([]glob.Glob, 0, len(config.ExcludePatterns))
	for _, pattern := range config.ExcludePatterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			log.Printf("Invalid glob pattern %s: %v", pattern, err)
			continue
		}
		patterns = append(patterns, g)
	}

	return &DefaultScanner{
		path:    config.Path,
		watcher: watcher,
		parser:  parser,
		storage: storage,
		exclude: patterns,
	}
}

type DefaultScanner struct {
	path    string
	watcher Watcher
	parser  Parser
	storage Storage
	exclude []glob.Glob
}

func (s *DefaultScanner) isExcluded(path string) bool {
	for _, pattern := range s.exclude {
		if pattern.Match(path) {
			return true
		}
	}
	return false
}

func (s *DefaultScanner) Scan() error {
	err := filepath.Walk(s.path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if s.isExcluded(walkPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			if err = s.watcher.Add(walkPath); err != nil {
				return err
			}
			return nil
		}

		if filepath.Ext(walkPath) == ".md" {
			data, err := s.parser.Parse(walkPath)
			if err != nil {
				log.Printf("Error parsing markdown file %s: %v", walkPath, err)
				return nil
			}
			s.storage.Save(data)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

type Storage interface {
	Save(data File) error
	Update(data File) error
	Delete(path string) error
	Close() error
	Clear() error
	Init() error
	Watch() error
}

func NewStorage(storageType string, conn string) (Storage, error) {
	switch storageType {
	case "memory":
		return NewMemoryStorage()
	case "mongodb":
		return NewMongoDBStorage(conn)
	case "sqlite":
		return NewSQLiteStorage(conn)
	default:
		return nil, fmt.Errorf("invalid storage type: %s", storageType)
	}
}

var ErrNotFound = errors.New("not found")

type MemoryStorage struct {
	data map[string]File
}

func NewMemoryStorage() (*MemoryStorage, error) {
	return &MemoryStorage{
		data: make(map[string]File),
	}, nil
}

func (s *MemoryStorage) Save(data File) error {
	s.data[data.AbsPath] = data
	return nil
}

func (s *MemoryStorage) Update(data File) error {
	if _, ok := s.data[data.AbsPath]; !ok {
		return ErrNotFound
	}
	s.data[data.AbsPath] = data
	return nil
}

func (s *MemoryStorage) Delete(path string) error {
	if _, ok := s.data[path]; !ok {
		return ErrNotFound
	}
	delete(s.data, path)
	return nil
}

func (s *MemoryStorage) Close() error {
	return nil
}

func (s *MemoryStorage) Clear() error {
	s.data = make(map[string]File)
	return nil
}

func (s *MemoryStorage) Init() error {
	return nil
}

func (s *MemoryStorage) Watch() error {
	return nil
}

type MongoDBStorage struct {
	client     *mongo.Client
	collection *mongo.Collection
	ctx        context.Context
}

func NewMongoDBStorage(conn string) (*MongoDBStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	collection := client.Database("notes").Collection("files")

	return &MongoDBStorage{
		client:     client,
		collection: collection,
		ctx:        context.Background(),
	}, nil
}

func (s *MongoDBStorage) Save(data File) error {
	doc := bson.M{
		"_id":         data.RelPath,
		"slug":        data.Slug,
		"content":     data.Content,
		"frontmatter": data.FrontMatter,
		"updated":     time.Now(),
	}

	opts := options.Replace().SetUpsert(true)
	filter := bson.M{"_id": data.RelPath}

	_, err := s.collection.ReplaceOne(s.ctx, filter, doc, opts)
	return err
}

func (s *MongoDBStorage) Update(data File) error {
	filter := bson.M{"_id": data.RelPath}
	update := bson.M{
		"$set": bson.M{
			"content":     data.Content,
			"frontmatter": data.FrontMatter,
			"updated":     time.Now(),
		},
	}

	result, err := s.collection.UpdateOne(s.ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *MongoDBStorage) Delete(path string) error {
	filter := bson.M{"abs_path": path}
	update := bson.M{
		"$set": bson.M{
			"deleted": time.Now(),
		},
	}

	result, err := s.collection.UpdateOne(s.ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *MongoDBStorage) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.client.Disconnect(ctx)
}

func (s *MongoDBStorage) Clear() error {
	return s.collection.Drop(s.ctx)
}

func (s *MongoDBStorage) Init() error {
	_, err := s.collection.Indexes().CreateOne(s.ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "abs_path", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	return nil
}

func (s *MongoDBStorage) Watch() error {
	pipeline := mongo.Pipeline{}
	stream, err := s.collection.Watch(s.ctx, pipeline)
	if err != nil {
		return fmt.Errorf("failed to create change stream: %w", err)
	}

	go func() {
		defer stream.Close(s.ctx)

		for stream.Next(s.ctx) {
			var changeDoc struct {
				OperationType string `bson:"operationType"`
				FullDocument  File   `bson:"fullDocument"`
				DocumentKey   struct {
					ID interface{} `bson:"_id"`
				} `bson:"documentKey"`
			}

			if err := stream.Decode(&changeDoc); err != nil {
				log.Printf("Error decoding change stream document: %v", err)
				continue
			}

			switch changeDoc.OperationType {
			case "insert", "update", "replace":
				err := writeFileToDisk(changeDoc.FullDocument)
				if err != nil {
					log.Printf("Error writing file to disk: %v", err)
				}
			case "delete":
				// Get path from document key and delete file
				// This requires storing the path in the _id or retrieving it before deletion
				// For simplicity, we'll need to query for the path using the document key
				// This is a limitation of this approach
				log.Printf("Delete operation detected but path information is not available in change stream")
			}
		}

		if err := stream.Err(); err != nil {
			log.Printf("Error in change stream: %v", err)
		}
	}()
	return nil
}

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(conn string) (*SQLiteStorage, error) {
	// Create the directory for the database file if it doesn't exist
	dir := filepath.Dir(conn)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	return &SQLiteStorage{
		db: db,
	}, nil
}

func (s *SQLiteStorage) Save(data File) error {
	// Serialize frontmatter to JSON
	frontmatterJSON, err := json.Marshal(data.FrontMatter)
	if err != nil {
		return fmt.Errorf("failed to serialize frontmatter: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO files (path, slug, content, frontmatter, updated)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
		slug = excluded.slug,
		content = excluded.content,
		frontmatter = excluded.frontmatter,
		updated = excluded.updated
	`, data.RelPath, data.Slug, data.Content, string(frontmatterJSON), time.Now())

	return err
}

func (s *SQLiteStorage) Update(data File) error {
	// Serialize frontmatter to JSON
	frontmatterJSON, err := json.Marshal(data.FrontMatter)
	if err != nil {
		return fmt.Errorf("failed to serialize frontmatter: %w", err)
	}

	result, err := s.db.Exec(`
		UPDATE files
		SET path = ?, slug = ?, content = ?, frontmatter = ?, updated = ?
		WHERE path = ?
	`, data.RelPath, data.Slug, data.Content, string(frontmatterJSON), time.Now(), data.RelPath)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *SQLiteStorage) Delete(path string) error {
	result, err := s.db.Exec("DELETE FROM files WHERE path = ?", path)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func (s *SQLiteStorage) Clear() error {
	_, err := s.db.Exec("DROP TABLE IF EXISTS files")
	return err
}

func (s *SQLiteStorage) Init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			path TEXT PRIMARY KEY,
			slug TEXT,
			content TEXT,
			frontmatter TEXT,
			updated DATETIME,
			deleted DATETIME
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_path ON files(path)`)
	return err
}

func (s *SQLiteStorage) Watch() error {
	return nil
}

type WatcherEvent struct {
	EventType string
	Path      string
}

type Watcher interface {
	Init(path string, handler WatcherEventHandler)
	Add(path string) error
	Watch()
}

func NewWatcher(config *Config, parser Parser, storage Storage) Watcher {
	eventHandler := &DefaultEventHandler{
		config:  config,
		parser:  parser,
		storage: storage,
	}
	fsnotifyWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	watcher := &FSNotifyWatcher{
		watcher:      fsnotifyWatcher,
		eventHandler: eventHandler,
		parser:       parser,
	}
	return watcher
}

type FSNotifyWatcher struct {
	watcher      *fsnotify.Watcher
	eventHandler WatcherEventHandler
	parser       Parser
}

func (w *FSNotifyWatcher) Init(path string, handler WatcherEventHandler) {
}

func (w *FSNotifyWatcher) Add(path string) error {
	if err := w.watcher.Add(path); err != nil {
		return err
	}
	return nil
}

func (w *FSNotifyWatcher) Watch() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if filepath.Ext(event.Name) != ".md" {
				return
			}
			// Handle new directory creation
			if event.Op&fsnotify.Create == fsnotify.Create {
				// Check if the created item is a directory
				fi, err := os.Stat(event.Name)
				if err == nil && fi.IsDir() {
					// Skip dotdirs
					baseName := filepath.Base(event.Name)
					if len(baseName) > 0 && baseName[0] == '.' {
						continue
					}

					// Add the new directory to the watcher
					if err := w.watcher.Add(event.Name); err != nil {
						log.Println("Error watching new directory:", err)
					} else {
						fmt.Println("Added new directory to watch:", event.Name)
					}
				}
			}
			w.eventHandler.Handle(WatcherEvent{EventType: event.Op.String(), Path: event.Name})
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error:", err)
		}
	}
}

type WatcherEventHandler interface {
	Handle(event WatcherEvent)
}

type DefaultEventHandler struct {
	config  *Config
	storage Storage
	parser  Parser
}

func (h *DefaultEventHandler) Handle(event WatcherEvent) {
	switch event.EventType {
	case "CREATE", "WRITE":
		data, err := h.parser.Parse(event.Path)
		if err != nil {
			log.Printf("Error parsing markdown file %s: %v", event.Path, err)
			return
		}

		if event.EventType == "CREATE" {
			h.storage.Save(data)
		} else {
			h.storage.Update(data)
		}
	case "REMOVE", "RENAME":
		relPath, _ := filepath.Rel(h.config.Path, event.Path)
		h.storage.Delete(relPath)
	}
}

type File struct {
	FrontMatter map[string]interface{}
	Content     string
	AbsPath     string
	RelPath     string
	Slug        string
}

type Parser interface {
	Parse(path string) (File, error)
}

type DefaultParser struct {
	Config *Config
}

func NewParser(config *Config) Parser {
	return &DefaultParser{
		Config: config,
	}
}

func (p *DefaultParser) Parse(path string) (File, error) {
	relPath, _ := filepath.Rel(p.Config.Path, path)
	fileName := filepath.Base(path)
	slug := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	data := File{
		AbsPath:     path,
		RelPath:     relPath,
		Slug:        slug,
		FrontMatter: make(map[string]interface{}),
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return data, err
	}

	contentStr := string(content)

	// Check if file has frontmatter (starts with ---)
	if strings.HasPrefix(contentStr, "---\n") {
		// Find the closing frontmatter delimiter
		parts := strings.SplitN(contentStr[4:], "---\n", 2)
		if len(parts) == 2 {
			// Parse YAML frontmatter
			frontMatter := parts[0]
			err = yaml.Unmarshal([]byte(frontMatter), &data.FrontMatter)
			if err != nil {
				log.Printf("Error parsing frontmatter in %s: %v", path, err)
			}

			// Set content to everything after frontmatter
			data.Content = parts[1]
		} else {
			// Invalid frontmatter format
			data.Content = contentStr
		}
	} else {
		// No frontmatter
		data.Content = contentStr
	}

	return data, nil
}

func writeFileToDisk(file File) error {
	// Ensure directory exists
	dir := filepath.Dir(file.AbsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Construct file content with frontmatter if it exists
	var content strings.Builder

	if len(file.FrontMatter) > 0 {
		// Add frontmatter
		content.WriteString("---\n")
		frontmatterBytes, err := yaml.Marshal(file.FrontMatter)
		if err != nil {
			return fmt.Errorf("failed to marshal frontmatter: %w", err)
		}
		content.Write(frontmatterBytes)
		content.WriteString("---\n")
	}

	// Add content
	content.WriteString(file.Content)

	// Write to file
	if err := os.WriteFile(file.AbsPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
