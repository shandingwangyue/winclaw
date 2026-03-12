package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type MarkdownStore struct {
	BasePath string
}

func NewMarkdownStore(basePath string) *MarkdownStore {
	os.MkdirAll(basePath, 0755)
	return &MarkdownStore{BasePath: basePath}
}

func (m *MarkdownStore) SaveConversation(sessionID string, messages []Message) error {
	filename := fmt.Sprintf("%s.md", time.Now().Format("2006-01-02_15-04"))
	filepath := filepath.Join(m.BasePath, filename)

	content := fmt.Sprintf("# Conversation - %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	content += "---\n\n"

	for _, msg := range messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Assistant"
		}
		content += fmt.Sprintf("**%s** (%s):\n\n%s\n\n---\n\n",
			role, msg.Timestamp.Format("15:04:05"), msg.Content)
	}

	return os.WriteFile(filepath, []byte(content), 0644)
}

func (m *MarkdownStore) ListConversations() ([]string, error) {
	entries, err := os.ReadDir(m.BasePath)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

func (m *MarkdownStore) GetConversation(filename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(m.BasePath, filename))
	return string(data), err
}
