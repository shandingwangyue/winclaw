package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"winclaw/backend/ai"
	"winclaw/backend/skills"
)

type AgentCore struct {
	config        *AgentConfig
	aiClient      *ai.OpenAIClient
	skillManager  *skills.SkillManager
	promptManager *PromptManager
	mcpClients    map[string]*MCPClient
	channels      map[string]ChannelAdapter
	sessionStore  *SessionStore
}

type AgentConfig struct {
	Name             string  `json:"name"`
	Model            string  `json:"model"`
	APIKey           string  `json:"api_key"`
	BaseURL          string  `json:"base_url"`
	Temperature      float64 `json:"temperature"`
	MaxTokens        int     `json:"max_tokens"`
	SystemPrompt     string  `json:"system_prompt"`
	EnableMCP        bool    `json:"enable_mcp"`
	EnableChannels   bool    `json:"enable_channels"`
	ContextWindow    int     `json:"context_window"`
	ConversationType string  `json:"conversation_type"`
}

type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Channel   string                 `json:"channel"`
	Messages  []Message              `json:"messages"`
	Context   map[string]interface{} `json:"context"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func NewAgentCore(cfg *AgentConfig, skillMgr *skills.SkillManager) *AgentCore {
	ac := &AgentCore{
		config:        cfg,
		skillManager:  skillMgr,
		promptManager: NewPromptManager(),
		mcpClients:    make(map[string]*MCPClient),
		channels:      make(map[string]ChannelAdapter),
		sessionStore:  NewSessionStore(),
	}

	if cfg.APIKey != "" {
		ac.aiClient = ai.NewOpenAIClient(cfg.APIKey, cfg.BaseURL, cfg.Model, cfg.Temperature)
	}

	return ac
}

func (ac *AgentCore) ProcessMessage(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
	session, err := ac.sessionStore.GetOrCreateSession(req.SessionID, req.UserID, req.Channel)
	if err != nil {
		return nil, err
	}

	session.Messages = append(session.Messages, Message{
		Role:      "user",
		Content:   req.Content,
		Timestamp: time.Now(),
		Metadata:  req.Metadata,
	})

	prompt, err := ac.promptManager.BuildPrompt(ac.config.SystemPrompt, session, req)
	if err != nil {
		return nil, err
	}

	ac.updateContext(session, req)

	toolDefs := ac.buildToolDefinitions()
	toolCall, err := ac.planToolCalls(req.Content, prompt, toolDefs)
	if err != nil {
		return nil, err
	}

	var toolResults []ToolResult
	if toolCall != nil {
		toolResults, err = ac.executeTools(ctx, toolCall)
		if err != nil {
			return nil, err
		}
		prompt = ac.injectToolResults(prompt, toolResults)
	}

	response, err := ac.aiClient.Chat(prompt, req.Content)
	if err != nil {
		return nil, err
	}

	session.Messages = append(session.Messages, Message{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	})

	ac.sessionStore.SaveSession(session)

	return &ProcessResponse{
		SessionID: session.ID,
		Content:   response,
		Metadata: map[string]interface{}{
			"tools_used":        len(toolResults),
			"conversation_type": ac.config.ConversationType,
		},
	}, nil
}

type ProcessRequest struct {
	SessionID string
	UserID    string
	Channel   string
	Content   string
	Metadata  map[string]interface{}
}

type ProcessResponse struct {
	SessionID string
	Content   string
	Metadata  map[string]interface{}
}

func (ac *AgentCore) updateContext(session *Session, req *ProcessRequest) {
	if session.Context == nil {
		session.Context = make(map[string]interface{})
	}
	session.Context["last_request"] = req.Content
	session.Context["last_channel"] = req.Channel
	session.UpdatedAt = time.Now()
}

func (ac *AgentCore) buildToolDefinitions() []ToolDefinition {
	var tools []ToolDefinition

	tools = append(tools, ToolDefinition{
		Name:        "execute_skill",
		Description: "Execute a predefined skill or function",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"skill_name": map[string]string{"type": "string"},
				"params":     map[string]interface{}{"type": "object"},
			},
			"required": []string{"skill_name"},
		},
	})

	tools = append(tools, ToolDefinition{
		Name:        "mcp_call",
		Description: "Call an MCP server tool",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"server": map[string]string{"type": "string"},
				"tool":   map[string]string{"type": "string"},
				"params": map[string]interface{}{"type": "object"},
			},
			"required": []string{"server", "tool"},
		},
	})

	tools = append(tools, ToolDefinition{
		Name:        "route_to_agent",
		Description: "Route to a specialized sub-agent",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"agent_type": map[string]string{"type": "string"},
				"task":       map[string]string{"type": "string"},
			},
			"required": []string{"agent_type", "task"},
		},
	})

	tools = append(tools, ToolDefinition{
		Name:        "get_context",
		Description: "Get information from conversation context",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]string{"type": "string"},
			},
			"required": []string{"key"},
		},
	})

	return tools
}

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolCall struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

type ToolResult struct {
	Tool   string `json:"tool"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

