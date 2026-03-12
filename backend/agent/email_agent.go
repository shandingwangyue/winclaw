package agent

import (
	"context"
	"fmt"
	"strings"
	"winclaw/backend/ai"
	"winclaw/backend/services"
)

// EmailAgent 邮件处理Agent
type EmailAgent struct {
	aiClient *ai.OpenAIClient
	emailSvc services.EmailService
}

// NewEmailAgent 创建邮件处理Agent
func NewEmailAgent(aiClient *ai.OpenAIClient) *EmailAgent {
	return &EmailAgent{
		aiClient: aiClient,
		emailSvc: services.NewMockEmailService(),
	}
}

// Process 处理邮件相关请求
func (ea *EmailAgent) Process(ctx context.Context, query string, params map[string]string) (string, error) {
	// 根据查询内容判断用户意图
	if containsAny(query, []string{"发送", "发邮件", "写邮件", "新邮件", "send", "compose"}) {
		return ea.sendEmail(ctx, query, params)
	} else if containsAny(query, []string{"收件箱", "收邮件", "查看邮件", "邮件列表", "inbox", "emails", "messages"}) {
		return ea.listEmails(ctx, query, params)
	} else if containsAny(query, []string{"回复", "回邮件", "答复", "reply", "respond"}) {
		return ea.replyToEmail(ctx, query, params)
	} else if containsAny(query, []string{"转发", "转邮件", "forward"}) {
		return ea.forwardEmail(ctx, query, params)
	} else if containsAny(query, []string{"删除", "删除邮件", "remove", "delete"}) {
		return ea.deleteEmail(ctx, query, params)
	} else if containsAny(query, []string{"搜索", "查找", "search", "find"}) {
		return ea.searchEmails(ctx, query, params)
	} else {
		// 使用AI进行更复杂的意图理解
		return ea.processWithAI(ctx, query, params)
	}
}

// sendEmail 发送邮件
func (ea *EmailAgent) sendEmail(ctx context.Context, query string, params map[string]string) (string, error) {
	// 从参数中提取邮件信息
	to := params["recipients"]
	subject := params["subject"]

	if to == "" {
		return "请提供收件人地址", nil
	}

	// 创建邮件对象
	email := &services.Email{
		To:      []string{to},
		Subject: subject,
		Body:    query, // 使用整个查询作为邮件正文
	}

	// 发送邮件
	sentEmail, err := ea.emailSvc.SendEmail(ctx, email)
	if err != nil {
		return "发送邮件失败: " + err.Error(), nil
	}

	result := "邮件已成功发送:\n"
	result += "收件人: " + to + "\n"
	result += "主题: " + sentEmail.Subject + "\n"
	result += "时间: " + sentEmail.Timestamp.Format("2006-01-02 15:04:05")

	return result, nil
}

// listEmails 列出邮件
func (ea *EmailAgent) listEmails(ctx context.Context, query string, params map[string]string) (string, error) {
	emails, err := ea.emailSvc.ListEmails(ctx, "INBOX", 10, 0)
	if err != nil {
		return "获取邮件列表失败: " + err.Error(), nil
	}

	if len(emails) == 0 {
		return "收件箱中没有邮件", nil
	}

	result := "收件箱中的最新邮件:\n\n"
	for i, email := range emails {
		result += fmt.Sprintf("%d. %s\n", i+1, email.Subject)
		result += fmt.Sprintf("   发件人: %s\n", email.From)
		result += fmt.Sprintf("   时间: %s\n", email.Timestamp.Format("01/02 15:04"))

		// 显示邮件预览（截取前100个字符）
		bodyPreview := email.Body
		if len(bodyPreview) > 100 {
			bodyPreview = bodyPreview[:100] + "..."
		}
		result += fmt.Sprintf("   预览: %s\n\n", bodyPreview)
	}

	return result, nil
}

