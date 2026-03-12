package agent

import (
	"context"
	"regexp"
	"strings"

	"winclaw/backend/ai"
	"winclaw/backend/utils"
)

// IntentType 定义意图类型
type IntentType string

const (
	IntentSchedule IntentType = "schedule" // 日程管理
	IntentEmail    IntentType = "email"    // 邮件处理
	IntentDocument IntentType = "document" // 文档协作
	IntentApproval IntentType = "approval" // 审批流程
	IntentAnalysis IntentType = "analysis" // 数据分析
	IntentGeneral  IntentType = "general"  // 通用对话
	IntentUnknown  IntentType = "unknown"  // 未知意图
)

// IntentRecognitionResult 意图识别结果
type IntentRecognitionResult struct {
	Type       IntentType        `json:"type"`
	Confidence float64           `json:"confidence"`
	Parameters map[string]string `json:"parameters"`
	RawQuery   string            `json:"raw_query"`
}

// AgentRouter Agent调度器
type AgentRouter struct {
	aiClient      *ai.OpenAIClient
	scheduleAgent *ScheduleAgent
	emailAgent    *EmailAgent
	documentAgent *DocumentAgent
	approvalAgent *ApprovalAgent
	analysisAgent *AnalysisAgent
	generalAgent  *GeneralAgent
}

// NewAgentRouter 创建新的Agent调度器
func NewAgentRouter(aiClient *ai.OpenAIClient) *AgentRouter {
	router := &AgentRouter{
		aiClient: aiClient,
	}

	// 初始化各个专业Agent
	router.scheduleAgent = NewScheduleAgent(aiClient)
	router.emailAgent = NewEmailAgent(aiClient)
	router.documentAgent = NewDocumentAgent(aiClient)
	router.approvalAgent = NewApprovalAgent(aiClient)
	router.analysisAgent = NewAnalysisAgent(aiClient)
	router.generalAgent = NewGeneralAgent(aiClient)

	return router
}

// Route 根据用户输入路由到相应的Agent
func (r *AgentRouter) Route(ctx context.Context, userInput string) (string, error) {
	intent := r.RecognizeIntent(userInput)

	switch intent.Type {
	case IntentSchedule:
		return r.scheduleAgent.Process(ctx, userInput, intent.Parameters)
	case IntentEmail:
		return r.emailAgent.Process(ctx, userInput, intent.Parameters)
	case IntentDocument:
		return r.documentAgent.Process(ctx, userInput, intent.Parameters)
	case IntentApproval:
		return r.approvalAgent.Process(ctx, userInput, intent.Parameters)
	case IntentAnalysis:
		return r.analysisAgent.Process(ctx, userInput, intent.Parameters)
	case IntentGeneral:
		return r.generalAgent.Process(ctx, userInput, intent.Parameters)
	default:
		return r.generalAgent.Process(ctx, userInput, intent.Parameters)
	}
}

// RecognizeIntent 识别用户意图
func (r *AgentRouter) RecognizeIntent(query string) IntentRecognitionResult {
	queryLower := strings.ToLower(query)

	// 日程管理意图识别
	if r.containsAny(queryLower, []string{
		"安排", "预约", "会议", "日程", "时间", "提醒", "日历", "今天", "明天", "下周", "几点", "几点钟",
		"schedule", "meeting", "calendar", "appointment", "remind", "today", "tomorrow", "next week",
	}) {
		params := r.extractTimeAndPeople(query)
		return IntentRecognitionResult{
			Type:       IntentSchedule,
			Confidence: 0.9,
			Parameters: params,
			RawQuery:   query,
		}
	}

	// 邮件处理意图识别
	if r.containsAny(queryLower, []string{
		"邮件", "发邮件", "收件箱", "发信", "抄送", "密送", "回复", "转发", "附件", "收邮件",
		"email", "mail", "inbox", "send", "reply", "forward", "attachment", "cc", "bcc",
	}) {
		params := r.extractEmailInfo(query)
		return IntentRecognitionResult{
			Type:       IntentEmail,
			Confidence: 0.9,
			Parameters: params,
			RawQuery:   query,
		}
	}

	// 文档协作意图识别
	if r.containsAny(queryLower, []string{
		"文档", "写", "编辑", "创建", "word", "pdf", "表格", "excel", "ppt", "报告", "总结", "大纲",
		"document", "write", "edit", "create", "word", "pdf", "sheet", "excel", "report", "summary", "outline",
	}) {
		params := r.extractDocInfo(query)
		return IntentRecognitionResult{
			Type:       IntentDocument,
			Confidence: 0.85,
			Parameters: params,
			RawQuery:   query,
		}
	}

	// 审批流程意图识别
	if r.containsAny(queryLower, []string{
		"审批", "申请", "请假", "报销", "流程", "同意", "拒绝", "批准", "提交", "审核",
		"approve", "request", "leave", "reimburse", "process", "submit", "review",
	}) {
		params := r.extractApprovalInfo(query)
		return IntentRecognitionResult{
			Type:       IntentApproval,
			Confidence: 0.85,
			Parameters: params,
			RawQuery:   query,
		}
	}

	// 数据分析意图识别
	if r.containsAny(queryLower, []string{
		"统计", "分析", "图表", "报表", "数据", "趋势", "增长", "下降", "同比", "环比", "销量", "收入",
		"analyze", "chart", "report", "data", "trend", "sales", "revenue", "growth",
	}) {
		params := r.extractAnalysisInfo(query)
		return IntentRecognitionResult{
			Type:       IntentAnalysis,
			Confidence: 0.8,
			Parameters: params,
			RawQuery:   query,
		}
	}

	// 默认为通用对话
	return IntentRecognitionResult{
		Type:       IntentGeneral,
		Confidence: 0.7,
		Parameters: map[string]string{"query": query},
		RawQuery:   query,
	}
}

