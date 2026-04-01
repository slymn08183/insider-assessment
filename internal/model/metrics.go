package model

import (
	"fmt"
	"time"
)

// MetricsQuery GET /metrics request params
type MetricsQuery struct {
	EventName string    `form:"event_name" binding:"required"`
	From      time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To        time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
	Channel   string    `form:"channel"`
	GroupBy   string    `form:"group_by"`
}

// Validate MetricsQuery
func (q *MetricsQuery) Validate() error {

	if q.GroupBy != "" && q.GroupBy != "daily" && q.GroupBy != "hourly" {
		return fmt.Errorf("group_by must be 'daily' or 'hourly'")
	}

	if !q.From.IsZero() && !q.To.IsZero() && q.From.After(q.To) {
		return fmt.Errorf("'from' cannot be after 'to'")
	}

	return nil
}

// MetricsResult Aggregated metric results
type MetricsResult struct {
	TotalCount  int64          `json:"total_count"`
	UniqueUsers int64          `json:"unique_users"`
	Breakdown   []BreakdownRow `json:"breakdown"`
}

// BreakdownRow group_by results
type BreakdownRow struct {
	Period      string `json:"period"`
	Count       int64  `json:"count"`
	UniqueUsers int64  `json:"unique_users"`
}
