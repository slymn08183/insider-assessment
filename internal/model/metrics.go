package model

import "time"

// MetricsQuery GET /metrics request params
type MetricsQuery struct {
	EventName string
	From      time.Time
	To        time.Time
	Channel   string
	GroupBy   string // "daily" or "hourly"
}

// MetricsResult Aggregated metric results
type MetricsResult struct {
	TotalCount  int64          `json:"total_count"`
	UniqueUsers int64          `json:"unique_users"`
	Breakdown   []BreakdownRow `json:"breakdown,omitempty"`
}

// BreakdownRow group_by results
type BreakdownRow struct {
	Period      string `json:"period"`
	Count       int64  `json:"count"`
	UniqueUsers int64  `json:"unique_users"`
}
