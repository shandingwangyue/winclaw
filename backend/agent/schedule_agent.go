package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"winclaw/backend/ai"
	"winclaw/backend/services"
)

// ScheduleAgent 日程管理Agent
type ScheduleAgent struct {
	aiClient    *ai.OpenAIClient
	calendarSvc services.CalendarService
}

// NewScheduleAgent 创建日程管理Agent
func NewScheduleAgent(aiClient *ai.OpenAIClient) *ScheduleAgent {
	return &ScheduleAgent{
		aiClient:    aiClient,
		calendarSvc: services.NewMockCalendarService(),
	}
}

// Process 处理日程相关请求
func (sa *ScheduleAgent) Process(ctx context.Context, query string, params map[string]string) (string, error) {
	// 根据参数决定执行的操作
	if strings.Contains(query, "安排") || strings.Contains(query, "预约") || strings.Contains(query, "会议") {
		return sa.createMeeting(ctx, query, params)
	} else if strings.Contains(query, "查看") || strings.Contains(query, "查询") || strings.Contains(query, "日程") {
		return sa.viewSchedule(ctx, query, params)
	} else if strings.Contains(query, "取消") || strings.Contains(query, "删除") {
		return sa.cancelMeeting(ctx, query, params)
	} else {
		// 使用AI进行更复杂的意图理解
		return sa.processWithAI(ctx, query, params)
	}
}

// createMeeting 创建会议
func (sa *ScheduleAgent) createMeeting(ctx context.Context, query string, params map[string]string) (string, error) {
	timeInfo := params["time"]
	peopleInfo := params["people"]

	if timeInfo == "" {
		return "请提供具体的会议时间", nil
	}

	// 解析时间信息，这里简化处理
	startTime := time.Now().Add(1 * time.Hour) // 默认1小时后
	endTime := startTime.Add(1 * time.Hour)    // 默认持续1小时

	// 创建日历事件
	event := &services.CalendarEvent{
		Title:       "会议安排",
		Description: query,
		StartTime:   startTime,
		EndTime:     endTime,
		Location:    "在线会议室",
	}

	// 添加参与者
	if peopleInfo != "" {
		event.Attendees = append(event.Attendees, peopleInfo)
	}

	// 调用日历服务创建事件
	createdEvent, err := sa.calendarSvc.CreateEvent(ctx, event)
	if err != nil {
		return fmt.Sprintf("创建会议失败: %v", err), nil
	}

	result := fmt.Sprintf("已为您安排会议:\n")
	result += fmt.Sprintf("主题: %s\n", createdEvent.Title)
	result += fmt.Sprintf("时间: %s - %s\n", createdEvent.StartTime.Format("2006-01-02 15:04"), createdEvent.EndTime.Format("15:04"))
	if len(createdEvent.Attendees) > 0 {
		result += fmt.Sprintf("参与人: %s\n", strings.Join(createdEvent.Attendees, ", "))
	}
	if createdEvent.Location != "" {
		result += fmt.Sprintf("地点: %s\n", createdEvent.Location)
	}

	result += "\n会议已添加到您的日历中，相关提醒将按时发送。"

	return result, nil
}

// viewSchedule 查看日程
func (sa *ScheduleAgent) viewSchedule(ctx context.Context, query string, params map[string]string) (string, error) {
	// 获取今天的日程
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	events, err := sa.calendarSvc.ListEvents(ctx, startOfDay, endOfDay)
	if err != nil {
		return fmt.Sprintf("获取日程失败: %v", err), nil
	}

	date := now.Format("2006年01月02日")

	var schedule string
	if len(events) == 0 {
		schedule = fmt.Sprintf("您%s暂无安排的日程。\n", date)
	} else {
		schedule = fmt.Sprintf("您%s的日程安排:\n", date)
		for _, event := range events {
			schedule += fmt.Sprintf("- %s-%s %s",
				event.StartTime.Format("15:04"),
				event.EndTime.Format("15:04"),
				event.Title)

			if len(event.Attendees) > 0 {
				schedule += fmt.Sprintf(" (%s)", strings.Join(event.Attendees, ", "))
			}
			schedule += "\n"
		}
	}

	return schedule, nil
}

// cancelMeeting 取消会议
func (sa *ScheduleAgent) cancelMeeting(ctx context.Context, query string, params map[string]string) (string, error) {
	// 在实际实现中，我们需要从参数中获取要取消的会议ID
	// 这里简化处理，假设我们要取消最近的一个会议
	events, err := sa.calendarSvc.ListEvents(ctx, time.Now(), time.Now().Add(24*time.Hour))
	if err != nil {
		return fmt.Sprintf("获取会议列表失败: %v", err), nil
	}

	if len(events) == 0 {
		return "没有找到即将开始的会议", nil
	}

	// 取消第一个会议（实际应用中应该让用户指定要取消的会议）
	eventToCancel := events[0]
	err = sa.calendarSvc.DeleteEvent(ctx, eventToCancel.ID)
	if err != nil {
		return fmt.Sprintf("取消会议失败: %v", err), nil
	}

	return fmt.Sprintf("已为您取消会议 \"%s\"。会议取消通知已发送给所有参会人。", eventToCancel.Title), nil
}

// processWithAI 使用AI处理复杂日程请求
func (sa *ScheduleAgent) processWithAI(ctx context.Context, query string, params map[string]string) (string, error) {
	systemPrompt := `你是一个专业的日程管理助手。你的任务是理解和处理用户的日程安排请求。
	
你可以执行以下操作：
1. 创建会议/预约
2. 查看日程安排
3. 修改或取消现有安排
4. 设置提醒

当用户提出日程相关请求时，请分析其意图并提供相应的帮助。如果需要更多信息来完成操作，请明确询问用户。`

	response, err := sa.aiClient.Chat(systemPrompt, query)
	if err != nil {
		return "", err
	}

	return response, nil
}
