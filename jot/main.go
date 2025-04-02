package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

func main() {
	pref := tele.Settings{
		Token: os.Getenv("TELEGRAM_BOT_TOKEN"),
		Poller: &tele.LongPoller{
			Timeout: 10 * time.Second,
			AllowedUpdates: []string{
				"message",
				"edited_message",
				"channel_post",
				"edited_channel_post",
			},
		},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}
	adminIDStr := os.Getenv("BOT_ADMIN_ID")
	if adminIDStr == "" {
		log.Fatal("BOT_ADMIN_ID environment variable must be set")
	}
	adminID, err := strconv.ParseInt(strings.TrimSpace(adminIDStr), 10, 64)
	if err != nil {
		log.Fatalf("Invalid BOT_ADMIN_ID: %v", err)
	}
	filenameTemplate := os.Getenv("FILENAME_TEMPLATE")
	if filenameTemplate == "" {
		log.Fatal("FILENAME_TEMPLATE environment variable must be set")
	}
	b.Use(middleware.Whitelist(adminID))
	saveDir := os.Getenv("INBOX_PATH")
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		log.Fatal("Failed to create save directory:", err)
	}

	b.Handle(tele.OnText, handler(saveDir, filenameTemplate))
	b.Handle(tele.OnChannelPost, handler(saveDir, filenameTemplate))
	b.Handle(tele.OnEdited, handler(saveDir, filenameTemplate))
	b.Handle(tele.OnEditedChannelPost, handler(saveDir, filenameTemplate))
	log.Println("Bot starting...")
	b.Start()

}

func handler(saveDir string, filenameTemplate string) func(tele.Context) error {
	return func(c tele.Context) error {
		saveMessage(c.Message(), saveDir, filenameTemplate)
		return nil
	}
}

type MessageContext struct {
	Source    string
	MessageID int
	Tags      []string
	Created   string
	Modified  string
	Content   string
}

func saveMessage(m *tele.Message, saveDir string, filenameTemplate string) {
	filename := m.Time().Format(filenameTemplate)
	filepath := filepath.Join(saveDir, filename)
	tmpl, err := template.ParseFiles("template.md.tmpl")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		return
	}
	context := MessageContext{
		Source:    "telegram",
		MessageID: m.ID,
		Tags:      []string{"telegram", "inbox"},
		Created:   m.Time().Format(time.RFC3339),
		Modified:  time.Now().Format(time.RFC3339),
		Content:   m.Text,
	}

	var content strings.Builder
	if err := tmpl.Execute(&content, context); err != nil {
		log.Printf("Error executing template: %v", err)
		return
	}
	if err := os.WriteFile(filepath, []byte(content.String()), 0644); err != nil {
		log.Printf("Error saving message to file: %v", err)
	} else {
		log.Printf("Message saved to %s", filepath)
	}
}
