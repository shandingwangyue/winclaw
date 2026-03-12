package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"winclaw/backend/agent"
	"winclaw/backend/ai"
	"winclaw/backend/skills"
	"winclaw/backend/storage"
	"winclaw/backend/system"
	"winclaw/backend/voice"
)

type App struct {
	ctx            context.Context
	config         storage.Config
	db             *storage.DB
	md             *storage.MarkdownStore
	client         *ai.OpenAIClient
	perm           *system.PermissionManager
	exec           *system.SystemExecutor
	skills         *skills.SkillManager
	stt            *voice.STT
	tts            *voice.TTS
	unifiedAgent   *agent.UnifiedAgent
	channelManager *agent.ChannelManager
	mcpManager     *agent.MCPManager
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	appDir := "."

	db, err := storage.NewDB(appDir)
	if err != nil {
		fmt.Println("DB init error:", err)
		a.db = nil
		return
	}
	a.db = db

	cfg, _ := db.GetConfig()
	a.config = cfg
	a.md = storage.NewMarkdownStore(cfg.Storage.ConversationPath)

	a.client = ai.NewOpenAIClient(cfg.AI.APIKey, cfg.AI.BaseUrl, cfg.AI.Model, cfg.AI.Temperature)

	a.perm = system.NewPermissionManager()
	a.perm.SetPermission(system.Permission{
		Level:       system.PermissionLevel(cfg.Permission.Level),
		ConfirmMode: system.ConfirmMode(cfg.Permission.ConfirmMode),
	})

	a.exec = system.NewSystemExecutor(a.perm)
	a.skills = skills.NewSkillManager("skills/python")
	a.stt = voice.NewSTT(cfg.AI.APIKey)
	a.tts = voice.NewTTS(cfg.AI.APIKey)

	a.initUnifiedAgent(cfg)
}