func (ac *AgentCore) planToolCalls(userInput, prompt string, tools []ToolDefinition) (*ToolCall, error) {
	toolsJSON, _ := json.Marshal(tools)
	planningPrompt := fmt.Sprintf(`Based on the user input, determine if tools are needed.
User Input: %s

Available Tools:
%s

Respond with JSON for a single tool call if needed, or null if no tools required:
{"name": "tool_name", "params": {"param1": "value1"}}`, userInput, string(toolsJSON))

	response, err := ac.aiClient.Chat(planningPrompt, prompt)
	if err != nil {
		return nil, err
	}

	var toolCall ToolCall
	if err := json.Unmarshal([]byte(response), &toolCall); err != nil {
		return nil, nil
	}

	if toolCall.Name == "" {
		return nil, nil
	}

	return &toolCall, nil
}

func (ac *AgentCore) executeTools(ctx context.Context, toolCall *ToolCall) ([]ToolResult, error) {
	var results []ToolResult

	switch toolCall.Name {
	case "execute_skill":
		skillName, _ := toolCall.Params["skill_name"].(string)
		params, _ := toolCall.Params["params"].(map[string]interface{})
		result, err := ac.skillManager.ExecuteSkill(skillName, params)
		results = append(results, ToolResult{
			Tool:   "execute_skill",
			Result: result,
			Error:  errString(err),
		})

	case "mcp_call":
		server, _ := toolCall.Params["server"].(string)
		tool, _ := toolCall.Params["tool"].(string)
		params, _ := toolCall.Params["params"].(map[string]interface{})
		if mcp, ok := ac.mcpClients[server]; ok {
			result, err := mcp.CallTool(ctx, tool, params)
			results = append(results, ToolResult{
				Tool:   "mcp_call",
				Result: result,
				Error:  errString(err),
			})
		}

	case "route_to_agent":
		agentType, _ := toolCall.Params["agent_type"].(string)
		task, _ := toolCall.Params["task"].(string)
		result, err := ac.routeToAgent(ctx, agentType, task)
		results = append(results, ToolResult{
			Tool:   "route_to_agent",
			Result: result,
			Error:  errString(err),
		})

	case "get_context":
		key, _ := toolCall.Params["key"].(string)
		session := ac.sessionStore.GetCurrentSession()
		var result string
		if session != nil && session.Context != nil {
			if val, ok := session.Context[key]; ok {
				data, _ := json.Marshal(val)
				result = string(data)
			}
		}
		results = append(results, ToolResult{
			Tool:   "get_context",
			Result: result,
		})
	}

	return results, nil
}

