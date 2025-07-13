package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"strings"

	"github.com/go-kivik/kivik/v4"
	_ "github.com/go-kivik/kivik/v4/couchdb"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "couch-sync",
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start the server",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "couch", Value: "https://example.com/db", Usage: "CouchDB URL"},
					&cli.StringFlag{Name: "db", Value: "notes-sync-test", Usage: "Database name"},
					&cli.IntFlag{Name: "port", Value: 8080, Usage: "HTTP port"},
				},
				Action: func(c *cli.Context) error {
					couchURL := c.String("couch")
					dbName := c.String("db")
					port := c.Int("port")

					client, err := kivik.New("couch", couchURL)
					if err != nil {
						return err
					}

					db := client.DB(dbName)

					go loadNotes(db)
					// go syncFromDB(db)

					http.Handle("/", http.FileServer(http.Dir("web")))
					log.Printf("Serving on :%d", port)
					return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func loadNotes(db *kivik.DB) {
	files, err := os.ReadDir("notes")
	if err != nil {
		log.Println(err)
		return
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		path := filepath.Join("notes", f.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			log.Println(err)
			continue
		}

		docID := strings.TrimSuffix(f.Name(), ".md")
		doc := map[string]interface{}{"content": string(content)}
		_, err = db.Put(context.Background(), docID, doc)
		if err != nil {
			log.Println(err)
		} else {
			log.Printf("Loaded %s into database", docID)
		}
	}
}

func syncFromDB(db *kivik.DB) {
	changes := db.Changes(context.Background(), kivik.Params(map[string]interface{}{"feed": "continuous", "since": "now"}))
	defer changes.Close()

	for changes.Next() {
		if changes.Deleted() {
			// Handle deletion if needed
			continue
		}
		var doc map[string]interface{}
		if err := changes.ScanDoc(&doc); err != nil {
			log.Println(err)
			continue
		}
		content, ok := doc["content"].(string)
		if !ok {
			continue
		}
		filePath := filepath.Join("notes", changes.ID()+".md")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			log.Println(err)
		} else {
			log.Printf("Synced %s to file", changes.ID())
		}
	}
	if err := changes.Err(); err != nil {
		log.Println(err)
	}
}
