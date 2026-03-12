package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"winclaw/backend/ai"
	"winclaw/backend/skills"
	"winclaw/backend/storage"
	"winclaw/backend/system"
)

type OpenClawCompat struct {
	agentCore    *AgentCore
	storage      *storage.DB
	executor     *system.SystemExecutor
	perm         *system.PermissionManager
	aiClient     *ai.OpenAIClient
	skillManager *skills.SkillManager
}

func NewOpenClawCompat(db *storage.DB, exec *system.SystemExecutor, perm *system.PermissionManager, skillMgr *skills.SkillManager) *OpenClawCompat {
	compat := &OpenClawCompat{
		storage:      db,
		executor:     exec,
		perm:         perm,
		skillManager: skillMgr,
	}

	cfg, _ := db.GetConfig()
	aiClient := ai.NewOpenAIClient(cfg.AI.APIKey, cfg.AI.BaseUrl, cfg.AI.Model, cfg.AI.Temperature)
	compat.aiClient = aiClient

	agentConfig := &AgentConfig{
		Name:             "WinClaw",
		Model:            cfg.AI.Model,
		APIKey:           cfg.AI.APIKey,
		BaseURL:          cfg.AI.BaseUrl,
		Temperature:      cfg.AI.Temperature,
		MaxTokens:        4096,
		SystemPrompt:     getDefaultSystemPrompt(),
		EnableMCP:        false,
		EnableChannels:   false,
		ContextWindow:    10,
		ConversationType: "general",
	}

	compat.agentCore = NewAgentCore(agentConfig, skillMgr)

	return compat
}

