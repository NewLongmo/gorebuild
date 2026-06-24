package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"dw0rdwk/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderEventRepository struct {
	db *gorm.DB
}

func (r *OrderEventRepository) Create(ctx context.Context, item *models.OrderEvent) error {
	if item.OrderID == 0 {
		return gorm.ErrRecordNotFound
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if item.Level == "" {
		item.Level = "info"
	}
	if item.Source == "" {
		item.Source = "system"
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *OrderEventRepository) ListByOrder(ctx context.Context, orderID uint, visibleOnly bool, limit int) ([]models.OrderEvent, error) {
	if orderID == 0 {
		return []models.OrderEvent{}, nil
	}
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	query := r.db.WithContext(ctx).Model(&models.OrderEvent{}).Where("order_id = ?", orderID)
	if visibleOnly {
		query = query.Where("visible_to_user = ?", true)
	}
	var items []models.OrderEvent
	err := query.Order("id DESC").Limit(limit).Find(&items).Error
	return items, err
}

type RecommendationRepository struct {
	db *gorm.DB
}

type RecommendationRow struct {
	models.RecommendedClass
	ClassName      string  `json:"className"`
	ClassCategory  string  `json:"classCategory"`
	ClassPrice     float64 `json:"classPrice"`
	ClassStatus    string  `json:"classStatus"`
	ClassDocking   string  `json:"classDocking"`
	ClassQuery     string  `json:"classQuery"`
	ClassDesc      string  `json:"classDescription"`
	ClassBridge    bool    `json:"classBridgeEnabled"`
	ClassCreatedAt string  `json:"classCreatedAt"`
}

func (r *RecommendationRepository) List(ctx context.Context, includeHidden bool, page, perPage int) (Page[RecommendationRow], error) {
	page, perPage = normalizePage(page, perPage)
	query := recommendationRowsQuery(r.db.WithContext(ctx))
	if !includeHidden {
		query = query.Where("recommended_classes.visible = ?", true)
	}
	return paginate[RecommendationRow](query, page, perPage, "recommended_classes.sort_order ASC, recommended_classes.id DESC")
}

func (r *RecommendationRepository) ListVisible(ctx context.Context, limit int) ([]RecommendationRow, error) {
	if limit < 1 {
		limit = 8
	}
	if limit > 30 {
		limit = 30
	}
	var items []RecommendationRow
	err := recommendationRowsQuery(r.db.WithContext(ctx)).
		Where("recommended_classes.visible = ?", true).
		Where("course_classes.status = ?", "online").
		Order("recommended_classes.sort_order ASC, recommended_classes.id DESC").
		Limit(limit).
		Find(&items).
		Error
	return items, err
}

func recommendationRowsQuery(db *gorm.DB) *gorm.DB {
	return db.
		Table("recommended_classes").
		Select(`recommended_classes.*,
			course_classes.name AS class_name,
			course_classes.category AS class_category,
			course_classes.price AS class_price,
			course_classes.status AS class_status,
			course_classes.docking_platform AS class_docking,
			course_classes.query_platform AS class_query,
			course_classes.description AS class_desc,
			course_classes.bridge_enabled AS class_bridge,
			DATE_FORMAT(course_classes.created_at, '%Y-%m-%d %H:%i:%s') AS class_created_at`).
		Joins("LEFT JOIN course_classes ON course_classes.id = recommended_classes.class_id")
}

func (r *RecommendationRepository) Find(ctx context.Context, id uint) (models.RecommendedClass, error) {
	var item models.RecommendedClass
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (r *RecommendationRepository) Create(ctx context.Context, item *models.RecommendedClass) error {
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = now
	}
	if item.SortOrder == 0 {
		item.SortOrder = 10
	}
	return r.db.WithContext(ctx).Select("*").Create(item).Error
}

func (r *RecommendationRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	values["updated_at"] = time.Now()
	return updateByID[models.RecommendedClass](r.db.WithContext(ctx), "id", id, values)
}

func (r *RecommendationRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.RecommendedClass](r.db.WithContext(ctx), "id", id)
}

type OrderLeaderboardRow struct {
	Name        string  `json:"name"`
	Count       int64   `json:"count"`
	Amount      float64 `json:"amount"`
	LastOrderAt string  `json:"lastOrderAt"`
}

func (r *OrderRepository) PlatformLeaderboard(ctx context.Context, limit int) ([]OrderLeaderboardRow, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	var rows []OrderLeaderboardRow
	err := r.db.WithContext(ctx).
		Table("orders").
		Select("COALESCE(NULLIF(platform, ''), '未命名平台') AS name, COUNT(*) AS count, COALESCE(SUM(fee), 0) AS amount, DATE_FORMAT(MAX(created_at), '%Y-%m-%d %H:%i:%s') AS last_order_at").
		Group("COALESCE(NULLIF(platform, ''), '未命名平台')").
		Order("count DESC, MAX(created_at) DESC").
		Limit(limit).
		Find(&rows).
		Error
	return rows, err
}

func (r *OrderRepository) CategoryLeaderboard(ctx context.Context, limit int) ([]OrderLeaderboardRow, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	var rows []OrderLeaderboardRow
	err := r.db.WithContext(ctx).
		Table("orders").
		Select("COALESCE(NULLIF(course_classes.category, ''), '未分类') AS name, COUNT(*) AS count, COALESCE(SUM(orders.fee), 0) AS amount, DATE_FORMAT(MAX(orders.created_at), '%Y-%m-%d %H:%i:%s') AS last_order_at").
		Joins("LEFT JOIN course_classes ON course_classes.id = orders.class_id").
		Group("COALESCE(NULLIF(course_classes.category, ''), '未分类')").
		Order("count DESC, MAX(orders.created_at) DESC").
		Limit(limit).
		Find(&rows).
		Error
	return rows, err
}

type SystemJobRepository struct {
	db *gorm.DB
}

func (r *SystemJobRepository) List(ctx context.Context) ([]models.SystemJob, error) {
	var items []models.SystemJob
	err := r.db.WithContext(ctx).Order("name ASC").Find(&items).Error
	return items, err
}

func (r *SystemJobRepository) Find(ctx context.Context, name string) (models.SystemJob, error) {
	var item models.SystemJob
	err := r.db.WithContext(ctx).Where("name = ?", strings.TrimSpace(name)).First(&item).Error
	return item, err
}

func (r *SystemJobRepository) IsEnabled(ctx context.Context, name string) bool {
	item, err := r.Find(ctx, name)
	if err != nil {
		return true
	}
	return item.Enabled
}

func (r *SystemJobRepository) MarkStarted(ctx context.Context, name string) (time.Time, error) {
	started := time.Now()
	item := models.SystemJob{
		Name:          strings.TrimSpace(name),
		Status:        "running",
		Enabled:       true,
		LastStartedAt: &started,
		HeartbeatAt:   &started,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"status":          "running",
			"last_started_at": started,
			"heartbeat_at":    started,
			"last_error":      "",
		}),
	}).Create(&item).Error
	return started, err
}

