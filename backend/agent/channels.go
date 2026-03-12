package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ChannelManager struct {
	adapters map[string]ChannelAdapter
	mu       sync.RWMutex
	config   ChannelConfig
}

type ChannelConfig struct {
	QQ       QQConfig
	Feishu   FeishuConfig
	Webhook  WebhookConfig
	Telegram TelegramConfig
}

type QQConfig struct {
	Enabled      bool
	BotID        string
	AccessToken  string
	Secret       string
	APIURL       string
	WebSocketURL string
	GroupID      string
}

type FeishuConfig struct {
	Enabled           bool
	AppID             string
	AppSecret         string
	VerificationToken string
	EncryptKey        string
}

type WebhookConfig struct {
	Enabled bool
	URL     string
	Secret  string
}

type TelegramConfig struct {
	Enabled  bool
	BotToken string
	APIURL   string
}

func NewChannelManager() *ChannelManager {
	return &ChannelManager{
		adapters: make(map[string]ChannelAdapter),
	}
}

func (cm *ChannelManager) RegisterAdapter(name string, adapter ChannelAdapter) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.adapters[name] = adapter
}

func (cm *ChannelManager) GetAdapter(name string) (ChannelAdapter, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	adapter, ok := cm.adapters[name]
	return adapter, ok
}

func (cm *ChannelManager) ListAdapters() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	names := make([]string, 0, len(cm.adapters))
	for name := range cm.adapters {
		names = append(names, name)
	}
	return names
}

func (cm *ChannelManager) SendToChannel(ctx context.Context, channelName, userID, content string) error {
	cm.mu.RLock()
	adapter, ok := cm.adapters[channelName]
	cm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("channel not found: %s", channelName)
	}

	return adapter.SendMessage(ctx, userID, content)
}

type QQAdapter struct {
	BaseChannelAdapter
	config     QQConfig
	httpClient *http.Client
	wsConn     *WebSocketConnection
	handler    MessageHandler
}

type MessageHandler func(ctx context.Context, msg *QQMessage) (string, error)

type QQMessage struct {
	MsgType    string `json:"msg_type"`
	RawMessage string `json:"raw_message"`
	Message    string `json:"message"`
	MessageID  string `json:"message_id"`
	UserID     string `json:"user_id"`
	ChannelID  string `json:"channel_id"`
	GroupID    string `json:"group_id"`
	Sender     QQUser `json:"sender"`
	Timestamp  int64  `json:"timestamp"`
}

type QQUser struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Card     string `json:"card"`
	Role     string `json:"role"`
}

func NewQQAdapter(config QQConfig) *QQAdapter {
	return &QQAdapter{
		config:             config,
		BaseChannelAdapter: BaseChannelAdapter{ChannelName: "qq"},
		httpClient:         &http.Client{Timeout: 30 * time.Second},
	}
}

func (qa *QQAdapter) SetHandler(handler MessageHandler) {
	qa.handler = handler
}

func (qa *QQAdapter) SendMessage(ctx context.Context, userID, content string) error {
	if !qa.config.Enabled {
		return fmt.Errorf("QQ adapter not enabled")
	}

	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}

	if qa.config.GroupID != "" {
		payload["group_id"] = qa.config.GroupID
	} else {
		payload["user_id"] = userID
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", qa.config.APIURL+"/send_msg", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+qa.config.AccessToken)

	resp, err := qa.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if status, ok := result["status"].(string); ok && status == "failed" {
		return fmt.Errorf("failed to send message: %v", result)
	}

	return nil
}

func (qa *QQAdapter) ReceiveMessage(ctx context.Context, msg Message) (string, error) {
	if qa.handler == nil {
		return "", fmt.Errorf("no message handler configured")
	}

	qqMsg := &QQMessage{
		Message:   msg.Content,
		UserID:    msg.Metadata["user_id"].(string),
		Timestamp: msg.Timestamp.Unix(),
	}

	if groupID, ok := msg.Metadata["group_id"].(string); ok {
		qqMsg.GroupID = groupID
	}

	return qa.handler(ctx, qqMsg)
}

