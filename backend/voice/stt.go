package voice

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type STT struct {
	apiKey string
}

type TTS struct {
	apiKey string
}

func NewSTT(apiKey string) *STT {
	return &STT{apiKey: apiKey}
}

func (s *STT) Recognize(audioData []byte) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("API key not configured")
	}
	return "", fmt.Errorf("STT not implemented - need audio file")
}

func (s *STT) RecognizeFromFile(filepath string) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("API key not configured")
	}
	return "Transcription placeholder", nil
}

func (t *TTS) Speak(text string) error {
	if t.apiKey == "" {
		return fmt.Errorf("API key not configured")
	}
	fmt.Println("Speaking:", text)
	return nil
}

func NewTTS(apiKey string) *TTS {
	return &TTS{apiKey: apiKey}
}

func init() {
	_ = http.MethodGet
	_ = time.Second
	_ = os.Args
	_ = exec.Command
}
