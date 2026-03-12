package agent

import (
	"context"

	"winclaw/backend/ai"
)

// GeneralAgent 通用对话Agent
type GeneralAgent struct {
	aiClient *ai.OpenAIClient
}

// NewGeneralAgent 创建通用对话Agent
func NewGeneralAgent(aiClient *ai.OpenAIClient) *GeneralAgent {
	return &GeneralAgent{
		aiClient: aiClient,
	}
}

// Process 处理通用对话请求
func (ga *GeneralAgent) Process(ctx context.Context, query string, params map[string]string) (string, error) {
	systemPrompt := `你是一个专业的办公助手，能够帮助用户处理各种办公事务。
	
你可以协助用户：
1. 回答一般性问题
2. 提供办公建议
3. 协助处理日常办公任务
4. 如果用户的问题涉及特定领域（如日程、邮件、文档等），请提示用户可以使用专门的功能`

	response, err := ga.aiClient.Chat(systemPrompt, query)
	if err != nil {
		return "", err
	}

	return response, nil
}
