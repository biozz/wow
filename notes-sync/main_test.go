package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestParseMarkdownFile(t *testing.T) {
	tmpDir := os.TempDir()
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		expected MarkdownData
	}{
		{
			name: "with_valid_frontmatter",
			content: `---
title: Test Document
tags:
  - golang
  - testing
date: 2023-05-01
---
# Test Content
This is a test markdown file.`,
			expected: MarkdownData{
				FrontMatter: map[string]interface{}{
					"title": "Test Document",
					"tags":  []interface{}{"golang", "testing"},
					"date":  "2023-05-01",
				},
				Content: "# Test Content\nThis is a test markdown file.",
			},
		},
		{
			name: "without_frontmatter",
			content: `# No Frontmatter
Just content here.`,
			expected: MarkdownData{
				FrontMatter: map[string]interface{}{},
				Content:     "# No Frontmatter\nJust content here.",
			},
		},
		{
			name: "with_invalid_frontmatter",
			content: `---
invalid: yaml:
  - missing colon
---
# Content with invalid frontmatter`,
			expected: MarkdownData{
				FrontMatter: map[string]interface{}{},
				Content: `---
invalid: yaml:
  - missing colon
---
# Content with invalid frontmatter`,
			},
		},
		{
			name: "with_empty_frontmatter",
			content: `---
---
# Content with empty frontmatter`,
			expected: MarkdownData{
				FrontMatter: map[string]interface{}{},
				Content:     "# Content with empty frontmatter",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tmpDir, tc.name+".md")
			err := os.WriteFile(filePath, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse the file
			got, err := parseMarkdownFile(filePath)
			if err != nil {
				t.Fatalf("parseMarkdownFile failed: %v", err)
			}

			// Check path
			if got.Path != filePath {
				t.Errorf("Expected path %s, got %s", filePath, got.Path)
			}

			// Check content
			if got.Content != tc.expected.Content {
				t.Errorf("Content mismatch\nExpected: %q\nGot: %q", tc.expected.Content, got.Content)
			}

			// Check frontmatter (excluding Path which is set dynamically)
			if !reflect.DeepEqual(got.FrontMatter, tc.expected.FrontMatter) {
				t.Errorf("FrontMatter mismatch\nExpected: %+v\nGot: %+v", tc.expected.FrontMatter, got.FrontMatter)
			}
		})
	}

	// Test non-existent file
	t.Run("non_existent_file", func(t *testing.T) {
		_, err := parseMarkdownFile(filepath.Join(tmpDir, "does-not-exist.md"))
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})
}

