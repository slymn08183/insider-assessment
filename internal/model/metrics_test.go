package model

import (
	"testing"
	"time"
)

func TestMetricsQueryValidate(t *testing.T) {
	// query
	q := &MetricsQuery{
		EventName: "product_view",
		GroupBy:   "daily",
	}
	if err := q.Validate(); err != nil {
		t.Errorf("valid query should pass, got: %v", err)
	}

	// group_by empty
	q.GroupBy = ""
	if err := q.Validate(); err != nil {
		t.Errorf("empty group_by should pass, got: %v", err)
	}

	// invalid group_by
	q.GroupBy = "weekly"
	if err := q.Validate(); err == nil {
		t.Error("invalid group_by should fail")
	}

	// hourly
	q.GroupBy = "hourly"
	if err := q.Validate(); err != nil {
		t.Errorf("hourly group_by should pass, got: %v", err)
	}

	// from > to
	q.GroupBy = ""
	q.From = time.Date(2024, 8, 10, 0, 0, 0, 0, time.UTC)
	q.To = time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := q.Validate(); err == nil {
		t.Error("from after to should fail")
	}

	// from < to
	q.To = time.Date(2024, 8, 20, 0, 0, 0, 0, time.UTC)
	if err := q.Validate(); err != nil {
		t.Errorf("from before to should pass, got: %v", err)
	}
}
