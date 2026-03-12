package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type OpenAIClient struct {
	APIKey      string
	BaseUrl     string
	Model       string
	Temperature float64
	Messages    []Message
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type Response struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func NewOpenAIClient(apiKey, baseUrl, model string, temp float64) *OpenAIClient {
	if baseUrl == "" {
		baseUrl = "https://api.openai.com/v1"
	}
	return &OpenAIClient{
		APIKey:      apiKey,
		BaseUrl:     baseUrl,
		Model:       model,
		Temperature: temp,
		Messages:    []Message{},
	}
}

func (c *OpenAIClient) AddMessage(role, content string) {
	c.Messages = append(c.Messages, Message{Role: role, Content: content})
}

func (c *OpenAIClient) ClearMessages() {
	c.Messages = []Message{}
}

func (c *OpenAIClient) Chat(systemPrompt string, userMessage string) (string, error) {
	if c.APIKey == "" {
		return "请先在设置中配置OpenAI API Key", nil
	}

	messages := []Message{{Role: "system", Content: systemPrompt}}
	messages = append(messages, c.Messages...)
	messages = append(messages, Message{Role: "user", Content: userMessage})

	req := Request{
		Model:       c.Model,
		Messages:    messages,
		Temperature: c.Temperature,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	url := c.BaseUrl
	if url == "" {
		url = "https://api.openai.com/v1"
	}
	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}

	httpReq, err := http.NewRequest("POST", url+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	reply := response.Choices[0].Message.Content
	c.AddMessage("user", userMessage)
	c.AddMessage("assistant", reply)

	return reply, nil
}

func (c *OpenAIClient) SetConfig(apiKey, baseUrl, model string, temp float64) {
	c.APIKey = apiKey
	c.BaseUrl = baseUrl
	c.Model = model
	c.Temperature = temp
}