// containsAny 检查字符串是否包含任意关键词
func (r *AgentRouter) containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if utils.ContainsIgnoreCase(text, keyword) {
			return true
		}
	}
	return false
}

// extractTimeAndPeople 提取时间和人员信息
func (r *AgentRouter) extractTimeAndPeople(query string) map[string]string {
	params := make(map[string]string)

	// 提取时间信息
	timePattern := regexp.MustCompile(`(\d{1,2}[点时:]\d{2}|\d{1,2}[点时]|今天|明天|后天|[周星期]\d|下周一|下周二|下周三|下周四|下周五|下周六|下周日|这周|下周|本月|下月)`)
	timeMatches := timePattern.FindAllString(query, -1)
	if len(timeMatches) > 0 {
		params["time"] = strings.Join(timeMatches, ", ")
	}

	// 提取人员信息
	peoplePattern := regexp.MustCompile(`(跟|和|与)([^，。！？\s]+)(讨论|开会|会议|聊天|沟通|交流|约|见面)`)
	peopleMatches := peoplePattern.FindStringSubmatch(query)
	if len(peopleMatches) > 2 {
		params["people"] = peopleMatches[2]
	}

	params["query"] = query
	return params
}

// extractEmailInfo 提取邮件相关信息
func (r *AgentRouter) extractEmailInfo(query string) map[string]string {
	params := make(map[string]string)

	// 提取收件人
	emailPattern := regexp.MustCompile(`([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	emailMatches := emailPattern.FindAllString(query, -1)
	if len(emailMatches) > 0 {
		params["recipients"] = strings.Join(emailMatches, ", ")
	}

	// 提取主题
	subjectPattern := regexp.MustCompile(`(主题|标题|subject)[:：]\s*([^，。！？\n]+)`)
	subjectMatches := subjectPattern.FindStringSubmatch(query)
	if len(subjectMatches) > 2 {
		params["subject"] = subjectMatches[2]
	}

	params["query"] = query
	return params
}

// extractDocInfo 提取文档相关信息
func (r *AgentRouter) extractDocInfo(query string) map[string]string {
	params := make(map[string]string)

	// 提取文档类型
	docTypes := []string{"word", "excel", "ppt", "pdf", "文档", "表格", "报告", "总结", "大纲", "笔记"}
	for _, docType := range docTypes {
		if strings.Contains(strings.ToLower(query), docType) {
			params["doc_type"] = docType
			break
		}
	}

	params["query"] = query
	return params
}

// extractApprovalInfo 提取审批相关信息
func (r *AgentRouter) extractApprovalInfo(query string) map[string]string {
	params := make(map[string]string)

	// 提取审批类型
	approvalTypes := []string{"请假", "报销", "采购", "出差", "加班", "外出", "申请", "审批"}
	for _, approvalType := range approvalTypes {
		if strings.Contains(query, approvalType) {
			params["type"] = approvalType
			break
		}
	}

	params["query"] = query
	return params
}

// extractAnalysisInfo 提取分析相关信息
func (r *AgentRouter) extractAnalysisInfo(query string) map[string]string {
	params := make(map[string]string)

	// 提取分析指标
	indicators := []string{"销售额", "收入", "利润", "成本", "费用", "销量", "用户数", "转化率", "增长率"}
	for _, indicator := range indicators {
		if strings.Contains(query, indicator) {
			params["indicator"] = indicator
			break
		}
	}

	params["query"] = query
	return params
}

// Process 处理用户请求
func (r *AgentRouter) Process(ctx context.Context, userInput string) (string, error) {
	return r.Route(ctx, userInput)
}
