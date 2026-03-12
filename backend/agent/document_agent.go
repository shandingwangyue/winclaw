package agent

import (
	"context"
	"fmt"
	"winclaw/backend/ai"
	"winclaw/backend/services"
	"winclaw/backend/utils"
)

// DocumentAgent 文档协作Agent
type DocumentAgent struct {
	aiClient *ai.OpenAIClient
	docSvc   services.DocumentService
}

// NewDocumentAgent 创建文档协作Agent
func NewDocumentAgent(aiClient *ai.OpenAIClient) *DocumentAgent {
	return &DocumentAgent{
		aiClient: aiClient,
		docSvc:   services.NewMockDocumentService(),
	}
}

// Process 处理文档相关请求
func (da *DocumentAgent) Process(ctx context.Context, query string, params map[string]string) (string, error) {
	// 根据查询内容判断用户意图
	if utils.ContainsIgnoreCase(query, "创建") || utils.ContainsIgnoreCase(query, "新建") || utils.ContainsIgnoreCase(query, "写") || utils.ContainsIgnoreCase(query, "撰写") || utils.ContainsIgnoreCase(query, "制作") || utils.ContainsIgnoreCase(query, "create") || utils.ContainsIgnoreCase(query, "new") || utils.ContainsIgnoreCase(query, "write") || utils.ContainsIgnoreCase(query, "make") {
		return da.createDocument(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "搜索") || utils.ContainsIgnoreCase(query, "查找") || utils.ContainsIgnoreCase(query, "找") || utils.ContainsIgnoreCase(query, "search") || utils.ContainsIgnoreCase(query, "find") || utils.ContainsIgnoreCase(query, "lookup") {
		return da.searchDocuments(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "list") || utils.ContainsIgnoreCase(query, "查看") || utils.ContainsIgnoreCase(query, "显示") || utils.ContainsIgnoreCase(query, "展示") || utils.ContainsIgnoreCase(query, "show") || utils.ContainsIgnoreCase(query, "browse") || utils.ContainsIgnoreCase(query, "documents") || utils.ContainsIgnoreCase(query, "files") {
		return da.listDocuments(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "分享") || utils.ContainsIgnoreCase(query, "共享") || utils.ContainsIgnoreCase(query, "分享链接") || utils.ContainsIgnoreCase(query, "share") || utils.ContainsIgnoreCase(query, "publish") {
		return da.shareDocument(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "导入") || utils.ContainsIgnoreCase(query, "上传") || utils.ContainsIgnoreCase(query, "import") || utils.ContainsIgnoreCase(query, "upload") {
		return da.importDocument(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "导出") || utils.ContainsIgnoreCase(query, "下载") || utils.ContainsIgnoreCase(query, "export") || utils.ContainsIgnoreCase(query, "download") {
		return da.exportDocument(ctx, query, params)
	} else {
		// 使用AI进行更复杂的意图理解
		return da.processWithAI(ctx, query, params)
	}
}

// createDocument 创建文档
func (da *DocumentAgent) createDocument(ctx context.Context, query string, params map[string]string) (string, error) {
	title := params["title"]
	if title == "" {
		// 从查询中提取标题
		title = "新文档"
	}

	docType := params["doc_type"]
	if docType == "" {
		docType = "word" // 默认类型
	}

	// 创建文档对象
	doc := &services.Document{
		Title:   title,
		Content: query, // 使用查询作为初始内容
		Type:    docType,
	}

	createdDoc, err := da.docSvc.CreateDocument(ctx, doc)
	if err != nil {
		return "创建文档失败: " + err.Error(), nil
	}

	result := fmt.Sprintf("文档已创建成功:\n")
	result += fmt.Sprintf("标题: %s\n", createdDoc.Title)
	result += fmt.Sprintf("类型: %s\n", createdDoc.Type)
	result += fmt.Sprintf("ID: %s\n", createdDoc.ID)
	result += fmt.Sprintf("创建时间: %s", createdDoc.CreatedAt.Format("2006-01-02 15:04:05"))

	return result, nil
}

// searchDocuments 搜索文档
func (da *DocumentAgent) searchDocuments(ctx context.Context, query string, params map[string]string) (string, error) {
	// 从查询中提取搜索关键词
	searchTerm := query
	if params["query"] != "" {
		searchTerm = params["query"]
	}

	docs, err := da.docSvc.SearchDocuments(ctx, searchTerm)
	if err != nil {
		return "搜索文档失败: " + err.Error(), nil
	}

	if len(docs) == 0 {
		return "未找到匹配的文档", nil
	}

	result := fmt.Sprintf("找到 %d 个匹配的文档:\n\n", len(docs))
	for i, doc := range docs {
		result += fmt.Sprintf("%d. %s\n", i+1, doc.Title)
		result += fmt.Sprintf("   类型: %s\n", doc.Type)
		result += fmt.Sprintf("   作者: %s\n", doc.Author)
		result += fmt.Sprintf("   更新时间: %s\n\n", doc.UpdatedAt.Format("01/02 15:04"))
	}

	return result, nil
}

// listDocuments 列出文档
func (da *DocumentAgent) listDocuments(ctx context.Context, query string, params map[string]string) (string, error) {
	docs, err := da.docSvc.ListDocuments(ctx, "", 10, 0)
	if err != nil {
		return "获取文档列表失败: " + err.Error(), nil
	}

	if len(docs) == 0 {
		return "没有找到文档", nil
	}

	result := "文档列表:\n\n"
	for i, doc := range docs {
		result += fmt.Sprintf("%d. %s\n", i+1, doc.Title)
		result += fmt.Sprintf("   类型: %s\n", doc.Type)
		result += fmt.Sprintf("   作者: %s\n", doc.Author)
		result += fmt.Sprintf("   更新时间: %s\n\n", doc.UpdatedAt.Format("01/02 15:04"))
	}

	return result, nil
}

// shareDocument 分享文档
func (da *DocumentAgent) shareDocument(ctx context.Context, query string, params map[string]string) (string, error) {
	// 简化处理，分享第一个文档
	docs, err := da.docSvc.ListDocuments(ctx, "", 1, 0)
	if err != nil || len(docs) == 0 {
		return "没有找到可分享的文档", nil
	}

	docToShare := docs[0]

	shareLink, err := da.docSvc.ShareDocument(ctx, docToShare.ID, "read")
	if err != nil {
		return "分享文档失败: " + err.Error(), nil
	}

	return fmt.Sprintf("文档 '%s' 已分享，链接: %s", docToShare.Title, shareLink), nil
}

// importDocument 导入文档
func (da *DocumentAgent) importDocument(ctx context.Context, query string, params map[string]string) (string, error) {
	title := params["title"]
	if title == "" {
		title = "导入的文档"
	}

	docType := params["doc_type"]
	if docType == "" {
		docType = "txt"
	}

	// 模拟导入内容
	content := []byte("这是导入的文档内容")

	importedDoc, err := da.docSvc.ImportDocument(ctx, content, title, docType)
	if err != nil {
		return "导入文档失败: " + err.Error(), nil
	}

	return fmt.Sprintf("文档已导入成功: %s (ID: %s)", importedDoc.Title, importedDoc.ID), nil
}

// exportDocument 导出文档
func (da *DocumentAgent) exportDocument(ctx context.Context, query string, params map[string]string) (string, error) {
	// 简化处理，导出第一个文档
	docs, err := da.docSvc.ListDocuments(ctx, "", 1, 0)
	if err != nil || len(docs) == 0 {
		return "没有找到可导出的文档", nil
	}

	docToExport := docs[0]

	_, err = da.docSvc.ExportDocument(ctx, docToExport.ID, "pdf")
	if err != nil {
		return "导出文档失败: " + err.Error(), nil
	}

	return fmt.Sprintf("文档 '%s' 已导出成功", docToExport.Title), nil
}

// processWithAI 使用AI处理复杂文档请求
func (da *DocumentAgent) processWithAI(ctx context.Context, query string, params map[string]string) (string, error) {
	systemPrompt := `你是一个专业的文档助手，能够帮助用户处理各种文档相关任务。
	
你可以协助用户：
1. 创建各类文档（Word、Excel、PPT、PDF等）
2. 编辑和格式化文档
3. 提供文档结构建议
4. 撰写报告、总结、大纲等
5. 文档模板推荐
6. 多文档内容整合
7. 搜索和管理现有文档

请根据用户的具体需求提供相应的帮助。`

	response, err := da.aiClient.Chat(systemPrompt, query)
	if err != nil {
		return "", err
	}

	return response, nil
}

// End of file
