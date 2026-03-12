package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

type PromptManager struct {
	templates     map[string]*PromptTemplate
	systemPrompts map[string]string
	hooks         []PromptHook
}

type PromptTemplate struct {
	Name      string
	Template  string
	Variables []string
	CreatedAt time.Time
}

type PromptHook func(prompt string, context *PromptContext) string

type PromptContext struct {
	SessionID string
	UserID    string
	Channel   string
	Message   string
	History   []Message
	Context   map[string]interface{}
	Time      string
	Metadata  map[string]interface{}
}

func NewPromptManager() *PromptManager {
	pm := &PromptManager{
		templates:     make(map[string]*PromptTemplate),
		systemPrompts: make(map[string]string),
		hooks:         []PromptHook{},
	}
	pm.registerDefaultTemplates()
	return pm
}

func (pm *PromptManager) registerDefaultTemplates() {
	pm.AddTemplate("default", `{{.SystemPrompt}}

## Current Context
- Time: {{.Time}}
- Channel: {{.Channel}}
- User: {{.UserID}}

{{if .History}}
## Conversation History
{{range .History}}
- {{.Role}}: {{.Content}}
{{end}}
{{end}}

{{if .Context}}
## Session Context
{{range $key, $value := .Context}}
- {{$key}}: {{$value}}
{{end}}
{{end}}

## Current Message
{{.Message}}

Response:`, []string{"SystemPrompt", "Time", "Channel", "UserID", "Message", "History", "Context"})

	pm.AddTemplate("concise", `{{.SystemPrompt}}

Context: {{.Channel}} | {{.Time}}
{{if .History}}History: {{len .History}} messages{{end}}
User: {{.Message}}

Response:`, []string{"SystemPrompt", "Time", "Channel", "History", "Message"})

	pm.AddTemplate("detailed", `{{.SystemPrompt}}

## Session Information
- Session ID: {{.SessionID}}
- User ID: {{.UserID}}
- Channel: {{.Channel}}
- Timestamp: {{.Time}}

## Conversation History
{{range .History}}
### {{.Role}}
{{.Content}}
{{end}}

## Current Context
{{range $key, $value := .Context}}
- **{{$key}}**: {{$value}}
{{end}}

## User Request
{{.Message}}

Please analyze and respond:`, []string{"SessionID", "SystemPrompt", "Time", "Channel", "UserID", "History", "Context", "Message"})

	pm.AddTemplate("tools", `{{.SystemPrompt}}

## Available Tools
{{.Tools}}

## Context
- Time: {{.Time}}
- Channel: {{.Channel}}

{{if .History}}
## Recent History (last {{.HistoryLength}} messages)
{{range .History}}
{{.Role}}: {{.Content}}
{{end}}
{{end}}

## User Input
{{.Message}}

Think step by step. Use tools if needed.`, []string{"SystemPrompt", "Tools", "Time", "Channel", "History", "HistoryLength", "Message"})

	pm.systemPrompts["general"] = `You are a helpful AI assistant. Provide accurate, concise, and helpful responses.`
	pm.systemPrompts["technical"] = `You are a technical assistant with deep knowledge in software development, system administration, and IT. Provide detailed technical explanations.`
	pm.systemPrompts["creative"] = `You are a creative assistant. Think creatively, suggest innovative ideas, and help with creative tasks.`
	pm.systemPrompts["analytical"] = `You are an analytical assistant. Analyze data carefully, think logically, and provide evidence-based conclusions.`
	pm.systemPrompts["customer_service"] = `You are a customer service representative. Be polite, patient, and helpful. Address customer concerns professionally.`
}

func (pm *PromptManager) AddTemplate(name, tmpl string, vars []string) {
	pm.templates[name] = &PromptTemplate{
		Name:      name,
		Template:  tmpl,
		Variables: vars,
		CreatedAt: time.Now(),
	}
}

