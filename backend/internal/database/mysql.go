package database

import (
	"fmt"
	"time"

	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg config.MySQLConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMinutes) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleMinutes) * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.InviteCode{},
		&models.CourseClass{},
		&models.CourseCategory{},
		&models.ClassFavorite{},
		&models.SpecialPrice{},
		&models.Order{},
		&models.OrderEvent{},
		&models.WorkOrder{},
		&models.RechargeCard{},
		&models.RecommendedClass{},
		&models.Connector{},
		&models.PlatformPlugin{},
		&models.WorkerNode{},
		&models.WorkerCommand{},
		&models.WorkerProxy{},
		&models.AdminMenu{},
		&models.OperationLog{},
		&models.SystemJob{},
		&models.SiteConfig{},
	)
}