func (qa *QQAdapter) GetChannelName() string {
	return "qq"
}

func (qa *QQAdapter) StartWebSocket(ctx context.Context) error {
	if qa.config.WebSocketURL == "" {
		return fmt.Errorf("WebSocket URL not configured")
	}

	qa.wsConn = NewWebSocketConnection(qa.config.WebSocketURL, qa.config.AccessToken)
	return qa.wsConn.Connect(ctx, func(data []byte) {
		var msg QQMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}

		if qa.handler != nil {
			qa.handler(ctx, &msg)
		}
	})
}

func (qa *QQAdapter) StopWebSocket() {
	if qa.wsConn != nil {
		qa.wsConn.Disconnect()
	}
}

type FeishuAdapter struct {
	BaseChannelAdapter
	config            FeishuConfig
	httpClient        *http.Client
	handler           FeishuMessageHandler
	appAccessToken    string
	tenantAccessToken string
}

type FeishuMessageHandler func(ctx context.Context, msg *FeishuMessage) (string, error)

type FeishuMessage struct {
	MsgType   string     `json:"msg_type"`
	MessageID string     `json:"message_id"`
	Content   string     `json:"content"`
	ChatID    string     `json:"chat_id"`
	User      FeishuUser `json:"sender"`
}

type FeishuUser struct {
	UserID    string `json:"user_id"`
	OpenID    string `json:"open_id"`
	UnionID   string `json:"union_id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type FeishuEvent struct {
	Challenge string             `json:"challenge"`
	Type      string             `json:"type"`
	Event     FeishuEventContent `json:"event"`
}

type FeishuEventContent struct {
	MsgType   string     `json:"msg_type"`
	MessageID string     `json:"message_id"`
	ChatID    string     `json:"chat_id"`
	Content   string     `json:"content"`
	Sender    FeishuUser `json:"sender"`
}

func NewFeishuAdapter(config FeishuConfig) *FeishuAdapter {
	return &FeishuAdapter{
		config:             config,
		BaseChannelAdapter: BaseChannelAdapter{ChannelName: "feishu"},
		httpClient:         &http.Client{Timeout: 30 * time.Second},
	}
}

func (fa *FeishuAdapter) SetHandler(handler FeishuMessageHandler) {
	fa.handler = handler
}

func (fa *FeishuAdapter) SendMessage(ctx context.Context, chatID, content string) error {
	if !fa.config.Enabled {
		return fmt.Errorf("Feishu adapter not enabled")
	}

	if fa.tenantAccessToken == "" {
		if err := fa.refreshAccessToken(ctx); err != nil {
			return err
		}
	}

	payload := map[string]interface{}{
		"chat_id":  chatID,
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.feishu.cn/open-apim/v3/message/send", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+fa.tenantAccessToken)

	resp, err := fa.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if code, ok := result["code"].(float64); ok && code != 0 {
		return fmt.Errorf("feishu API error: %v", result)
	}

	return nil
}

func (fa *FeishuAdapter) ReceiveMessage(ctx context.Context, msg Message) (string, error) {
	if fa.handler == nil {
		return "", fmt.Errorf("no message handler configured")
	}

	feishuMsg := &FeishuMessage{
		Content: msg.Content,
	}

	if chatID, ok := msg.Metadata["chat_id"].(string); ok {
		feishuMsg.ChatID = chatID
	}

	if userID, ok := msg.Metadata["user_id"].(string); ok {
		feishuMsg.User.OpenID = userID
	}

	return fa.handler(ctx, feishuMsg)
}

func (fa *FeishuAdapter) HandleWebhook(ctx context.Context, payload []byte) (string, error) {
	var event FeishuEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return "", err
	}

	if event.Type == "url_verification" {
		return event.Challenge, nil
	}

	if event.Type == "event_callback" && fa.handler != nil {
		msg := &FeishuMessage{
			MessageID: event.Event.MessageID,
			Content:   event.Event.Content,
			ChatID:    event.Event.ChatID,
			User:      event.Event.Sender,
		}
		return fa.handler(ctx, msg)
	}

	return "", nil
}

func (fa *FeishuAdapter) refreshAccessToken(ctx context.Context) error {
	payload := map[string]string{
		"app_id":     fa.config.AppID,
		"app_secret": fa.config.AppSecret,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.feishu.cn/open-apim/v3/tenant_access_token/internal", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := fa.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if token, ok := result["tenant_access_token"].(string); ok {
		fa.tenantAccessToken = token
		return nil
	}

	return fmt.Errorf("failed to get tenant access token: %v", result)
}

func (fa *FeishuAdapter) GetChannelName() string {
	return "feishu"
}

type WebhookAdapter struct {
	BaseChannelAdapter
	config     WebhookConfig
	httpClient *http.Client
}

func NewWebhookAdapter(config WebhookConfig) *WebhookAdapter {
	return &WebhookAdapter{
		config:             config,
		BaseChannelAdapter: BaseChannelAdapter{ChannelName: "webhook"},
		httpClient:         &http.Client{Timeout: 30 * time.Second},
	}
}

func (wa *WebhookAdapter) SendMessage(ctx context.Context, userID, content string) error {
	if !wa.config.Enabled {
		return fmt.Errorf("webhook adapter not enabled")
	}

	payload := map[string]string{
		"content": content,
		"user_id": userID,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", wa.config.URL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if wa.config.Secret != "" {
		req.Header.Set("X-Webhook-Secret", wa.config.Secret)
	}

	resp, err := wa.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

func (wa *WebhookAdapter) ReceiveMessage(ctx context.Context, msg Message) (string, error) {
	return msg.Content, nil
}

func (wa *WebhookAdapter) GetChannelName() string {
	return "webhook"
}

type TelegramAdapter struct {
	BaseChannelAdapter
	config     TelegramConfig
	httpClient *http.Client
	handler    TelegramMessageHandler
}

type TelegramMessageHandler func(ctx context.Context, msg *TelegramUpdate) (string, error)

type TelegramUpdate struct {
	UpdateID int64
	Message  TelegramMessage
}

type TelegramMessage struct {
	MessageID int64        `json:"message_id"`
	From      TelegramUser `json:"from"`
	Chat      TelegramChat `json:"chat"`
	Text      string       `json:"text"`
	Date      int64        `json:"date"`
}

type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type TelegramChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

func NewTelegramAdapter(config TelegramConfig) *TelegramAdapter {
	return &TelegramAdapter{
		config:             config,
		BaseChannelAdapter: BaseChannelAdapter{ChannelName: "telegram"},
		httpClient:         &http.Client{Timeout: 30 * time.Second},
	}
}

func (ta *TelegramAdapter) SetHandler(handler TelegramMessageHandler) {
	ta.handler = handler
}

func (ta *TelegramAdapter) SendMessage(ctx context.Context, userID, content string) error {
	if !ta.config.Enabled {
		return fmt.Errorf("Telegram adapter not enabled")
	}

	apiURL := ta.config.APIURL
	if apiURL == "" {
		apiURL = "https://api.telegram.org/bot" + ta.config.BotToken
	}

	var chatID int64
	fmt.Sscanf(userID, "%d", &chatID)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    content,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL+"/sendMessage", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ta.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result["ok"].(bool) {
		return fmt.Errorf("Telegram API error: %v", result)
	}

	return nil
}

func (ta *TelegramAdapter) ReceiveMessage(ctx context.Context, msg Message) (string, error) {
	if ta.handler == nil {
		return msg.Content, nil
	}
	update := &TelegramUpdate{
		Message: TelegramMessage{
			Text: msg.Content,
			From: TelegramUser{
				ID: 0,
			},
		},
	}
	return ta.handler(ctx, update)
}

func (ta *TelegramAdapter) GetChannelName() string {
	return "telegram"
}

func (ta *TelegramAdapter) HandleUpdate(ctx context.Context, update *TelegramUpdate) (string, error) {
	if ta.handler == nil {
		return "", fmt.Errorf("no message handler configured")
	}
	return ta.handler(ctx, update)
}

type WebSocketConnection struct {
	url       string
	token     string
	conn      *http.Client
	connected bool
	onMessage func([]byte)
}

func NewWebSocketConnection(url, token string) *WebSocketConnection {
	return &WebSocketConnection{
		url:   url,
		token: token,
		conn:  &http.Client{},
	}
}

func (ws *WebSocketConnection) Connect(ctx context.Context, onMessage func([]byte)) error {
	ws.onMessage = onMessage
	ws.connected = true
	return nil
}

func (ws *WebSocketConnection) Disconnect() {
	ws.connected = false
}

func (ws *WebSocketConnection) IsConnected() bool {
	return ws.connected
}

type UniversalChannelAdapter struct {
	Name        string
	SendFunc    func(ctx context.Context, userID, content string) error
	ReceiveFunc func(ctx context.Context, msg Message) (string, error)
}

func (u *UniversalChannelAdapter) SendMessage(ctx context.Context, userID, content string) error {
	if u.SendFunc == nil {
		return fmt.Errorf("send function not configured")
	}
	return u.SendFunc(ctx, userID, content)
}

func (u *UniversalChannelAdapter) ReceiveMessage(ctx context.Context, msg Message) (string, error) {
	if u.ReceiveFunc == nil {
		return msg.Content, nil
	}
	return u.ReceiveFunc(ctx, msg)
}

func (u *UniversalChannelAdapter) GetChannelName() string {
	return u.Name
}

func CreateCustomChannel(name string, sendFunc func(ctx context.Context, userID, content string) error, receiveFunc func(ctx context.Context, msg Message) (string, error)) ChannelAdapter {
	return &UniversalChannelAdapter{
		Name:        name,
		SendFunc:    sendFunc,
		ReceiveFunc: receiveFunc,
	}
}

type ChannelMessage struct {
	Channel   string
	UserID    string
	Content   string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

type ChannelDispatcher struct {
	manager   *ChannelManager
	agentCore *AgentCore
}

func NewChannelDispatcher(manager *ChannelManager, agentCore *AgentCore) *ChannelDispatcher {
	return &ChannelDispatcher{
		manager:   manager,
		agentCore: agentCore,
	}
}

func (cd *ChannelDispatcher) Dispatch(ctx context.Context, channelMsg *ChannelMessage) (string, error) {
	adapter, ok := cd.manager.GetAdapter(channelMsg.Channel)
	if !ok {
		return "", fmt.Errorf("channel not found: %channel")
	}

	msg := Message{
		Role:      "user",
		Content:   channelMsg.Content,
		Timestamp: channelMsg.Timestamp,
		Metadata:  channelMsg.Metadata,
	}

	reply, err := adapter.ReceiveMessage(ctx, msg)
	if err != nil {
		return "", err
	}

	req := &ProcessRequest{
		SessionID: "",
		UserID:    channelMsg.UserID,
		Channel:   channelMsg.Channel,
		Content:   reply,
		Metadata:  channelMsg.Metadata,
	}

	resp, err := cd.agentCore.ProcessMessage(ctx, req)
	if err != nil {
		return "", err
	}

	if err := adapter.SendMessage(ctx, channelMsg.UserID, resp.Content); err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (cd *ChannelDispatcher) HandleWebhook(ctx context.Context, channel string, payload []byte) (string, error) {
	var msg Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return "", err
	}

	adapter, ok := cd.manager.GetAdapter(channel)
	if !ok {
		return "", fmt.Errorf("channel not found: %s", channel)
	}

	return adapter.ReceiveMessage(ctx, msg)
}

var _ ChannelAdapter = (*QQAdapter)(nil)
var _ ChannelAdapter = (*FeishuAdapter)(nil)
var _ ChannelAdapter = (*WebhookAdapter)(nil)
var _ ChannelAdapter = (*TelegramAdapter)(nil)
var _ ChannelAdapter = (*UniversalChannelAdapter)(nil)