func (oc *OpenClawCompat) ProcessMessage(ctx context.Context, msg string, channel string) (string, error) {
	req := &ProcessRequest{
		SessionID: "",
		UserID:    "",
		Channel:   channel,
		Content:   msg,
		Metadata:  map[string]interface{}{},
	}

	resp, err := oc.agentCore.ProcessMessage(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (oc *OpenClawCompat) GetSkillSummaries() string {
	if oc.skillManager != nil {
		return oc.skillManager.GetSkillSummaries()
	}
	return ""
}

func (oc *OpenClawCompat) SendMessage(msg string) (string, error) {
	if oc.aiClient == nil {
		return "", fmt.Errorf("AI client not initialized")
	}

	skillInfo := oc.GetSkillSummaries()

	systemPrompt := `You are WinClaw, a helpful AI assistant that can help users with various tasks on your computer.

## Available Skills
When user asks about weather, use the "weather" skill with {"city":"cityname"}.
When user wants to summarize text, use the "summarize" skill with {"text":"...", "max_sentences":3}.
When user wants to read news, use the "news" skill with {"category":"top","limit":10}.
When user needs calculations, use the "calculator" skill with {"action":"calculate","expression":"..."}.

## Installed Apps
You can scan and launch installed applications:
- [SYSTEM:ScanInstalledApps:{}] - Scan all installed Windows applications
- [SYSTEM:LaunchApp:{"path":"C:\\Path\\To\\App.exe"}] - Launch a specific application
- [SYSTEM:FindExecutable:{"name":"app name"}] - Find an application's executable path

## Task Decomposition
When user wants to open an application (like Word, Excel, Notepad), use the [SYSTEM:FindExecutable] command to find and launch it.

For example:
- User: "打开Word" 
- You: [SYSTEM:FindExecutable:{"name":"Word"}]

The system will find and launch the application automatically.

If user wants to do a complex task like "打开word写一篇AI的文章", break it into steps:
[STEPS:[{"description":"打开Microsoft Word","action":"launch_app:Word"},{"description":"生成AI相关文章","action":"ai_generate:写一篇AI的文章"}]]

` + skillInfo + `

## How to call a skill
When you need to use a skill, output this format in your response:
[SKILL:name:{"param1":"value1"}]

Example: [SKILL:weather:{"city":"Beijing"}]
Example: [SKILL:calculator:{"action":"calculate","expression":"2+2"}]

The skill result will be returned and you can then respond to the user.

## System Operations
You can also directly execute system operations:
- [SYSTEM:OpenBrowser:{"url":"https://example.com"}] - Open a URL in default browser
- [SYSTEM:RunCommand:{"cmd":"notepad"}] - Run a command/program
- [SYSTEM:GetSystemInfo:{}] - Get system information
- [SYSTEM:ScanInstalledApps:{}] - Scan installed applications
- [SYSTEM:LaunchApp:{"path":"C:\\Program Files\\App\\app.exe"}] - Launch an application

Example: [SYSTEM:OpenBrowser:{"url":"https://google.com"}]
Example: [SYSTEM:RunCommand:{"cmd":"notepad.exe"}]
Example: [SYSTEM:RunCommand:{"cmd":"cmd /c dir"}]
Example: [SYSTEM:ScanInstalledApps:{}]
Example: [SYSTEM:LaunchApp:{"path":"C:\\Program Files\\Microsoft Office\\root\\Office16\\WINWORD.EXE"}]

You have access to the following capabilities:
- File operations (read, write, list)
- Run programs
- Open browsers
- System information
- Scan and launch installed applications
- Task decomposition
- Skills execution

Always be helpful and concise. If you need to perform an operation, explain what you're going to do first.`

	reply, err := oc.aiClient.Chat(systemPrompt, msg)
	if err != nil {
		return "", err
	}

	reply = oc.processSkillCalls(reply)
	reply = oc.processSystemCalls(reply)

	if oc.storage != nil {
		oc.storage.SaveMessage(&storage.Message{Role: "user", Content: msg})
		oc.storage.SaveMessage(&storage.Message{Role: "assistant", Content: reply})
	}

	return reply, nil
}

func (oc *OpenClawCompat) processSkillCalls(response string) string {
	re := regexp.MustCompile(`\[SKILL:(\w+):(.+?)\]`)
	matches := re.FindAllStringSubmatch(response, -1)

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		skillName := m[1]
		paramsStr := m[2]

		var skillParams map[string]interface{}
		if err := json.Unmarshal([]byte(paramsStr), &skillParams); err != nil {
			skillParams = map[string]interface{}{"raw": paramsStr}
		}

		result, err := oc.ExecuteSkill(skillName, skillParams)
		if err != nil {
			response = strings.Replace(response, m[0], fmt.Sprintf("Error executing skill: %v", err), 1)
		} else {
			response = strings.Replace(response, m[0], fmt.Sprintf("Result: %s", result), 1)
		}
	}

	return response
}

func (oc *OpenClawCompat) processSystemCalls(response string) string {
	re := regexp.MustCompile(`\[SYSTEM:(\w+):(.+?)\]`)
	matches := re.FindAllStringSubmatch(response, -1)

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		sysOp := m[1]
		paramsStr := m[2]

		var sysParams map[string]interface{}
		if err := json.Unmarshal([]byte(paramsStr), &sysParams); err != nil {
			sysParams = map[string]interface{}{"raw": paramsStr}
		}

		var result string
		var err error

		switch sysOp {
		case "OpenBrowser":
			url, _ := sysParams["url"].(string)
			err = oc.executor.OpenBrowser(url)
			if err != nil {
				result = fmt.Sprintf("Failed to open browser: %v", err)
			} else {
				result = "Browser opened successfully"
			}
		case "RunCommand":
			cmd, _ := sysParams["cmd"].(string)
			result, err = oc.executor.RunCommand(cmd)
		case "GetSystemInfo":
			result, err = oc.executor.GetSystemInfo()
		case "ScanInstalledApps":
			apps, err := oc.executor.ScanInstalledApps()
			if err != nil {
				result = fmt.Sprintf("Error scanning apps: %v", err)
			} else {
				result = fmt.Sprintf("Found %d installed applications", len(apps))
			}
		case "LaunchApp":
			path, _ := sysParams["path"].(string)
			err = oc.executor.LaunchApp(path)
			if err != nil {
				result = fmt.Sprintf("Failed to launch app: %v", err)
			} else {
				result = fmt.Sprintf("Application launched: %s", path)
			}
		case "FindExecutable":
			name, _ := sysParams["name"].(string)
			path, err := oc.executor.FindExecutable(name)
			if err != nil {
				result = fmt.Sprintf("Failed to find executable: %v", err)
			} else {
				result = fmt.Sprintf("Found: %s", path)
			}
		default:
			result = fmt.Sprintf("Unknown operation: %s", sysOp)
		}

		if err != nil {
			response = strings.Replace(response, m[0], fmt.Sprintf("Error: %v", err), 1)
		} else {
			response = strings.Replace(response, m[0], fmt.Sprintf("Result: %s", result), 1)
		}
	}

	return response
}

