package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type Note struct {
	ID          string      `json:"id"`
	Slug        string      `json:"slug"`
	Path        string      `json:"path"`
	Frontmatter interface{} `json:"frontmatter"`
	Content     string      `json:"content"`
	Created     string      `json:"created"`
	Modified    string      `json:"modified"`
}

func main() {
	cmd := &cli.Command{
		Name: "export",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "home",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			repoPath := c.String("home")
			if repoPath == "" {
				return fmt.Errorf("please provide a path to the notes repository")
			}

			notes, err := buildNotes(repoPath)
			if err != nil {
				return err
			}

			notesJSON, err := json.MarshalIndent(notes, "", "  ")
			if err != nil {
				return err
			}

			err = os.WriteFile("notes.json", notesJSON, 0644)
			if err != nil {
				return err
			}

			fmt.Println("notes.json has been created successfully.")
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func buildNotes(repoPath string) ([]Note, error) {
	var notes []Note

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() && filepath.Base(path) == "templates" {
			return filepath.SkipDir
		}

		if !info.IsDir() && filepath.Ext(path) == ".md" {
			rawContent, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			slug := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			id := generateID(slug)
			frontmatter, content := extractFrontmatter(string(rawContent), path)
			note := Note{
				ID:          id,
				Slug:        slug,
				Path:        path,
				Frontmatter: frontmatter,
				Content:     content,
				Created:     info.ModTime().Format(time.RFC3339),
				Modified:    info.ModTime().Format(time.RFC3339),
			}

			notes = append(notes, note)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return notes, nil
}

func extractFrontmatter(content string, path string) (interface{}, string) {
	var frontmatter interface{}
	var yamlContent string
	var restContent string

	if len(content) > 3 && content[:3] == "---" {
		end := 3
		for i := 3; i < len(content); i++ {
			if content[i:i+3] == "---" {
				end = i
				break
			}
		}
		yamlContent = content[3:end]
		restContent = content[end+3:]
	} else {
		restContent = content
	}

	if yamlContent != "" {
		err := yaml.Unmarshal([]byte(yamlContent), &frontmatter)
		if err != nil {
			fmt.Println("Error parsing YAML frontmatter: %v, %s", err, path)
			frontmatter = nil
		}
	}

	return frontmatter, restContent
}

func generateID(slug string) string {
	hash := sha256.Sum256([]byte(slug))
	return hex.EncodeToString(hash[:])
}
