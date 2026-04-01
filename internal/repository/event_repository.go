package repository

import (
	"github.com/slymn08183/insider-assessment/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

// BatchInsert
// ON CONFLICT (event_hash) DO NOTHING
func (r *EventRepository) BatchInsert(events []*model.Event) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "event_hash"}},
		DoNothing: true,
	}).Create(&events).Error
}

// GetMetrics Filter by event_name and return aggregated results
func (r *EventRepository) GetMetrics(q model.MetricsQuery) (*model.MetricsResult, error) {
	result := &model.MetricsResult{}

	base := r.db.Model(&model.Event{}).Where("event_name = ?", q.EventName)

	if !q.From.IsZero() {
		base = base.Where("timestamp >= ?", q.From)
	}
	if !q.To.IsZero() {
		base = base.Where("timestamp <= ?", q.To)
	}
	if q.Channel != "" {
		base = base.Where("channel = ?", q.Channel)
	}

	// Total count
	// &gorm.Session{} creates a fresh copy so the base is not mutated
	if err := base.Session(&gorm.Session{}).Count(&result.TotalCount).Error; err != nil {
		return nil, err
	}

	// Unique users
	if err := base.Session(&gorm.Session{}).Distinct("user_id").Count(&result.UniqueUsers).Error; err != nil {
		return nil, err
	}

	// Breakdown (daily/hourly)
	if q.GroupBy == "daily" || q.GroupBy == "hourly" {
		var err error
		result.Breakdown, err = r.getBreakdown(base.Session(&gorm.Session{}), q.GroupBy)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// getBreakdown group by time
func (r *EventRepository) getBreakdown(base *gorm.DB, groupBy string) ([]model.BreakdownRow, error) {
	trunc := "day"
	if groupBy == "hourly" {
		trunc = "hour"
	}

	var rows []model.BreakdownRow
	err := base.
		Select("date_trunc(?, to_timestamp(timestamp)) as period, count(*) as count, count(distinct user_id) as unique_users", trunc).
		Group("period").
		Order("period").
		Find(&rows).Error

	return rows, err
}