func (oc *OpenClawCompat) RegisterMCPClient(name, url string) error {
	mcpClient := NewMCPClient(name, url, nil)
	ctx := context.Background()
	return mcpClient.Connect(ctx)
}

func (oc *OpenClawCompat) RegisterChannel(name string, adapter ChannelAdapter) {
	oc.agentCore.RegisterChannel(name, adapter)
}

func (oc *OpenClawCompat) ExecuteSkill(name string, params map[string]interface{}) (string, error) {
	return oc.agentCore.skillManager.ExecuteSkill(name, params)
}

func (oc *OpenClawCompat) RunCommand(cmd string) (string, error) {
	if !oc.perm.CanExecute("run_command") {
		return "", fmt.Errorf("permission denied")
	}
	return oc.executor.RunCommand(cmd)
}

func (oc *OpenClawCompat) GetHistory(sessionID string) ([]storage.Message, error) {
	if oc.storage == nil {
		return []storage.Message{}, nil
	}
	return oc.storage.GetMessages(sessionID)
}

func (oc *OpenClawCompat) SaveMessage(role, content string) error {
	if oc.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	return oc.storage.SaveMessage(&storage.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

func (oc *OpenClawCompat) GetConfig() (map[string]interface{}, error) {
	if oc.storage == nil {
		return map[string]interface{}{}, nil
	}
	cfg, err := oc.storage.GetConfig()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"ai": map[string]interface{}{
			"model":       cfg.AI.Model,
			"api_key":     cfg.AI.APIKey,
			"base_url":    cfg.AI.BaseUrl,
			"temperature": cfg.AI.Temperature,
		},
		"permission": map[string]interface{}{
			"level":        cfg.Permission.Level,
			"confirm_mode": cfg.Permission.ConfirmMode,
		},
	}, nil
}

func (oc *OpenClawCompat) SetConfig(configMap map[string]interface{}) error {
	if oc.storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	cfg, _ := oc.storage.GetConfig()

	if ai, ok := configMap["ai"].(map[string]interface{}); ok {
		if model, ok := ai["model"].(string); ok {
			cfg.AI.Model = model
		}
		if apiKey, ok := ai["api_key"].(string); ok {
			cfg.AI.APIKey = apiKey
		}
		if baseURL, ok := ai["base_url"].(string); ok {
			cfg.AI.BaseUrl = baseURL
		}
		if temp, ok := ai["temperature"].(float64); ok {
			cfg.AI.Temperature = temp
		}
	}

	if perm, ok := configMap["permission"].(map[string]interface{}); ok {
		if level, ok := perm["level"].(string); ok {
			cfg.Permission.Level = level
		}
		if confirmMode, ok := perm["confirm_mode"].(string); ok {
			cfg.Permission.ConfirmMode = confirmMode
		}
	}

	return oc.storage.SaveConfig(cfg)
}

func (oc *OpenClawCompat) AddSkill(name string, fn skills.SkillFunc) {
	oc.agentCore.skillManager.AddSkill(name, fn)
}

func (oc *OpenClawCompat) ListSkills() []skills.Skill {
	return oc.agentCore.skillManager.ListSkills()
}

func (oc *OpenClawCompat) SetSystemPrompt(prompt string) {
	oc.agentCore.SetSystemPrompt(prompt)
}

func (oc *OpenClawCompat) AddPromptTemplate(name, template string, vars []string) {
	oc.agentCore.AddPromptTemplate(name, template, vars)
}

func (oc *OpenClawCompat) GetAIChat() func(systemPrompt, userMessage string) (string, error) {
	return oc.aiClient.Chat
}

func getDefaultSystemPrompt() string {
	return `You are WinClaw, a helpful AI assistant that can help users with various tasks on your computer.

## Available Skills
When user asks about weather, use the "weather" skill.
When user wants to summarize text, use the "summarize" skill.
When user wants to read news, use the "news" skill.
When user needs calculations, use the "calculator" skill.

## System Operations
You can execute various system operations:
- Open browsers
- Run commands
- Read/write files
- Get system information
- Scan and launch installed applications

Always be helpful and concise. If you need to perform an operation, explain what you're going to do first.`
}

type UnifiedAgent struct {
	agentCore    *AgentCore
	channelMgr   *ChannelManager
	mcpManager   *MCPManager
	compat       *OpenClawCompat
	skillManager *skills.SkillManager
	executor     *system.SystemExecutor
	perm         *system.PermissionManager
}

func NewUnifiedAgent(cfg *AgentConfig, db *storage.DB, exec *system.SystemExecutor, perm *system.PermissionManager, skillMgr *skills.SkillManager) *UnifiedAgent {
	ua := &UnifiedAgent{
		skillManager: skillMgr,
		executor:     exec,
		perm:         perm,
	}

	ua.agentCore = NewAgentCore(cfg, skillMgr)
	ua.channelMgr = NewChannelManager()
	ua.mcpManager = NewMCPManager()
	ua.compat = NewOpenClawCompat(db, exec, perm, skillMgr)

	return ua
}

func (ua *UnifiedAgent) Process(ctx context.Context, msg, channel, userID string) (string, error) {
	_, ok := ua.channelMgr.GetAdapter(channel)
	if !ok {
		return ua.compat.ProcessMessage(ctx, msg, channel)
	}

	dispatcher := NewChannelDispatcher(ua.channelMgr, ua.agentCore)
	channelMsg := &ChannelMessage{
		Channel:   channel,
		UserID:    userID,
		Content:   msg,
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{},
	}

	return dispatcher.Dispatch(ctx, channelMsg)
}

func (ua *UnifiedAgent) RegisterMCP(name, url string) error {
	client := ua.mcpManager.RegisterClient(name, url, nil)
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		return err
	}
	ua.agentCore.RegisterMCPClient(name, client)
	return nil
}

func (ua *UnifiedAgent) RegisterChannel(name string, adapter ChannelAdapter) {
	ua.channelMgr.RegisterAdapter(name, adapter)
	ua.agentCore.RegisterChannel(name, adapter)
}

func (ua *UnifiedAgent) UseLegacyMode() bool {
	return ua.compat != nil
}

func (ua *UnifiedAgent) GetAgentCore() *AgentCore {
	return ua.agentCore
}

func (ua *UnifiedAgent) GetChannelManager() *ChannelManager {
	return ua.channelMgr
}

func (ua *UnifiedAgent) GetMCPManager() *MCPManager {
	return ua.mcpManager
}

func (ua *UnifiedAgent) GetCompat() *OpenClawCompat {
	return ua.compat
}

func (ua *UnifiedAgent) SetPromptTemplate(templateType, prompt string) {
	ua.agentCore.SetSystemPrompt(prompt)
}

func (ua *UnifiedAgent) EnableChannelMode() {
	if ua.agentCore != nil {
		ua.agentCore.config.EnableChannels = true
	}
}

func (ua *UnifiedAgent) EnableMCPMode() {
	if ua.agentCore != nil {
		ua.agentCore.config.EnableMCP = true
	}
}
