package services

import (
	"context"
	"errors"
	"time"
	"winclaw/backend/utils"
)

// Email 消息结构
type Email struct {
	ID          string    `json:"id"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	From        string    `json:"from"`
	To          []string  `json:"to"`
	Cc          []string  `json:"cc"`
	Bcc         []string  `json:"bcc"`
	Attachments []string  `json:"attachments"`
	Timestamp   time.Time `json:"timestamp"`
	Read        bool      `json:"read"`
	Folder      string    `json:"folder"`
}

// EmailService 邮件服务接口
type EmailService interface {
	SendEmail(ctx context.Context, email *Email) (*Email, error)
	GetEmail(ctx context.Context, id string) (*Email, error)
	ListEmails(ctx context.Context, folder string, limit int, offset int) ([]*Email, error)
	DeleteEmail(ctx context.Context, id string) error
	MoveEmail(ctx context.Context, id, destinationFolder string) error
	MarkAsRead(ctx context.Context, id string) error
	ReplyToEmail(ctx context.Context, originalId string, reply *Email) (*Email, error)
	ForwardEmail(ctx context.Context, originalId string, forward *Email) (*Email, error)
	CreateDraft(ctx context.Context, draft *Email) (*Email, error)
	GetFolders(ctx context.Context) ([]string, error)
	SearchEmails(ctx context.Context, query string) ([]*Email, error)
}

// MockEmailService 模拟邮件服务实现
type MockEmailService struct{}

// NewMockEmailService 创建模拟邮件服务
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{}
}

// SendEmail 发送邮件
func (mes *MockEmailService) SendEmail(ctx context.Context, email *Email) (*Email, error) {
	if email.Subject == "" || len(email.To) == 0 {
		return nil, errors.New("subject and recipient are required")
	}

	email.ID = "mock_email_" + email.Timestamp.Format("20060102_150405")
	if email.Timestamp.IsZero() {
		email.Timestamp = time.Now()
	}

	return email, nil
}

// GetEmail 获取邮件
func (mes *MockEmailService) GetEmail(ctx context.Context, id string) (*Email, error) {
	if id == "" {
		return nil, errors.New("email id is required")
	}

	// 返回模拟数据
	email := &Email{
		ID:        id,
		Subject:   "模拟邮件主题",
		Body:      "这是模拟邮件内容。",
		From:      "sender@example.com",
		To:        []string{"recipient@example.com"},
		Timestamp: time.Now().Add(-1 * time.Hour),
		Read:      false,
		Folder:    "INBOX",
	}

	return email, nil
}

// ListEmails 列出邮件
func (mes *MockEmailService) ListEmails(ctx context.Context, folder string, limit int, offset int) ([]*Email, error) {
	if limit <= 0 {
		limit = 10
	}

	emails := []*Email{
		{
			ID:        "email_1",
			Subject:   "项目进度更新",
			Body:      "这是我们本周的项目进度更新...",
			From:      "project-manager@company.com",
			To:        []string{"team@company.com"},
			Timestamp: time.Now().Add(-2 * time.Hour),
			Read:      false,
			Folder:    "INBOX",
		},
		{
			ID:        "email_2",
			Subject:   "会议邀请：产品评审",
			Body:      "邀请您参加明天的产品评审会议...",
			From:      "admin@company.com",
			To:        []string{"you@company.com"},
			Timestamp: time.Now().Add(-24 * time.Hour),
			Read:      true,
			Folder:    "INBOX",
		},
		{
			ID:        "email_3",
			Subject:   "Q4财务报告",
			Body:      "请查收第四季度财务报告...",
			From:      "finance@company.com",
			To:        []string{"managers@company.com"},
			Timestamp: time.Now().Add(-48 * time.Hour),
			Read:      true,
			Folder:    "INBOX",
		},
	}

	// 应用分页
	start := offset
	if start >= len(emails) {
		return []*Email{}, nil
	}

	end := start + limit
	if end > len(emails) {
		end = len(emails)
	}

	return emails[start:end], nil
}

// DeleteEmail 删除邮件
func (mes *MockEmailService) DeleteEmail(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("email id is required")
	}

	return nil
}

// MoveEmail 移动邮件
func (mes *MockEmailService) MoveEmail(ctx context.Context, id, destinationFolder string) error {
	if id == "" || destinationFolder == "" {
		return errors.New("email id and destination folder are required")
	}

	return nil
}

// MarkAsRead 标记为已读
func (mes *MockEmailService) MarkAsRead(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("email id is required")
	}

	return nil
}

// ReplyToEmail 回复邮件
func (mes *MockEmailService) ReplyToEmail(ctx context.Context, originalId string, reply *Email) (*Email, error) {
	if originalId == "" {
		return nil, errors.New("original email id is required")
	}

	reply.ID = "reply_" + originalId
	reply.Timestamp = time.Now()

	return reply, nil
}

// ForwardEmail 转发邮件
func (mes *MockEmailService) ForwardEmail(ctx context.Context, originalId string, forward *Email) (*Email, error) {
	if originalId == "" {
		return nil, errors.New("original email id is required")
	}

	forward.ID = "forward_" + originalId
	forward.Timestamp = time.Now()

	return forward, nil
}

// CreateDraft 创建草稿
func (mes *MockEmailService) CreateDraft(ctx context.Context, draft *Email) (*Email, error) {
	draft.ID = "draft_" + time.Now().Format("20060102_150405")
	draft.Timestamp = time.Now()
	draft.Folder = "DRAFTS"

	return draft, nil
}

// GetFolders 获取文件夹列表
func (mes *MockEmailService) GetFolders(ctx context.Context) ([]string, error) {
	return []string{"INBOX", "SENT", "DRAFTS", "TRASH", "ARCHIVE"}, nil
}

// SearchEmails 搜索邮件
func (mes *MockEmailService) SearchEmails(ctx context.Context, query string) ([]*Email, error) {
	// 返回匹配搜索词的邮件
	allEmails, _ := mes.ListEmails(ctx, "INBOX", 100, 0)

	var results []*Email
	for _, email := range allEmails {
		if utils.ContainsIgnoreCase(email.Subject, query) || utils.ContainsIgnoreCase(email.Body, query) {
			results = append(results, email)
		}
	}

	return results, nil
}

// End of file
