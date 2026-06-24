package repository

import (
	"context"
	"strings"
	"time"

	"dw0rdwk/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlatformPluginRepository struct {
	db *gorm.DB
}

func (r *PlatformPluginRepository) List(ctx context.Context, search string, status string, page, perPage int) (Page[models.PlatformPlugin], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.PlatformPlugin{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("code LIKE ? OR name LIKE ? OR description LIKE ?", like, like, like)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	return paginate[models.PlatformPlugin](query, page, perPage, "sort_order ASC, code ASC")
}

func (r *PlatformPluginRepository) Find(ctx context.Context, code string) (models.PlatformPlugin, error) {
	var item models.PlatformPlugin
	err := r.db.WithContext(ctx).Where("code = ?", strings.TrimSpace(code)).First(&item).Error
	return item, err
}

func (r *PlatformPluginRepository) Create(ctx context.Context, item *models.PlatformPlugin) error {
	normalizePlatformPlugin(item)
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *PlatformPluginRepository) UpsertDefaults(ctx context.Context, items []models.PlatformPlugin) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		normalizePlatformPlugin(&items[i])
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoNothing: true,
	}).Create(&items).Error
}

func (r *PlatformPluginRepository) Update(ctx context.Context, code string, values map[string]any) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return gorm.ErrRecordNotFound
	}
	if len(values) == 0 {
		return nil
	}
	values["updated_at"] = time.Now()
	result := r.db.WithContext(ctx).Model(&models.PlatformPlugin{}).Where("code = ?", code).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *PlatformPluginRepository) Delete(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return gorm.ErrRecordNotFound
	}
	result := r.db.WithContext(ctx).Where("code = ?", code).Delete(&models.PlatformPlugin{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func normalizePlatformPlugin(item *models.PlatformPlugin) {
	item.Code = strings.TrimSpace(item.Code)
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	item.Status = strings.TrimSpace(item.Status)
	if item.Status == "" {
		item.Status = "disabled"
	}
	if item.SortOrder == 0 {
		item.SortOrder = 10
	}
	if item.MaxConcurrency < 1 {
		item.MaxConcurrency = 1
	}
	if strings.TrimSpace(item.ConfigJSON) == "" {
		item.ConfigJSON = "{}"
	}
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = now
	}
}

type WorkerNodeRepository struct {
	db *gorm.DB
}

func (r *WorkerNodeRepository) List(ctx context.Context) ([]models.WorkerNode, error) {
	var items []models.WorkerNode
	err := r.db.WithContext(ctx).Order("status ASC, worker_id ASC").Find(&items).Error
	return items, err
}

func (r *WorkerNodeRepository) UpsertHeartbeat(ctx context.Context, item models.WorkerNode) error {
	item.WorkerID = strings.TrimSpace(item.WorkerID)
	if item.WorkerID == "" {
		return gorm.ErrRecordNotFound
	}
	now := time.Now()
	if item.HeartbeatAt == nil {
		item.HeartbeatAt = &now
	}
	if item.StartedAt == nil {
		item.StartedAt = &now
	}
	if item.Status == "" {
		item.Status = "running"
	}
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "worker_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"hostname":            item.Hostname,
			"status":              item.Status,
			"accept_new":          item.AcceptNew,
			"max_concurrency":     item.MaxConcurrency,
			"running_count":       item.RunningCount,
			"current_order_id":    item.CurrentOrderID,
			"current_plugin_code": item.CurrentPluginCode,
			"message":             item.Message,
			"heartbeat_at":        item.HeartbeatAt,
			"updated_at":          now,
		}),
	}).Create(&item).Error
}

func (r *WorkerNodeRepository) MarkStopped(ctx context.Context, workerID string, message string) error {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return gorm.ErrRecordNotFound
	}
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.WorkerNode{}).Where("worker_id = ?", workerID).Updates(map[string]any{
		"status":              "stopped",
		"accept_new":          false,
		"running_count":       0,
		"current_order_id":    0,
		"current_plugin_code": "",
		"message":             strings.TrimSpace(message),
		"heartbeat_at":        &now,
		"updated_at":          now,
	}).Error
}

type WorkerCommandRepository struct {
	db *gorm.DB
}

func (r *WorkerCommandRepository) List(ctx context.Context, workerID string, page, perPage int) (Page[models.WorkerCommand], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.WorkerCommand{})
	if workerID = strings.TrimSpace(workerID); workerID != "" {
		query = query.Where("worker_id = ?", workerID)
	}
	return paginate[models.WorkerCommand](query, page, perPage, "id DESC")
}