func (r *SystemJobRepository) MarkFinished(ctx context.Context, name string, started time.Time, summary any, runErr error) error {
	finished := time.Now()
	status := "success"
	lastError := ""
	if runErr != nil {
		status = "failed"
		lastError = runErr.Error()
	}
	summaryJSON := "{}"
	if summary != nil {
		if data, err := json.Marshal(summary); err == nil {
			summaryJSON = string(data)
		}
	}
	item := models.SystemJob{
		Name:            strings.TrimSpace(name),
		Status:          status,
		Enabled:         true,
		LastStartedAt:   &started,
		LastFinishedAt:  &finished,
		LastDurationMS:  finished.Sub(started).Milliseconds(),
		LastError:       lastError,
		LastSummaryJSON: summaryJSON,
		HeartbeatAt:     &finished,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"status":            status,
			"last_started_at":   started,
			"last_finished_at":  finished,
			"last_duration_ms":  finished.Sub(started).Milliseconds(),
			"last_error":        lastError,
			"last_summary_json": summaryJSON,
			"heartbeat_at":      finished,
		}),
	}).Create(&item).Error
}

func (r *SystemJobRepository) UpdateEnabled(ctx context.Context, name string, enabled bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return gorm.ErrRecordNotFound
	}
	now := time.Now()
	item := models.SystemJob{
		Name:        name,
		Status:      "idle",
		Enabled:     enabled,
		HeartbeatAt: &now,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"enabled":      enabled,
			"heartbeat_at": now,
		}),
	}).Create(&item).Error
}
