package services

import (
	"context"
	"errors"
	"time"
	"winclaw/backend/utils"
)

// Document 文档结构
type Document struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Type        string    `json:"type"` // word, excel, ppt, pdf, txt, etc.
	Author      string    `json:"author"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Size        int64     `json:"size"`
	Version     int       `json:"version"`
	Permissions string    `json:"permissions"` // read, write, share
	ShareLink   string    `json:"share_link"`
}

// DocumentService 文档服务接口
type DocumentService interface {
	CreateDocument(ctx context.Context, doc *Document) (*Document, error)
	GetDocument(ctx context.Context, id string) (*Document, error)
	UpdateDocument(ctx context.Context, doc *Document) (*Document, error)
	DeleteDocument(ctx context.Context, id string) error
	ListDocuments(ctx context.Context, folder string, limit int, offset int) ([]*Document, error)
	SearchDocuments(ctx context.Context, query string) ([]*Document, error)
	ShareDocument(ctx context.Context, id string, permissions string) (string, error)
	AddComment(ctx context.Context, docId string, comment string, author string) error
	GetComments(ctx context.Context, docId string) ([]Comment, error)
	ExportDocument(ctx context.Context, id string, format string) ([]byte, error)
	ImportDocument(ctx context.Context, content []byte, title, docType string) (*Document, error)
}

// Comment 评论结构
type Comment struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	Author     string    `json:"author"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// MockDocumentService 模拟文档服务实现
type MockDocumentService struct{}

// NewMockDocumentService 创建模拟文档服务
func NewMockDocumentService() *MockDocumentService {
	return &MockDocumentService{}
}

// CreateDocument 创建文档
func (mds *MockDocumentService) CreateDocument(ctx context.Context, doc *Document) (*Document, error) {
	if doc.Title == "" {
		return nil, errors.New("document title is required")
	}

	doc.ID = "doc_" + doc.Title
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	doc.Version = 1

	return doc, nil
}

// GetDocument 获取文档
func (mds *MockDocumentService) GetDocument(ctx context.Context, id string) (*Document, error) {
	if id == "" {
		return nil, errors.New("document id is required")
	}

	// 返回模拟数据
	doc := &Document{
		ID:        id,
		Title:     "模拟文档",
		Content:   "这是文档的模拟内容。",
		Type:      "word",
		Author:    "user@example.com",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
		Size:      1024,
		Version:   1,
	}

	return doc, nil
}

// UpdateDocument 更新文档
func (mds *MockDocumentService) UpdateDocument(ctx context.Context, doc *Document) (*Document, error) {
	if doc.ID == "" {
		return nil, errors.New("document id is required")
	}

	doc.UpdatedAt = time.Now()
	doc.Version++

	return doc, nil
}

// DeleteDocument 删除文档
func (mds *MockDocumentService) DeleteDocument(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("document id is required")
	}

	return nil
}

// ListDocuments 列出文档
func (mds *MockDocumentService) ListDocuments(ctx context.Context, folder string, limit int, offset int) ([]*Document, error) {
	if limit <= 0 {
		limit = 10
	}

	docs := []*Document{
		{
			ID:        "doc_1",
			Title:     "项目计划书",
			Content:   "这是项目计划书的内容...",
			Type:      "word",
			Author:    "project-manager@company.com",
			CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
			UpdatedAt: time.Now().Add(-2 * 24 * time.Hour),
			Size:      2048,
			Version:   3,
		},
		{
			ID:        "doc_2",
			Title:     "销售报告",
			Content:   "这是销售报告的内容...",
			Type:      "excel",
			Author:    "sales@company.com",
			CreatedAt: time.Now().Add(-3 * 24 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * 24 * time.Hour),
			Size:      4096,
			Version:   2,
		},
		{
			ID:        "doc_3",
			Title:     "产品设计文档",
			Content:   "这是产品设计文档的内容...",
			Type:      "ppt",
			Author:    "designer@company.com",
			CreatedAt: time.Now().Add(-1 * 24 * time.Hour),
			UpdatedAt: time.Now(),
			Size:      8192,
			Version:   1,
		},
	}

	// 应用分页
	start := offset
	if start >= len(docs) {
		return []*Document{}, nil
	}

	end := start + limit
	if end > len(docs) {
		end = len(docs)
	}

	return docs[start:end], nil
}

// ShareDocument 分享文档
func (mds *MockDocumentService) ShareDocument(ctx context.Context, id string, permissions string) (string, error) {
	if id == "" {
		return "", errors.New("document id is required")
	}

	shareLink := "https://example.com/share/" + id
	return shareLink, nil
}

// AddComment 添加评论
func (mds *MockDocumentService) AddComment(ctx context.Context, docId string, comment string, author string) error {
	if docId == "" || comment == "" {
		return errors.New("document id and comment are required")
	}

	return nil
}

// GetComments 获取评论
func (mds *MockDocumentService) GetComments(ctx context.Context, docId string) ([]Comment, error) {
	if docId == "" {
		return nil, errors.New("document id is required")
	}

	comments := []Comment{
		{
			ID:         "comment_1",
			DocumentID: docId,
			Author:     "reviewer@company.com",
			Content:    "文档内容很好，有一些小建议...",
			CreatedAt:  time.Now().Add(-12 * time.Hour),
		},
		{
			ID:         "comment_2",
			DocumentID: docId,
			Author:     "manager@company.com",
			Content:    "同意发布",
			CreatedAt:  time.Now().Add(-6 * time.Hour),
		},
	}

	return comments, nil
}

// ExportDocument 导出文档
func (mds *MockDocumentService) ExportDocument(ctx context.Context, id string, format string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("document id is required")
	}

	content := "这是导出的文档内容"
	return []byte(content), nil
}

// ImportDocument 导入文档
func (mds *MockDocumentService) ImportDocument(ctx context.Context, content []byte, title, docType string) (*Document, error) {
	if len(content) == 0 {
		return nil, errors.New("content is required")
	}

	doc := &Document{
		ID:        "imported_" + title,
		Title:     title,
		Content:   string(content),
		Type:      docType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}

	return doc, nil
}

// SearchDocuments 搜索文档
func (mds *MockDocumentService) SearchDocuments(ctx context.Context, query string) ([]*Document, error) {
	allDocs, _ := mds.ListDocuments(ctx, "", 100, 0)

	var results []*Document
	for _, doc := range allDocs {
		if utils.ContainsIgnoreCase(doc.Title, query) || utils.ContainsIgnoreCase(doc.Content, query) {
			results = append(results, doc)
		}
	}

	return results, nil
}
