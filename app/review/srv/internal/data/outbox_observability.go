package data

import (
	"context"

	"goshop/app/review/srv/internal/domain"
	"goshop/gmicro/core/metric"

	"gorm.io/gorm"
)

var metricReviewOutboxBacklog = metric.NewGaugeVec(&metric.GaugeVecOpts{
	Namespace: "review_service",
	Subsystem: "outbox",
	Name:      "backlog",
	Help:      "Current review rating outbox events grouped by status.",
	Labels:    []string{"status"},
})

func observeReviewOutboxBacklog(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}

	statuses := []string{"PENDING", "DONE"}
	for _, status := range statuses {
		var count int64
		if err := db.WithContext(ctx).
			Model(&domain.OutboxEvent{}).
			Where("status = ?", status).
			Count(&count).Error; err != nil {
			return err
		}
		metricReviewOutboxBacklog.Set(float64(count), status)
	}
	return nil
}
