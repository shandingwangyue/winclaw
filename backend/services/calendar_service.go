package services

import (
	"context"
	"errors"
	"time"
)

// CalendarEvent 日历事件结构
type CalendarEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Attendees   []string  `json:"attendees"`
	Location    string    `json:"location"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CalendarService 日历服务接口
type CalendarService interface {
	CreateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error)
	GetEvent(ctx context.Context, id string) (*CalendarEvent, error)
	UpdateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error)
	DeleteEvent(ctx context.Context, id string) error
	ListEvents(ctx context.Context, startTime, endTime time.Time) ([]*CalendarEvent, error)
	AddAttendee(ctx context.Context, eventId string, attendee string) error
	RemoveAttendee(ctx context.Context, eventId string, attendee string) error
	SendInvitation(ctx context.Context, eventId string) error
	SetReminder(ctx context.Context, eventId string, reminderMinutes int) error
}

// MockCalendarService 模拟日历服务实现
type MockCalendarService struct{}

// NewMockCalendarService 创建模拟日历服务
func NewMockCalendarService() *MockCalendarService {
	return &MockCalendarService{}
}

// CreateEvent 创建事件
func (mcs *MockCalendarService) CreateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error) {
	if event.Title == "" {
		return nil, errors.New("event title is required")
	}

	event.ID = "mock_event_" + event.StartTime.Format("20060102_150405")
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	return event, nil
}

// GetEvent 获取事件
func (mcs *MockCalendarService) GetEvent(ctx context.Context, id string) (*CalendarEvent, error) {
	if id == "" {
		return nil, errors.New("event id is required")
	}

	// 返回模拟数据
	event := &CalendarEvent{
		ID:        id,
		Title:     "模拟会议",
		StartTime: time.Now().Add(24 * time.Hour), // 明天
		EndTime:   time.Now().Add(25 * time.Hour), // 明天+1小时
		Attendees: []string{"user@example.com"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return event, nil
}

// UpdateEvent 更新事件
func (mcs *MockCalendarService) UpdateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error) {
	if event.ID == "" {
		return nil, errors.New("event id is required")
	}

	event.UpdatedAt = time.Now()
	return event, nil
}

// DeleteEvent 删除事件
func (mcs *MockCalendarService) DeleteEvent(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("event id is required")
	}

	// 模拟删除成功
	return nil
}

// ListEvents 列出事件
func (mcs *MockCalendarService) ListEvents(ctx context.Context, startTime, endTime time.Time) ([]*CalendarEvent, error) {
	events := []*CalendarEvent{
		{
			ID:        "event_1",
			Title:     "团队周会",
			StartTime: time.Now().Add(2 * time.Hour),
			EndTime:   time.Now().Add(3 * time.Hour),
			Attendees: []string{"team@example.com"},
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "event_2",
			Title:     "产品评审",
			StartTime: time.Now().Add(24 * time.Hour),
			EndTime:   time.Now().Add(25 * time.Hour),
			Attendees: []string{"product@example.com"},
			CreatedAt: time.Now().Add(-12 * time.Hour),
			UpdatedAt: time.Now(),
		},
	}

	return events, nil
}

// AddAttendee 添加参与者
func (mcs *MockCalendarService) AddAttendee(ctx context.Context, eventId string, attendee string) error {
	if eventId == "" || attendee == "" {
		return errors.New("event id and attendee are required")
	}

	return nil
}

// RemoveAttendee 移除参与者
func (mcs *MockCalendarService) RemoveAttendee(ctx context.Context, eventId string, attendee string) error {
	if eventId == "" || attendee == "" {
		return errors.New("event id and attendee are required")
	}

	return nil
}

// SendInvitation 发送邀请
func (mcs *MockCalendarService) SendInvitation(ctx context.Context, eventId string) error {
	if eventId == "" {
		return errors.New("event id is required")
	}

	return nil
}

// SetReminder 设置提醒
func (mcs *MockCalendarService) SetReminder(ctx context.Context, eventId string, reminderMinutes int) error {
	if eventId == "" {
		return errors.New("event id is required")
	}

	return nil
}