func (ac *AgentCore) injectToolResults(prompt string, results []ToolResult) string {
	var injected string
	for _, r := range results {
		if r.Error != "" {
			injected += fmt.Sprintf("\n[Tool Error: %s] %s", r.Tool, r.Error)
		} else {
			injected += fmt.Sprintf("\n[Tool Result: %s] %s", r.Tool, r.Result)
		}
	}
	return prompt + injected
}

func (ac *AgentCore) routeToAgent(ctx context.Context, agentType, task string) (string, error) {
	agentPrompts := map[string]string{
		"schedule": "You are a scheduling assistant. Handle calendar and scheduling tasks.",
		"email":    "You are an email assistant. Handle email composition and management.",
		"document": "You are a document assistant. Help with writing and editing documents.",
		"analysis": "You are a data analysis assistant. Help analyze data and generate insights.",
		"approval": "You are an approval workflow assistant. Handle approval requests.",
	}

	prompt, ok := agentPrompts[agentType]
	if !ok {
		return "", fmt.Errorf("unknown agent type: %s", agentType)
	}

	return ac.aiClient.Chat(prompt, task)
}

func (ac *AgentCore) RegisterMCPClient(name string, client *MCPClient) {
	ac.mcpClients[name] = client
}

func (ac *AgentCore) RegisterChannel(name string, adapter ChannelAdapter) {
	ac.channels[name] = adapter
}

func (ac *AgentCore) GetChannel(name string) ChannelAdapter {
	return ac.channels[name]
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type SessionStore struct {
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

func (ss *SessionStore) GetOrCreateSession(sessionID, userID, channel string) (*Session, error) {
	if sessionID != "" {
		if s, ok := ss.sessions[sessionID]; ok {
			return s, nil
		}
	}

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Channel:   channel,
		Messages:  []Message{},
		Context:   make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ss.sessions[session.ID] = session
	return session, nil
}

func (ss *SessionStore) SaveSession(session *Session) {
	ss.sessions[session.ID] = session
}

func (ss *SessionStore) GetCurrentSession() *Session {
	for _, s := range ss.sessions {
		return s
	}
	return nil
}

type ChannelAdapter interface {
	SendMessage(ctx context.Context, userID, content string) error
	ReceiveMessage(ctx context.Context, msg Message) (string, error)
	GetChannelName() string
}

type BaseChannelAdapter struct {
	ChannelName string
}

func (b *BaseChannelAdapter) GetChannelName() string {
	return b.ChannelName
}

type ProcessRequestExtension struct {
	Channel     string
	UserID      string
	MessageType string
	RawMessage  string
}

func (ac *AgentCore) SetSystemPrompt(prompt string) {
	ac.config.SystemPrompt = prompt
}

func (ac *AgentCore) AddPromptTemplate(name, template string, vars []string) {
	ac.promptManager.AddTemplate(name, template, vars)
}

func (ac *AgentCore) BuildContextualPrompt(session *Session, additionalContext string) string {
	var contextStrings []string
	contextStrings = append(contextStrings, fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC3339)))

	if session != nil {
		contextStrings = append(contextStrings, fmt.Sprintf("User ID: %s", session.UserID))
		contextStrings = append(contextStrings, fmt.Sprintf("Channel: %s", session.Channel))

		if len(session.Messages) > 0 {
			recentMsgs := session.Messages
			if len(recentMsgs) > ac.config.ContextWindow {
				recentMsgs = recentMsgs[len(recentMsgs)-ac.config.ContextWindow:]
			}
			var history []string
			for _, m := range recentMsgs {
				history = append(history, fmt.Sprintf("%s: %s", m.Role, m.Content))
			}
			contextStrings = append(contextStrings, fmt.Sprintf("Conversation history:\n%s", strings.Join(history, "\n")))
		}
	}

	if additionalContext != "" {
		contextStrings = append(contextStrings, fmt.Sprintf("Additional context: %s", additionalContext))
	}

	return strings.Join(contextStrings, "\n\n")
}
