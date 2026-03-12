package agent

import (
	"context"
	"fmt"
	"winclaw/backend/ai"
	"winclaw/backend/services"
	"winclaw/backend/utils"
)

// ApprovalAgent 审批流程Agent
type ApprovalAgent struct {
	aiClient *ai.OpenAIClient
	docSvc   services.DocumentService
}

// NewApprovalAgent 创建审批流程Agent
func NewApprovalAgent(aiClient *ai.OpenAIClient) *ApprovalAgent {
	return &ApprovalAgent{
		aiClient: aiClient,
		docSvc:   services.NewMockDocumentService(),
	}
}

// Process 处理审批相关请求
func (aa *ApprovalAgent) Process(ctx context.Context, query string, params map[string]string) (string, error) {
	// 根据查询内容判断用户意图
	if utils.ContainsIgnoreCase(query, "提交") || utils.ContainsIgnoreCase(query, "申请") || utils.ContainsIgnoreCase(query, "发起") ||
		utils.ContainsIgnoreCase(query, "submit") || utils.ContainsIgnoreCase(query, "request") || utils.ContainsIgnoreCase(query, "initiate") {
		return aa.submitApproval(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "审批") || utils.ContainsIgnoreCase(query, "批准") || utils.ContainsIgnoreCase(query, "同意") ||
		utils.ContainsIgnoreCase(query, "approve") || utils.ContainsIgnoreCase(query, "review") || utils.ContainsIgnoreCase(query, "consent") {
		return aa.approveRequest(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "查看") || utils.ContainsIgnoreCase(query, "查询") || utils.ContainsIgnoreCase(query, "状态") ||
		utils.ContainsIgnoreCase(query, "check") || utils.ContainsIgnoreCase(query, "status") || utils.ContainsIgnoreCase(query, "track") {
		return aa.checkStatus(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "撤回") || utils.ContainsIgnoreCase(query, "取消") || utils.ContainsIgnoreCase(query, "withdraw") ||
		utils.ContainsIgnoreCase(query, "cancel") {
		return aa.withdrawRequest(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "历史") || utils.ContainsIgnoreCase(query, "记录") || utils.ContainsIgnoreCase(query, "history") ||
		utils.ContainsIgnoreCase(query, "records") {
		return aa.getHistory(ctx, query, params)
	} else {
		// 使用AI进行更复杂的意图理解
		return aa.processWithAI(ctx, query, params)
	}
}

// submitApproval 提交审批申请
func (aa *ApprovalAgent) submitApproval(ctx context.Context, query string, params map[string]string) (string, error) {
	requestType := params["type"]
	if requestType == "" {
		// 从查询中推断申请类型
		if utils.ContainsIgnoreCase(query, "请假") || utils.ContainsIgnoreCase(query, "leave") {
			requestType = "leave"
		} else if utils.ContainsIgnoreCase(query, "报销") || utils.ContainsIgnoreCase(query, "reimbursement") {
			requestType = "expense"
		} else if utils.ContainsIgnoreCase(query, "采购") || utils.ContainsIgnoreCase(query, "purchase") {
			requestType = "procurement"
		} else {
			requestType = "general"
		}
	}

	title := params["title"]
	if title == "" {
		title = "审批申请"
	}

	result := fmt.Sprintf("审批申请已提交:\n")
	result += fmt.Sprintf("类型: %s\n", requestType)
	result += fmt.Sprintf("标题: %s\n", title)
	result += fmt.Sprintf("状态: 待审批\n")
	result += fmt.Sprintf("申请ID: APP-%d", 1000) // 简化的ID生成

	return result, nil
}

// approveRequest 审批请求
func (aa *ApprovalAgent) approveRequest(ctx context.Context, query string, params map[string]string) (string, error) {
	requestID := params["request_id"]
	if requestID == "" {
		return "请提供审批申请ID", nil
	}

	action := "approved"
	if utils.ContainsIgnoreCase(query, "拒绝") || utils.ContainsIgnoreCase(query, "驳回") || utils.ContainsIgnoreCase(query, "reject") {
		action = "rejected"
	}

	result := fmt.Sprintf("审批操作完成:\n")
	result += fmt.Sprintf("申请ID: %s\n", requestID)
	result += fmt.Sprintf("操作: %s\n", action)
	result += fmt.Sprintf("审批人: current_approver")

	return result, nil
}

// checkStatus 查询审批状态
func (aa *ApprovalAgent) checkStatus(ctx context.Context, query string, params map[string]string) (string, error) {
	requestID := params["request_id"]
	if requestID == "" {
		return "请提供审批申请ID", nil
	}

	result := fmt.Sprintf("审批状态:\n")
	result += fmt.Sprintf("申请ID: %s\n", requestID)
	result += fmt.Sprintf("状态: 待审批\n")
	result += fmt.Sprintf("当前审批人: supervisor@company.com\n")
	result += fmt.Sprintf("提交时间: %s", "2023-01-01 10:00:00")

	return result, nil
}

// withdrawRequest 撤回审批申请
func (aa *ApprovalAgent) withdrawRequest(ctx context.Context, query string, params map[string]string) (string, error) {
	requestID := params["request_id"]
	if requestID == "" {
		return "请提供审批申请ID", nil
	}

	result := fmt.Sprintf("审批申请已撤回:\n")
	result += fmt.Sprintf("申请ID: %s\n", requestID)

	return result, nil
}

// getHistory 获取审批历史
func (aa *ApprovalAgent) getHistory(ctx context.Context, query string, params map[string]string) (string, error) {
	history := []map[string]string{
		{
			"id":     "APP-1001",
			"title":  "年假申请",
			"type":   "leave",
			"status": "approved",
			"date":   "2023-01-15",
		},
		{
			"id":     "APP-1002",
			"title":  "差旅费报销",
			"type":   "expense",
			"status": "approved",
			"date":   "2023-01-20",
		},
		{
			"id":     "APP-1003",
			"title":  "办公用品采购",
			"type":   "procurement",
			"status": "pending",
			"date":   "2023-01-25",
		},
	}

	result := "审批历史记录:\n\n"
	for i, record := range history {
		result += fmt.Sprintf("%d. [%s] %s\n", i+1, record["status"], record["title"])
		result += fmt.Sprintf("   ID: %s\n", record["id"])
		result += fmt.Sprintf("   类型: %s\n", record["type"])
		result += fmt.Sprintf("   日期: %s\n\n", record["date"])
	}

	return result, nil
}

// processWithAI 使用AI处理复杂审批请求
func (aa *ApprovalAgent) processWithAI(ctx context.Context, query string, params map[string]string) (string, error) {
	systemPrompt := `你是一个专业的审批助手，能够帮助用户处理各种审批流程。
	
你可以协助用户：
1. 提交各类审批申请（请假、报销、采购等）
2. 查询审批状态
3. 处理审批操作（批准、拒绝）
4. 撤回审批申请
5. 查看审批历史记录
6. 解释审批流程和规则

请根据用户的具体需求提供相应的帮助。`

	response, err := aa.aiClient.Chat(systemPrompt, query)
	if err != nil {
		return "", err
	}

	return response, nil
}
