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

type MCPClient struct {
	name       string
	serverURL  string
	transport  MCPTransport
	tools      []MCPTool
	resources  []MCPResource
	prompts    []MCPPrompt
	connected  bool
	mu         sync.RWMutex
	httpClient *http.Client
}

type MCPTransport interface {
	Connect(ctx context.Context) error
	CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error)
	ListTools(ctx context.Context) ([]MCPTool, error)
	ListResources(ctx context.Context) ([]MCPResource, error)
	ListPrompts(ctx context.Context) ([]MCPPrompt, error)
	Disconnect() error
}

type HTTPTransport struct {
	serverURL string
	client    *http.Client
	headers   map[string]string
}

func NewHTTPTransport(url string, headers map[string]string) *HTTPTransport {
	return &HTTPTransport{
		serverURL: strings.TrimSuffix(url, "/"),
		client:    &http.Client{Timeout: 30 * time.Second},
		headers:   headers,
	}
}

func (t *HTTPTransport) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", t.serverURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MCP server returned status: %d", resp.StatusCode)
	}
	return nil
}

func (t *HTTPTransport) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	payload := map[string]interface{}{
		"method": "tools/call",
		"params": map[string]interface{}{
			"name": name,
			"args": args,
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", t.serverURL+"/api/mcp", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if errMsg, ok := result["error"].(string); ok {
		return "", fmt.Errorf("MCP tool error: %s", errMsg)
	}

	if content, ok := result["content"].(string); ok {
		return content, nil
	}

	data, _ := json.Marshal(result["result"])
	return string(data), nil
}

func (t *HTTPTransport) ListTools(ctx context.Context) ([]MCPTool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", t.serverURL+"/api/mcp/tools", nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

func (t *HTTPTransport) ListResources(ctx context.Context) ([]MCPResource, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", t.serverURL+"/api/mcp/resources", nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Resources []MCPResource `json:"resources"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Resources, nil
}

func (t *HTTPTransport) ListPrompts(ctx context.Context) ([]MCPPrompt, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", t.serverURL+"/api/mcp/prompts", nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Prompts []MCPPrompt `json:"prompts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Prompts, nil
}

func (t *HTTPTransport) Disconnect() error {
	return nil
}

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

type MCPPrompt struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Arguments   []string `json:"arguments"`
}

func NewMCPClient(name, serverURL string, headers map[string]string) *MCPClient {
	return &MCPClient{
		name:       name,
		serverURL:  serverURL,
		transport:  NewHTTPTransport(serverURL, headers),
		tools:      []MCPTool{},
		resources:  []MCPResource{},
		prompts:    []MCPPrompt{},
		connected:  false,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *MCPClient) Connect(ctx context.Context) error {
	if err := m.transport.Connect(ctx); err != nil {
		return err
	}

	tools, err := m.transport.ListTools(ctx)
	if err == nil {
		m.tools = tools
	}

	resources, err := m.transport.ListResources(ctx)
	if err == nil {
		m.resources = resources
	}

	prompts, err := m.transport.ListPrompts(ctx)
	if err == nil {
		m.prompts = prompts
	}

	m.mu.Lock()
	m.connected = true
	m.mu.Unlock()

	return nil
}

func (m *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	m.mu.RLock()
	if !m.connected {
		m.mu.RUnlock()
		return "", fmt.Errorf("MCP client not connected")
	}
	m.mu.RUnlock()
	return m.transport.CallTool(ctx, toolName, args)
}

func (m *MCPClient) GetTools() []MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tools
}

func (m *MCPClient) GetResources() []MCPResource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resources
}

func (m *MCPClient) GetPrompts() []MCPPrompt {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.prompts
}

func (m *MCPClient) GetPrompt(name string, args map[string]string) (string, error) {
	for _, p := range m.prompts {
		if p.Name == name {
			template := fmt.Sprintf("Prompt: %s\nDescription: %s\n", p.Name, p.Description)
			for _, arg := range p.Arguments {
				if val, ok := args[arg]; ok {
					template += fmt.Sprintf("%s: %s\n", arg, val)
				}
			}
			return template, nil
		}
	}
	return "", fmt.Errorf("prompt not found: %s", name)
}

func (m *MCPClient) ReadResource(ctx context.Context, uri string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.serverURL+"/api/mcp/resources/"+uri, nil)
	if err != nil {
		return "", err
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if content, ok := result["content"].(string); ok {
		return content, nil
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

func (m *MCPClient) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return m.transport.Disconnect()
}

func (m *MCPClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

type MCPManager struct {
	clients map[string]*MCPClient
	mu      sync.RWMutex
}

func NewMCPManager() *MCPManager {
	return &MCPManager{
		clients: make(map[string]*MCPClient),
	}
}

func (m *MCPManager) RegisterClient(name, serverURL string, headers map[string]string) *MCPClient {
	m.mu.Lock()
	defer m.mu.Unlock()

	client := NewMCPClient(name, serverURL, headers)
	m.clients[name] = client
	return client
}

func (m *MCPManager) GetClient(name string) (*MCPClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[name]
	return client, ok
}

func (m *MCPManager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

func (m *MCPManager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	clients := make([]*MCPClient, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	m.mu.RUnlock()

	for _, client := range clients {
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect %s: %w", client.name, err)
		}
	}
	return nil
}

func (m *MCPManager) DisconnectAll() error {
	m.mu.RLock()
	for _, client := range m.clients {
		client.Disconnect()
	}
	m.mu.RUnlock()
	return nil
}

func (m *MCPManager) BuildMCPToolsDescription() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var descriptions []string
	for name, client := range m.clients {
		if !client.connected {
			continue
		}
		for _, tool := range client.tools {
			desc := fmt.Sprintf("[MCP:%s:%s] - %s", name, tool.Name, tool.Description)
			descriptions = append(descriptions, desc)
		}
	}
	return strings.Join(descriptions, "\n")
}