func (a *App) initUnifiedAgent(cfg storage.Config) {
	agentConfig := &agent.AgentConfig{
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

	a.unifiedAgent = agent.NewUnifiedAgent(agentConfig, a.db, a.exec, a.perm, a.skills)
	a.channelManager = a.unifiedAgent.GetChannelManager()
	a.mcpManager = a.unifiedAgent.GetMCPManager()
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

type Message struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

func (a *App) SendMessage(msg string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("AI client not initialized")
	}

	skillInfo := ""
	if a.skills != nil {
		skillInfo = a.skills.GetSkillSummaries()
	}

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

	reply, err := a.client.Chat(systemPrompt, msg)
	if err != nil {
		return "", err
	}

	for {
		matched, skillName, skillParams, fullMatch := a.parseSkillCall(reply)
		if matched {
			result, err := a.ExecuteSkill(skillName, skillParams)
			if err != nil {
				reply = strings.Replace(reply, fullMatch, fmt.Sprintf("Error executing skill: %v", err), 1)
				break
			}
			formattedResult := a.formatSkillResult(skillName, result)
			reply = strings.Replace(reply, fullMatch, formattedResult, 1)
		} else {
			break
		}
	}

	for {
		matched, sysOp, sysParams, fullMatch := a.parseSystemCall(reply)
		if matched {
			var result string
			var err error
			switch sysOp {
			case "OpenBrowser":
				url, _ := sysParams["url"].(string)
				err = a.OpenBrowser(url)
				if err != nil {
					result = fmt.Sprintf("Failed to open browser: %v. Try using RunCommand to open: cmd /c start %s", err, url)
				} else {
					result = "Browser opened successfully"
				}
			case "RunCommand":
				cmd, _ := sysParams["cmd"].(string)
				result, err = a.RunCommand(cmd)
			case "GetSystemInfo":
				result, err = a.GetSystemInfo()
			case "ScanInstalledApps":
				apps, err := a.ScanInstalledApps()
				if err != nil {
					result = fmt.Sprintf("Error scanning apps: %v", err)
				} else {
					result = fmt.Sprintf("Found %d installed applications:\n", len(apps))
					for i, app := range apps {
						if i < 20 {
							result += fmt.Sprintf("- %s (%s)\n", app.Name, app.Vendor)
						}
					}
					if len(apps) > 20 {
						result += fmt.Sprintf("\n... and %d more", len(apps)-20)
					}
				}
			case "LaunchApp":
				path, _ := sysParams["path"].(string)
				err = a.LaunchApp(path)
				if err != nil {
					result = fmt.Sprintf("Failed to launch app: %v", err)
				} else {
					result = fmt.Sprintf("Application launched: %s", path)
				}
			case "FindExecutable":
				name, _ := sysParams["name"].(string)
				path, err := a.FindExecutable(name)
				if err != nil {
					result = fmt.Sprintf("Failed to find executable: %v", err)
				} else {
					result = fmt.Sprintf("Found: %s", path)
				}
			default:
				result = fmt.Sprintf("Unknown operation: %s", sysOp)
			}
			if err != nil {
				reply = strings.Replace(reply, fullMatch, fmt.Sprintf("Error: %v", err), 1)
				break
			}
			reply = strings.Replace(reply, fullMatch, fmt.Sprintf("Result: %s", result), 1)
		} else {
			break
		}
	}

	steps := a.parseTaskSteps(reply)
	fmt.Printf("[DEBUG] Steps found: %d, reply: %s\n", len(steps), reply)
	if len(steps) > 0 {
		reply = a.executeTaskSteps(steps, reply)
	}

	if a.db != nil {
		if err := a.db.SaveMessage(&storage.Message{
			Role:    "user",
			Content: msg,
		}); err != nil {
			fmt.Printf("Error saving user message: %v\n", err)
		}
		if err := a.db.SaveMessage(&storage.Message{
			Role:    "assistant",
			Content: reply,
		}); err != nil {
			fmt.Printf("Error saving assistant message: %v\n", err)
		}
	}

	return reply, nil
}

func (a *App) GetHistory() ([]Message, error) {
	if a.db == nil {
		return []Message{}, nil
	}
	messages, err := a.db.GetMessages("")
	if err != nil {
		return nil, err
	}
	var result []Message
	for _, m := range messages {
		result = append(result, Message{
			ID:        m.ID,
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: m.Timestamp.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

func (a *App) GetConfig() (map[string]interface{}, error) {
	if a.db == nil {
		return map[string]interface{}{
			"ai":         map[string]interface{}{"model": "gpt-4o", "apiKey": "", "baseUrl": "https://api.openai.com/v1", "temperature": 0.7},
			"voice":      map[string]interface{}{"enabled": false, "autoSpeak": false, "language": "zh-CN"},
			"permission": map[string]interface{}{"level": "medium", "confirmMode": "first"},
		}, nil
	}
	cfg, _ := a.db.GetConfig()
	m := map[string]interface{}{
		"ai": map[string]interface{}{
			"model":       cfg.AI.Model,
			"apiKey":      cfg.AI.APIKey,
			"baseUrl":     cfg.AI.BaseUrl,
			"temperature": cfg.AI.Temperature,
		},
		"voice": map[string]interface{}{
			"enabled":   cfg.Voice.Enabled,
			"autoSpeak": cfg.Voice.AutoSpeak,
			"language":  cfg.Voice.Language,
		},
		"permission": map[string]interface{}{
			"level":       cfg.Permission.Level,
			"confirmMode": cfg.Permission.ConfirmMode,
		},
	}
	return m, nil
}

func (a *App) SetConfig(configJSON string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var inputConfig map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &inputConfig); err != nil {
		return fmt.Errorf("invalid config format: %v", err)
	}

	cfg, _ := a.db.GetConfig()

	if ai, ok := inputConfig["ai"].(map[string]interface{}); ok {
		if model, ok := ai["model"].(string); ok {
			cfg.AI.Model = model
		}
		if key, ok := ai["apiKey"].(string); ok {
			cfg.AI.APIKey = key
		}
		if baseUrl, ok := ai["baseUrl"].(string); ok {
			cfg.AI.BaseUrl = baseUrl
		}
		if a.client != nil {
			a.client.SetConfig(cfg.AI.APIKey, cfg.AI.BaseUrl, cfg.AI.Model, cfg.AI.Temperature)
		}
	}

	if voice, ok := inputConfig["voice"].(map[string]interface{}); ok {
		if enabled, ok := voice["enabled"].(bool); ok {
			cfg.Voice.Enabled = enabled
		}
		if autoSpeak, ok := voice["autoSpeak"].(bool); ok {
			cfg.Voice.AutoSpeak = autoSpeak
		}
	}

	if perm, ok := inputConfig["permission"].(map[string]interface{}); ok {
		if level, ok := perm["level"].(string); ok {
			cfg.Permission.Level = level
		}
		if confirmMode, ok := perm["confirmMode"].(string); ok {
			cfg.Permission.ConfirmMode = confirmMode
		}
		if a.perm != nil {
			a.perm.SetPermission(system.Permission{
				Level:       system.PermissionLevel(cfg.Permission.Level),
				ConfirmMode: system.ConfirmMode(cfg.Permission.ConfirmMode),
			})
		}
	}

	err := a.db.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("save config failed: %v", err)
	}
	return nil
}

func (a *App) RunCommand(cmd string) (string, error) {
	if !a.perm.CanExecute("run_command") {
		return "", fmt.Errorf("permission denied")
	}
	return a.exec.RunCommand(cmd)
}

func (a *App) OpenBrowser(url string) error {
	if !a.perm.CanExecute("open_browser") {
		return fmt.Errorf("permission denied")
	}
	return a.exec.OpenBrowser(url)
}

func (a *App) ReadFile(path string) (string, error) {
	if !a.perm.CanExecute("read_file") {
		return "", fmt.Errorf("permission denied")
	}
	return a.exec.ReadFile(path)
}

func (a *App) WriteFile(path, content string) error {
	if !a.perm.CanExecute("write_file") {
		return fmt.Errorf("permission denied")
	}
	return a.exec.WriteFile(path, content)
}

func (a *App) ListDir(path string) ([]string, error) {
	if !a.perm.CanExecute("list_dir") {
		return nil, fmt.Errorf("permission denied")
	}
	return a.exec.ListDir(path)
}

func (a *App) GetSystemInfo() (string, error) {
	if !a.perm.CanExecute("system_info") {
		return "", fmt.Errorf("permission denied")
	}
	return a.exec.GetSystemInfo()
}

func (a *App) ListSkills() ([]skills.Skill, error) {
	return a.skills.ListSkills(), nil
}

func (a *App) ExecuteSkill(name string, params map[string]interface{}) (string, error) {
	return a.skills.ExecuteSkill(name, params)
}

func (a *App) Speak(text string) error {
	return a.tts.Speak(text)
}

func (a *App) SaveConversation() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	messages, err := a.db.GetMessages("")
	if err != nil {
		return fmt.Errorf("failed to get messages: %v", err)
	}
	return a.md.SaveConversation("", messages)
}

func (a *App) CheckPermission(operation string) bool {
	return a.perm.IsAllowedOperation(operation)
}

func (a *App) RegisterMCP(name, url string) error {
	if a.unifiedAgent == nil {
		return fmt.Errorf("unified agent not initialized")
	}
	return a.unifiedAgent.RegisterMCP(name, url)
}

func (a *App) GetMCPClients() []string {
	if a.mcpManager == nil {
		return []string{}
	}
	return a.mcpManager.ListClients()
}

func (a *App) RegisterQQChannel(botID, token, apiURL string) error {
	if a.channelManager == nil {
		return fmt.Errorf("channel manager not initialized")
	}
	qqConfig := agent.QQConfig{
		Enabled:     true,
		BotID:       botID,
		AccessToken: token,
		APIURL:      apiURL,
	}
	qqAdapter := agent.NewQQAdapter(qqConfig)
	a.channelManager.RegisterAdapter("qq", qqAdapter)
	return nil
}

func (a *App) RegisterFeishuChannel(appID, appSecret string) error {
	if a.channelManager == nil {
		return fmt.Errorf("channel manager not initialized")
	}
	feishuConfig := agent.FeishuConfig{
		Enabled:   true,
		AppID:     appID,
		AppSecret: appSecret,
	}
	feishuAdapter := agent.NewFeishuAdapter(feishuConfig)
	a.channelManager.RegisterAdapter("feishu", feishuAdapter)
	return nil
}

func (a *App) GetChannels() []string {
	if a.channelManager == nil {
		return []string{}
	}
	return a.channelManager.ListAdapters()
}

func (a *App) ProcessWithAgent(msg, channel, userID string) (string, error) {
	if a.unifiedAgent == nil {
		return "", fmt.Errorf("unified agent not initialized")
	}
	return a.unifiedAgent.Process(context.Background(), msg, channel, userID)
}

func (a *App) parseSkillCall(response string) (bool, string, map[string]interface{}, string) {
	re := regexp.MustCompile(`\[SKILL:(\w+):(.+?)\]`)
	matches := re.FindStringSubmatch(response)
	if len(matches) < 3 {
		return false, "", nil, ""
	}

	skillName := matches[1]
	paramsStr := matches[2]

	var skillParams map[string]interface{}
	if err := json.Unmarshal([]byte(paramsStr), &skillParams); err != nil {
		skillParams = map[string]interface{}{"raw": paramsStr}
	}

	return true, skillName, skillParams, matches[0]
}

func (a *App) parseSystemCall(response string) (bool, string, map[string]interface{}, string) {
	re := regexp.MustCompile(`\[SYSTEM:(\w+):(.+?)\]`)
	matches := re.FindStringSubmatch(response)
	if len(matches) < 3 {
		return false, "", nil, ""
	}

	sysOp := matches[1]
	paramsStr := matches[2]

	var sysParams map[string]interface{}
	if err := json.Unmarshal([]byte(paramsStr), &sysParams); err != nil {
		sysParams = map[string]interface{}{"raw": paramsStr}
	}

	return true, sysOp, sysParams, matches[0]
}

func (a *App) formatSkillResult(skillName string, rawResult string) string {
	if a.client == nil {
		return fmt.Sprintf("Skill result: %s", rawResult)
	}

	prompt := fmt.Sprintf(`You are a helpful assistant that formats skill results for users.
The user executed a skill called "%s" and received the following raw result:

Raw Result:
%s

Please format this result in a clear, user-friendly way. Use markdown formatting where appropriate (headers, bullet points, bold text, etc.).
Keep it concise but informative. If the result is JSON, format it as readable text.
If there's an error, explain it clearly to the user.
Just return the formatted result, no need for introductions like "Here is the result:".`, skillName, rawResult)

	formatted, err := a.client.Chat("You are a result formatter. Format skill outputs clearly for users using markdown.", prompt)
	if err != nil {
		return fmt.Sprintf("Skill result: %s", rawResult)
	}

	return formatted
}

func (a *App) ScanInstalledApps() ([]system.InstalledApp, error) {
	if a.exec == nil {
		return nil, fmt.Errorf("executor not initialized")
	}
	return a.exec.ScanInstalledApps()
}

func (a *App) LaunchApp(appPath string) error {
	if a.exec == nil {
		return fmt.Errorf("executor not initialized")
	}
	return a.exec.LaunchApp(appPath)
}

func (a *App) FindExecutable(appName string) (string, error) {
	if a.exec == nil {
		return "", fmt.Errorf("executor not initialized")
	}
	return a.exec.FindExecutable(appName)
}

func (a *App) ExecuteTaskSteps(task string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("AI client not initialized")
	}

	prompt := fmt.Sprintf(`Break down the following task into clear, executable steps. 
For each step, specify what action needs to be taken.
Format your response as a JSON array of steps with "description" and "action" fields.

Task: %s

Example format:
[STEPS:%s]`, task, ` [{"description":"Open Microsoft Word","action":"launch_app:Microsoft Office\\WINWORD.EXE"},{"description":"Write article about AI","action":"ai_generate:Write an article about AI"},{"description":"Paste content into Word","action":"clipboard_paste"}]`)

	reply, err := a.client.Chat("You are a task planning assistant that breaks down user requests into executable steps.", prompt)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`\[STEPS:(.+?)\]`)
	matches := re.FindStringSubmatch(reply)
	if len(matches) > 1 {
		return matches[0], nil
	}

	return reply, nil
}