func TestMemoryStorage(t *testing.T) {
	storage := &MemoryStorage{
		data: make(map[string]MarkdownData),
	}

	testData := MarkdownData{
		Path:    "/test/path.md",
		Content: "Test content",
		FrontMatter: map[string]interface{}{
			"title": "Test",
		},
	}

	// Test Save
	t.Run("Save", func(t *testing.T) {
		err := storage.Save(testData)
		if err != nil {
			t.Errorf("Save failed: %v", err)
		}

		// Verify data was stored
		stored, ok := storage.data[testData.Path]
		if !ok {
			t.Error("Data not found in storage after Save")
		}
		if !reflect.DeepEqual(stored, testData) {
			t.Errorf("Stored data mismatch\nExpected: %+v\nGot: %+v", testData, stored)
		}
	})

	// Test Update
	t.Run("Update_Success", func(t *testing.T) {
		updatedData := testData
		updatedData.Content = "Updated content"

		err := storage.Update(updatedData)
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}

		// Verify data was updated
		stored := storage.data[testData.Path]
		if stored.Content != updatedData.Content {
			t.Errorf("Expected updated content %q, got %q", updatedData.Content, stored.Content)
		}
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		nonExistentData := MarkdownData{
			Path: "/non/existent.md",
		}

		err := storage.Update(nonExistentData)
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	// Test Delete
	t.Run("Delete_Success", func(t *testing.T) {
		err := storage.Delete(testData.Path)
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		// Verify data was deleted
		_, ok := storage.data[testData.Path]
		if ok {
			t.Error("Data found in storage after Delete")
		}
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		err := storage.Delete("/non/existent.md")
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}

// Mock storage for testing event handler
type MockStorage struct {
	SaveCalled   bool
	UpdateCalled bool
	DeleteCalled bool
	LastPath     string
	LastData     MarkdownData
}

func (m *MockStorage) Save(data MarkdownData) error {
	m.SaveCalled = true
	m.LastPath = data.Path
	m.LastData = data
	return nil
}

func (m *MockStorage) Update(data MarkdownData) error {
	m.UpdateCalled = true
	m.LastPath = data.Path
	m.LastData = data
	return nil
}

func (m *MockStorage) Delete(path string) error {
	m.DeleteCalled = true
	m.LastPath = path
	return nil
}

func (m *MockStorage) Close() error {
	return nil
}

func TestDefaultEventHandler(t *testing.T) {
	// Create temp file for testing
	tmpFile, err := os.CreateTemp("", "handler-test*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "# Test Content"
	if _, err := tmpFile.Write([]byte(testContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	mockStorage := &MockStorage{}
	handler := &DefaultEventHandler{storage: mockStorage}

	tests := []struct {
		name      string
		eventType string
		checkFunc func(*testing.T)
	}{
		{
			name:      "CREATE event",
			eventType: "CREATE",
			checkFunc: func(t *testing.T) {
				if !mockStorage.SaveCalled {
					t.Error("Save was not called for CREATE event")
				}
				if mockStorage.UpdateCalled {
					t.Error("Update was incorrectly called for CREATE event")
				}
				if mockStorage.DeleteCalled {
					t.Error("Delete was incorrectly called for CREATE event")
				}
			},
		},
		{
			name:      "WRITE event",
			eventType: "WRITE",
			checkFunc: func(t *testing.T) {
				if !mockStorage.UpdateCalled {
					t.Error("Update was not called for WRITE event")
				}
			},
		},
		{
			name:      "REMOVE event",
			eventType: "REMOVE",
			checkFunc: func(t *testing.T) {
				if !mockStorage.DeleteCalled {
					t.Error("Delete was not called for REMOVE event")
				}
				if mockStorage.LastPath != tmpFile.Name() {
					t.Errorf("Wrong path, expected %q, got %q", tmpFile.Name(), mockStorage.LastPath)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			*mockStorage = MockStorage{}
			handler.handle(WatcherEvent{
				EventType: tc.eventType,
				Path:      tmpFile.Name(),
			})
			tc.checkFunc(t)
		})
	}
}

func TestMongoDBStorage(t *testing.T) {
	// Skip if no MongoDB connection is available
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		t.Skip("Skipping MongoDB test: MONGODB_URI environment variable not set")
	}

	// Setup MongoDB storage
	storage, err := NewMongoDBStorage(mongoURI)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer storage.Close()

	// Clean up any test data that might exist
	testPath := "/test/mongodb-test.md"
	_ = storage.Delete(testPath)

	testData := MarkdownData{
		Path:    testPath,
		Content: "Test content for MongoDB",
		FrontMatter: map[string]interface{}{
			"title": "MongoDB Test",
			"tags":  []string{"test", "mongodb"},
		},
	}

	// Test Save
	t.Run("Save", func(t *testing.T) {
		err := storage.Save(testData)
		if err != nil {
			t.Errorf("Save failed: %v", err)
		}
	})

	// Test Update
	t.Run("Update_Success", func(t *testing.T) {
		updatedData := testData
		updatedData.Content = "Updated content for MongoDB"

		err := storage.Update(updatedData)
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}

		// Verify data was updated by querying MongoDB
		filter := bson.M{"path": testData.Path}
		var result bson.M
		err = storage.collection.FindOne(storage.ctx, filter).Decode(&result)
		if err != nil {
			t.Errorf("Failed to find document: %v", err)
		}

		if content, ok := result["content"].(string); !ok || content != updatedData.Content {
			t.Errorf("Expected updated content %q, got %q", updatedData.Content, content)
		}
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		nonExistentData := MarkdownData{
			Path: "/non/existent/mongodb.md",
		}

		err := storage.Update(nonExistentData)
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	// Test Delete
	t.Run("Delete_Success", func(t *testing.T) {
		err := storage.Delete(testData.Path)
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		// Verify data was deleted by querying MongoDB
		filter := bson.M{"path": testData.Path}
		count, err := storage.collection.CountDocuments(storage.ctx, filter)
		if err != nil {
			t.Errorf("Failed to count documents: %v", err)
		}
		if count != 0 {
			t.Errorf("Document still exists after Delete")
		}
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		err := storage.Delete("/non/existent/mongodb.md")
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}