// replyToEmail 回复邮件
func (ea *EmailAgent) replyToEmail(ctx context.Context, query string, params map[string]string) (string, error) {
	// 这里需要获取原始邮件ID，简化处理
	emails, err := ea.emailSvc.ListEmails(ctx, "INBOX", 1, 0)
	if err != nil || len(emails) == 0 {
		return "没有找到可回复的邮件", nil
	}

	originalEmail := emails[0]

	reply := &services.Email{
		To:      []string{originalEmail.From},
		Subject: "回复: " + originalEmail.Subject,
		Body:    query,
	}

	replyEmail, err := ea.emailSvc.ReplyToEmail(ctx, originalEmail.ID, reply)
	if err != nil {
		return "回复邮件失败: " + err.Error(), nil
	}

	return "已回复邮件: " + replyEmail.Subject, nil
}

// forwardEmail 转发邮件
func (ea *EmailAgent) forwardEmail(ctx context.Context, query string, params map[string]string) (string, error) {
	// 这里需要获取原始邮件ID和转发目标，简化处理
	emails, err := ea.emailSvc.ListEmails(ctx, "INBOX", 1, 0)
	if err != nil || len(emails) == 0 {
		return "没有找到可转发的邮件", nil
	}

	originalEmail := emails[0]

	forwardTo := params["recipients"]
	if forwardTo == "" {
		return "请提供转发的收件人", nil
	}

	forward := &services.Email{
		To:      []string{forwardTo},
		Subject: "转发: " + originalEmail.Subject,
		Body:    originalEmail.Body + "\n\n---------- Forwarded message ---------\n" + query,
	}

	sentEmail, err := ea.emailSvc.ForwardEmail(ctx, originalEmail.ID, forward)
	if err != nil {
		return "转发邮件失败: " + err.Error(), nil
	}

	return "已转发邮件至: " + forwardTo + "，邮件ID: " + sentEmail.ID, nil
}

// deleteEmail 删除邮件
func (ea *EmailAgent) deleteEmail(ctx context.Context, query string, params map[string]string) (string, error) {
	// 简化处理，删除最新的邮件
	emails, err := ea.emailSvc.ListEmails(ctx, "INBOX", 1, 0)
	if err != nil || len(emails) == 0 {
		return "没有找到可删除的邮件", nil
	}

	emailToDelete := emails[0]

	err = ea.emailSvc.DeleteEmail(ctx, emailToDelete.ID)
	if err != nil {
		return "删除邮件失败: " + err.Error(), nil
	}

	return "已删除邮件: " + emailToDelete.Subject, nil
}

// searchEmails 搜索邮件
func (ea *EmailAgent) searchEmails(ctx context.Context, query string, params map[string]string) (string, error) {
	// 从查询中提取搜索关键词
	searchTerm := query
	if params["query"] != "" {
		searchTerm = params["query"]
	}

	emails, err := ea.emailSvc.SearchEmails(ctx, searchTerm)
	if err != nil {
		return "搜索邮件失败: " + err.Error(), nil
	}

	if len(emails) == 0 {
		return "未找到匹配的邮件", nil
	}

	result := fmt.Sprintf("找到 %d 封匹配的邮件:\n\n", len(emails))
	for i, email := range emails {
		result += fmt.Sprintf("%d. %s\n", i+1, email.Subject)
		result += fmt.Sprintf("   发件人: %s\n", email.From)
		result += fmt.Sprintf("   时间: %s\n\n", email.Timestamp.Format("01/02 15:04"))
	}

	return result, nil
}

// processWithAI 使用AI处理复杂邮件请求
func (ea *EmailAgent) processWithAI(ctx context.Context, query string, params map[string]string) (string, error) {
	systemPrompt := `你是一个专业的邮件助手，能够帮助用户处理各种邮件相关任务。
	
你可以协助用户：
1. 发送邮件
2. 查看收件箱
3. 回复邮件
4. 转发邮件
5. 撰写邮件草稿
6. 管理邮件标签和分类
7. 搜索邮件

请根据用户的具体需求提供相应的帮助。`

	response, err := ea.aiClient.Chat(systemPrompt, query)
	if err != nil {
		return "", err
	}

	return response, nil
}

// containsAny 检查字符串是否包含任意关键词
func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(text), strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}