type TaskStep struct {
	Description string `json:"description"`
	Action      string `json:"action"`
}

func (a *App) parseTaskSteps(response string) []TaskStep {
	re := regexp.MustCompile(`(?s)\[STEPS:(\[\{.*?\}\])\]`)
	matches := re.FindStringSubmatch(response)
	if len(matches) < 2 {
		return nil
	}

	var steps []TaskStep
	if err := json.Unmarshal([]byte(matches[1]), &steps); err != nil {
		return nil
	}
	return steps
}

func (a *App) executeTaskSteps(steps []TaskStep, response string) string {
	var results []string
	results = append(results, "Executing task steps:")

	for i, step := range steps {
		results = append(results, fmt.Sprintf("%d. %s", i+1, step.Description))

		action := step.Action
		if strings.HasPrefix(action, "launch_app:") {
			appName := strings.TrimPrefix(action, "launch_app:")
			path, err := a.FindExecutable(appName)
			if err != nil {
				results = append(results, fmt.Sprintf("   Error finding app: %v", err))
				continue
			}
			if err := a.LaunchApp(path); err != nil {
				results = append(results, fmt.Sprintf("   Error launching: %v", err))
			} else {
				results = append(results, fmt.Sprintf("   Launched: %s", path))
			}
		} else if strings.HasPrefix(action, "run:") {
			cmd := strings.TrimPrefix(action, "run:")
			result, err := a.RunCommand(cmd)
			if err != nil {
				results = append(results, fmt.Sprintf("   Error: %v", err))
			} else {
				results = append(results, fmt.Sprintf("   Result: %s", result))
			}
		} else if strings.HasPrefix(action, "ai_generate:") {
			content := strings.TrimPrefix(action, "ai_generate:")
			result, err := a.client.Chat("You are a helpful assistant. Write the requested content.", content)
			if err != nil {
				results = append(results, fmt.Sprintf("   Error generating: %v", err))
			} else {
				results = append(results, fmt.Sprintf("   Generated content: %s", result))
			}
		}
	}

	re := regexp.MustCompile(`\[STEPS:\{.*?\}\]`)
	return re.ReplaceAllString(response, strings.Join(results, "\n"))
}
