package vo

import (
	"fmt"
	"time"
)

var weekdayMap = map[time.Weekday]string{
	time.Monday:    "星期一",
	time.Tuesday:   "星期二",
	time.Wednesday: "星期三",
	time.Thursday:  "星期四",
	time.Friday:    "星期五",
	time.Saturday:  "星期六",
	time.Sunday:    "星期日",
}

const Zone = "Asia/Shanghai"

type StreamLaunchPlan struct {
	Question             string
	ConversationId       string
	ChatMode             ChatQueryMode
	SelectedDocumentId   int64
	SelectedDocumentName string
	SelectedTaskId       int64
	CurrentDate          time.Time
	CurrentDateText      string
}

// FillCurrentDate 填充当前日期，例如 "2026-01-15（星期三）"
func (s *StreamLaunchPlan) FillCurrentDate() {
	loc, _ := time.LoadLocation(Zone)
	s.CurrentDate = time.Now().In(loc)
	s.CurrentDateText = fmt.Sprintf("%s（%s）", s.CurrentDate.Format("2006-01-02"), weekdayMap[s.CurrentDate.Weekday()])
}
