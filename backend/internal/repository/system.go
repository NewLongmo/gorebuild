package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"dw0rdwk/backend/internal/models"

	"gorm.io/gorm"
)

type ConnectorRepository struct {
	db *gorm.DB
}

func (r *ConnectorRepository) List(ctx context.Context, search string, status string, page, perPage int) (Page[models.Connector], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.Connector{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR base_url LIKE ? OR kind LIKE ?", like, like, like)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	return paginate[models.Connector](query, page, perPage, "sort_order ASC, id DESC")
}

func (r *ConnectorRepository) Find(ctx context.Context, id uint) (models.Connector, error) {
	var connector models.Connector
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&connector).Error
	return connector, err
}

func (r *ConnectorRepository) FirstActive(ctx context.Context) (models.Connector, error) {
	var connector models.Connector
	err := r.db.WithContext(ctx).
		Where("status = ? AND base_url <> '' AND order_sync_enabled = ?", "active", true).
		Order("sort_order ASC, id ASC").
		First(&connector).
		Error
	return connector, err
}

func (r *ConnectorRepository) ListActiveWithBaseURL(ctx context.Context) ([]models.Connector, error) {
	var connectors []models.Connector
	err := r.db.WithContext(ctx).
		Model(&models.Connector{}).
		Where("status = ? AND base_url <> ''", "active").
		Order("sort_order ASC, id ASC").
		Find(&connectors).
		Error
	return connectors, err
}

func (r *ConnectorRepository) Create(ctx context.Context, connector *models.Connector) error {
	if connector.CreatedAt.IsZero() {
		connector.CreatedAt = time.Now()
	}
	if connector.Kind == "" {
		connector.Kind = "generic"
	}
	if connector.Status == "" {
		connector.Status = "active"
	}
	if connector.TimeoutMS == 0 {
		connector.TimeoutMS = 8000
	}
	if connector.PriceMode == "" {
		connector.PriceMode = "multiplier"
	}
	if connector.PriceValue == 0 {
		connector.PriceValue = 1
	}
	if connector.PriceRounding == "" {
		connector.PriceRounding = "none"
	}
	if connector.SortOrder == 0 {
		connector.SortOrder = 10
	}
	return r.db.WithContext(ctx).Select("*").Create(connector).Error
}

type LogRepository struct {
	db *gorm.DB
}

func (r *LogRepository) List(ctx context.Context, search string, page, perPage int) (Page[models.OperationLog], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.OperationLog{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("type LIKE ? OR text LIKE ? OR user_id = ?", like, like, search)
	}
	return paginate[models.OperationLog](query, page, perPage, "id DESC")
}

func (r *LogRepository) ListByUser(ctx context.Context, userID uint, search string, page, perPage int) (Page[models.OperationLog], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.OperationLog{}).Where("user_id = ?", userID)
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("type LIKE ? OR text LIKE ?", like, like)
	}
	return paginate[models.OperationLog](query, page, perPage, "id DESC")
}

func (r *LogRepository) Create(ctx context.Context, log *models.OperationLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *ConnectorRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return updateByID[models.Connector](r.db.WithContext(ctx), "id", id, values)
}

func (r *ConnectorRepository) Delete(ctx context.Context, id uint) error {
	_, err := r.DeleteCascade(ctx, id)
	return err
}

func (r *ConnectorRepository) DeleteCascade(ctx context.Context, id uint) (CascadeDeleteResult, error) {
	result := CascadeDeleteResult{ID: id}
	if id == 0 {
		return result, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var connector models.Connector
		if err := tx.Where("id = ?", id).First(&connector).Error; err != nil {
			return err
		}
		connectorID := strconv.FormatUint(uint64(id), 10)
		var classes []models.CourseClass
		if err := tx.
			Where("docking_platform = ? OR query_platform = ?", connectorID, connectorID).
			Find(&classes).
			Error; err != nil {
			return err
		}
		deleted, err := deleteCourseClassesCascadeTx(tx, classes, true)
		if err != nil {
			return err
		}
		result.DeletedClasses = deleted.DeletedClasses
		result.DeletedFavorites = deleted.DeletedFavorites
		result.DeletedSpecialPrices = deleted.DeletedSpecialPrices
		result.DeletedEmptyCategories = deleted.DeletedEmptyCategories
		deleteResult := tx.Where("id = ?", id).Delete(&models.Connector{})
		if deleteResult.Error != nil {
			return deleteResult.Error
		}
		if deleteResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	return result, err
}

type MenuRepository struct {
	db *gorm.DB
}

func (r *MenuRepository) List(ctx context.Context, includeHidden bool) ([]models.AdminMenu, error) {
	query := r.db.WithContext(ctx).Model(&models.AdminMenu{})
	if !includeHidden {
		query = query.Where("visible = ?", true)
	}
	var items []models.AdminMenu
	err := query.Order("parent_id ASC, sort_order ASC, id ASC").Find(&items).Error
	return items, err
}

func (r *MenuRepository) Find(ctx context.Context, id uint) (models.AdminMenu, error) {
	var item models.AdminMenu
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (r *MenuRepository) Create(ctx context.Context, item *models.AdminMenu) error {
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = now
	}
	if item.Type == "" {
		item.Type = "menu"
	}
	if item.SortOrder == 0 {
		item.SortOrder = 10
	}
	return r.db.WithContext(ctx).Select("*").Create(item).Error
}

func (r *MenuRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	values["updated_at"] = time.Now()
	return updateByID[models.AdminMenu](r.db.WithContext(ctx), "id", id, values)
}

func (r *MenuRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.AdminMenu](r.db.WithContext(ctx), "id", id)
}

func (r *MenuRepository) HasChildren(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.AdminMenu{}).Where("parent_id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *MenuRepository) ParentExists(ctx context.Context, parentID uint) (bool, error) {
	if parentID == 0 {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&models.AdminMenu{}).Where("id = ?", parentID).Count(&count).Error
	return count > 0, err
}

func (r *MenuRepository) WouldCreateCycle(ctx context.Context, id uint, parentID uint) (bool, error) {
	if id == 0 || parentID == 0 {
		return false, nil
	}
	if id == parentID {
		return true, nil
	}
	seen := map[uint]struct{}{}
	current := parentID
	for current != 0 {
		if current == id {
			return true, nil
		}
		if _, ok := seen[current]; ok {
			return true, nil
		}
		seen[current] = struct{}{}
		parent, err := r.Find(ctx, current)
		if err != nil {
			return false, err
		}
		current = parent.ParentID
	}
	return false, nil
}

type MenuSortItem struct {
	ID        uint
	ParentID  uint
	SortOrder int
}

func (r *MenuRepository) UpdateSort(ctx context.Context, items []MenuSortItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		for _, item := range items {
			if item.ID == 0 {
				continue
			}
			result := tx.Model(&models.AdminMenu{}).
				Where("id = ?", item.ID).
				Updates(map[string]any{
					"parent_id":  item.ParentID,
					"sort_order": item.SortOrder,
					"updated_at": now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		return nil
	})
}

type SettingRepository struct {
	db *gorm.DB
}

func (r *SettingRepository) All(ctx context.Context) (map[string]string, error) {
	var rows []models.SiteConfig
	if err := r.db.WithContext(ctx).Order("`key` ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.Key] = row.Value
	}
	return result, nil
}

func (r *SettingRepository) Upsert(ctx context.Context, values map[string]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			row := models.SiteConfig{Key: key, Value: value}
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
