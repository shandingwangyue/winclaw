package storage

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id"`
}

type Config struct {
	AI struct {
		Model       string  `json:"model"`
		APIKey      string  `json:"api_key"`
		BaseUrl     string  `json:"base_url"`
		Temperature float64 `json:"temperature"`
	} `json:"ai"`
	Voice struct {
		Enabled   bool   `json:"enabled"`
		AutoSpeak bool   `json:"auto_speak"`
		Language  string `json:"language"`
	} `json:"voice"`
	Permission struct {
		Level       string   `json:"level"`
		Whitelist   []string `json:"whitelist"`
		Blacklist   []string `json:"blacklist"`
		ConfirmMode string   `json:"confirm_mode"`
	} `json:"permission"`
	Storage struct {
		ConversationPath string `json:"conversation_path"`
		SaveHistory      bool   `json:"save_history"`
	} `json:"storage"`
}

var defaultConfig = Config{
	AI: struct {
		Model       string  `json:"model"`
		APIKey      string  `json:"api_key"`
		BaseUrl     string  `json:"base_url"`
		Temperature float64 `json:"temperature"`
	}{
		Model: "gpt-4o", BaseUrl: "https://api.openai.com/v1", Temperature: 0.7,
	},
	Voice: struct {
		Enabled   bool   `json:"enabled"`
		AutoSpeak bool   `json:"auto_speak"`
		Language  string `json:"language"`
	}{
		Enabled: true, AutoSpeak: false, Language: "zh-CN",
	},
	Permission: struct {
		Level       string   `json:"level"`
		Whitelist   []string `json:"whitelist"`
		Blacklist   []string `json:"blacklist"`
		ConfirmMode string   `json:"confirm_mode"`
	}{
		Level: "medium", ConfirmMode: "first",
	},
	Storage: struct {
		ConversationPath string `json:"conversation_path"`
		SaveHistory      bool   `json:"save_history"`
	}{
		ConversationPath: "conversations", SaveHistory: true,
	},
}

func NewDB(appDir string) (*DB, error) {
	dbPath := filepath.Join(appDir, "winclaw.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Setting{}, &Message{})
	if err != nil {
		return nil, err
	}

	d := &DB{db: db}
	if err := d.initConfig(appDir); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DB) initConfig(appDir string) error {
	var count int64
	d.db.Model(&Setting{}).Count(&count)
	if count == 0 {
		cfg, _ := json.Marshal(defaultConfig)
		d.db.Create(&Setting{Key: "config", Value: string(cfg)})
	}
	return nil
}

func (d *DB) GetConfig() (Config, error) {
	var setting Setting
	err := d.db.Where("key = ?", "config").First(&setting).Error
	if err != nil {
		return defaultConfig, err
	}
	var cfg Config
	if err := json.Unmarshal([]byte(setting.Value), &cfg); err != nil {
		return defaultConfig, err
	}
	return cfg, nil
}

func (d *DB) SaveConfig(cfg Config) error {
	data, _ := json.Marshal(cfg)
	var setting Setting
	if err := d.db.Where("key = ?", "config").First(&setting).Error; err != nil {
		return d.db.Create(&setting).Error
	}
	setting.Value = string(data)
	return d.db.Save(&setting).Error
}

func (d *DB) SaveMessage(msg *Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
		msg.Timestamp = time.Now()
	}
	return d.db.Create(msg).Error
}

func (d *DB) GetMessages(sessionID string) ([]Message, error) {
	var messages []Message
	err := d.db.Where("session_id = ?", sessionID).Order("timestamp asc").Find(&messages).Error
	return messages, err
}

func (d *DB) NewSession() string {
	return uuid.New().String()
}

func (d *DB) GetAppDir() string {
	return ""
}