func (pm *PromptManager) GetTemplate(name string) (*PromptTemplate, bool) {
	tmpl, ok := pm.templates[name]
	return tmpl, ok
}

func (pm *PromptManager) ListTemplates() []string {
	names := make([]string, 0, len(pm.templates))
	for name := range pm.templates {
		names = append(names, name)
	}
	return names
}

func (pm *PromptManager) BuildPrompt(systemPrompt string, session *Session, req *ProcessRequest) (string, error) {
	tmpl, ok := pm.templates["default"]
	if !ok {
		return fmt.Sprintf("%s\n\nUser: %s", systemPrompt, req.Content), nil
	}

	historyLength := 10
	if len(session.Messages) > historyLength {
		historyLength = len(session.Messages)
	}

	contextMap := make(map[string]interface{})
	if session.Context != nil {
		for k, v := range session.Context {
			contextMap[k] = v
		}
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			contextMap[k] = v
		}
	}

	data := PromptContext{
		SessionID: session.ID,
		UserID:    session.UserID,
		Channel:   session.Channel,
		Message:   req.Content,
		History:   session.Messages,
		Context:   contextMap,
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		Metadata:  req.Metadata,
	}

	t, err := template.New(tmpl.Name).Parse(tmpl.Template)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	if err := t.Execute(&builder, map[string]interface{}{
		"SystemPrompt":  systemPrompt,
		"SessionID":     data.SessionID,
		"UserID":        data.UserID,
		"Channel":       data.Channel,
		"Message":       data.Message,
		"History":       data.History,
		"HistoryLength": historyLength,
		"Context":       data.Context,
		"Time":          data.Time,
		"Metadata":      data.Metadata,
	}); err != nil {
		return "", err
	}

	prompt := builder.String()

	for _, hook := range pm.hooks {
		prompt = hook(prompt, &data)
	}

	return prompt, nil
}

func (pm *PromptManager) BuildPromptWithTemplate(templateName, systemPrompt string, session *Session, req *ProcessRequest) (string, error) {
	tmpl, ok := pm.templates[templateName]
	if !ok {
		return pm.BuildPrompt(systemPrompt, session, req)
	}

	data := PromptContext{
		SessionID: session.ID,
		UserID:    session.UserID,
		Channel:   session.Channel,
		Message:   req.Content,
		History:   session.Messages,
		Context:   session.Context,
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		Metadata:  req.Metadata,
	}

	t, err := template.New(tmpl.Name).Parse(tmpl.Template)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	err = t.Execute(&builder, map[string]interface{}{
		"SystemPrompt": systemPrompt,
		"SessionID":    data.SessionID,
		"UserID":       data.UserID,
		"Channel":      data.Channel,
		"Message":      data.Message,
		"History":      data.History,
		"Context":      data.Context,
		"Time":         data.Time,
		"Metadata":     data.Metadata,
	})
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

func (pm *PromptManager) AddHook(hook PromptHook) {
	pm.hooks = append(pm.hooks, hook)
}

func (pm *PromptManager) GetSystemPrompt(promptType string) string {
	if prompt, ok := pm.systemPrompts[promptType]; ok {
		return prompt
	}
	return pm.systemPrompts["general"]
}

func (pm *PromptManager) SetSystemPrompt(promptType, prompt string) {
	pm.systemPrompts[promptType] = prompt
}

func (pm *PromptManager) MergeSystemPrompts(prompts []string) string {
	return strings.Join(prompts, "\n\n")
}

func (pm *PromptManager) BuildDynamicPrompt(basePrompt string, dynamicData map[string]interface{}) string {
	prompt := basePrompt

	for key, value := range dynamicData {
		var replacement string
		switch v := value.(type) {
		case string:
			replacement = v
		case []string:
			replacement = strings.Join(v, ", ")
		default:
			data, _ := json.Marshal(v)
			replacement = string(data)
		}
		prompt = strings.ReplaceAll(prompt, fmt.Sprintf("{{.%s}}", key), replacement)
	}

	return prompt
}