func (r *WorkerCommandRepository) Create(ctx context.Context, item *models.WorkerCommand) error {
	item.WorkerID = strings.TrimSpace(item.WorkerID)
	item.Command = strings.TrimSpace(item.Command)
	if item.WorkerID == "" || item.Command == "" {
		return gorm.ErrRecordNotFound
	}
	if item.Status == "" {
		item.Status = "pending"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *WorkerCommandRepository) PendingForWorker(ctx context.Context, workerID string, limit int) ([]models.WorkerCommand, error) {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return []models.WorkerCommand{}, nil
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	var items []models.WorkerCommand
	err := r.db.WithContext(ctx).
		Where("status = ? AND worker_id IN ?", "pending", []string{workerID, "*"}).
		Order("id ASC").
		Limit(limit).
		Find(&items).
		Error
	return items, err
}

func (r *WorkerCommandRepository) Finish(ctx context.Context, id uint, status string, resultText string) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.WorkerCommand{}).Where("id = ?", id).Updates(map[string]any{
		"status":      strings.TrimSpace(status),
		"result":      truncateRepositoryText(resultText, 500),
		"executed_at": &now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type WorkerProxyRepository struct {
	db *gorm.DB
}

type WorkerProxyRow struct {
	models.WorkerProxy
	MaskedURL string `json:"maskedUrl"`
}

func (r *WorkerProxyRepository) List(ctx context.Context, search string, status string, page, perPage int) (Page[WorkerProxyRow], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.WorkerProxy{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR proxy_url LIKE ? OR id = ?", like, like, search)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	pageData, err := paginate[models.WorkerProxy](query, page, perPage, "status ASC, fail_count ASC, use_count ASC, id DESC")
	if err != nil {
		return Page[WorkerProxyRow]{}, err
	}
	rows := make([]WorkerProxyRow, 0, len(pageData.Items))
	for _, item := range pageData.Items {
		rows = append(rows, WorkerProxyRow{WorkerProxy: item, MaskedURL: maskProxyURL(item.ProxyURL)})
	}
	return Page[WorkerProxyRow]{Items: rows, Total: pageData.Total, Page: pageData.Page, PerPage: pageData.PerPage}, nil
}

func (r *WorkerProxyRepository) Create(ctx context.Context, item *models.WorkerProxy) error {
	normalizeWorkerProxy(item)
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *WorkerProxyRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	if len(values) == 0 {
		return nil
	}
	values["updated_at"] = time.Now()
	return updateByID[models.WorkerProxy](r.db.WithContext(ctx), "id", id, values)
}

func (r *WorkerProxyRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.WorkerProxy](r.db.WithContext(ctx), "id", id)
}

func (r *WorkerProxyRepository) Lease(ctx context.Context) (models.WorkerProxy, bool, error) {
	var leased models.WorkerProxy
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.WorkerProxy
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND in_use_count < max_concurrency", "active").
			Order("fail_count ASC, use_count ASC, last_used_at ASC, id ASC").
			First(&item).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		now := time.Now()
		if err := tx.Model(&models.WorkerProxy{}).Where("id = ?", item.ID).Updates(map[string]any{
			"in_use_count": gorm.Expr("in_use_count + 1"),
			"use_count":    gorm.Expr("use_count + 1"),
			"last_used_at": &now,
			"last_error":   "",
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}
		item.InUseCount++
		item.UseCount++
		item.LastUsedAt = &now
		leased = item
		return nil
	})
	if err != nil {
		return models.WorkerProxy{}, false, err
	}
	return leased, leased.ID != 0, nil
}

func (r *WorkerProxyRepository) Release(ctx context.Context, id uint, success bool, resultText string) error {
	if id == 0 {
		return nil
	}
	now := time.Now()
	values := map[string]any{
		"in_use_count": gorm.Expr("GREATEST(in_use_count - 1, 0)"),
		"updated_at":   now,
	}
	if success {
		values["success_count"] = gorm.Expr("success_count + 1")
		values["last_error"] = ""
	} else {
		values["fail_count"] = gorm.Expr("fail_count + 1")
		values["last_error"] = truncateRepositoryText(resultText, 500)
	}
	return r.db.WithContext(ctx).Model(&models.WorkerProxy{}).Where("id = ?", id).Updates(values).Error
}

func normalizeWorkerProxy(item *models.WorkerProxy) {
	item.Name = strings.TrimSpace(item.Name)
	item.ProxyURL = strings.TrimSpace(item.ProxyURL)
	item.Kind = strings.TrimSpace(item.Kind)
	if item.Kind == "" {
		item.Kind = "http"
	}
	item.Status = strings.TrimSpace(item.Status)
	if item.Status == "" {
		item.Status = "active"
	}
	if item.MaxConcurrency < 1 {
		item.MaxConcurrency = 1
	}
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = now
	}
}

func maskProxyURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	at := strings.LastIndex(value, "@")
	if at >= 0 {
		schemeSep := strings.Index(value, "://")
		if schemeSep >= 0 && schemeSep+3 < at {
			return value[:schemeSep+3] + "***:***" + value[at:]
		}
		return "***" + value[at:]
	}
	if len(value) <= 16 {
		return value
	}
	return value[:8] + "..." + value[len(value)-6:]
}

func truncateRepositoryText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
