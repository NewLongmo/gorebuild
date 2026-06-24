package repository

import (
	"context"

	"gorm.io/gorm"
)

type HealthRepository struct {
	db *gorm.DB
}

func (r *HealthRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
