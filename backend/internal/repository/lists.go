package repository

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"dw0rdwk/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrRechargeCardUsed = errors.New("recharge card already used")
var ErrOrderAlreadyRefunded = errors.New("order already refunded")

type ChildBalanceTransferResult struct {
	ParentID       uint    `json:"parentId"`
	ChildID        uint    `json:"childId"`
	ChargedAmount  float64 `json:"chargedAmount"`
	CreditedAmount float64 `json:"creditedAmount"`
}

type UserRepository struct {
	db *gorm.DB
}

func (r *UserRepository) List(ctx context.Context, search string, page, perPage int) (Page[models.User], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.User{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("account LIKE ? OR name LIKE ? OR id = ?", like, like, search)
	}
	return paginate[models.User](query, page, perPage, "id DESC")
}

func (r *UserRepository) ListAgentsForTree(ctx context.Context, limit int) ([]models.User, bool, error) {
	if limit < 1 {
		limit = 5000
	}
	if limit > 10000 {
		limit = 10000
	}
	var users []models.User
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("role = ?", "agent").
		Order("id ASC").
		Limit(limit + 1).
		Find(&users).
		Error
	if err != nil {
		return nil, false, err
	}
	truncated := len(users) > limit
	if truncated {
		users = users[:limit]
	}
	return users, truncated, nil
}

func (r *UserRepository) ListChildren(ctx context.Context, parentID uint, search string, page, perPage int) (Page[models.User], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.User{}).Where("parent_id = ?", parentID)
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("account LIKE ? OR name LIKE ? OR id = ?", like, like, search)
	}
	return paginate[models.User](query, page, perPage, "id DESC")
}

func (r *UserRepository) CountChildren(ctx context.Context, parentID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("parent_id = ?", parentID).Count(&count).Error
	return count, err
}

func (r *UserRepository) FindByAccount(ctx context.Context, account string) (models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Where("account = ?", strings.TrimSpace(account)).
		First(&user).
		Error
	return user, err
}

func (r *UserRepository) InviteCodeExists(ctx context.Context, code string, exceptID uint) (bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return false, nil
	}
	query := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("invite_code = ?", code)
	if exceptID != 0 {
		query = query.Where("id <> ?", exceptID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) Find(ctx context.Context, id uint) (models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	return user, err
}

func (r *UserRepository) FindForAPI(ctx context.Context, id uint, key string) (models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND api_key = ? AND api_key <> '' AND api_key <> '0'", id, strings.TrimSpace(key)).
		First(&user).
		Error
	return user, err
}

func (r *UserRepository) FindChild(ctx context.Context, parentID uint, childID uint) (models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND parent_id = ?", childID, parentID).
		First(&user).
		Error
	return user, err
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.Status == "" {
		user.Status = "active"
	}
	if user.PriceRate == 0 {
		user.PriceRate = 1
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) CreateChildWithBalance(ctx context.Context, parentID uint, user *models.User, initialBalance float64) error {
	if parentID == 0 {
		return gorm.ErrRecordNotFound
	}
	if initialBalance < 0 {
		return ErrInsufficientBalance
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if initialBalance > 0 {
			result := tx.Model(&models.User{}).
				Where("id = ? AND balance >= ?", parentID, initialBalance).
				Update("balance", gorm.Expr("balance - ?", initialBalance))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ErrInsufficientBalance
			}
		}
		user.ParentID = parentID
		user.Role = "agent"
		user.Balance = initialBalance
		now := time.Now()
		if user.CreatedAt.IsZero() {
			user.CreatedAt = now
		}
		if user.Status == "" {
			user.Status = "active"
		}
		if user.PriceRate == 0 {
			user.PriceRate = 1
		}
		return tx.Create(user).Error
	})
}

func (r *UserRepository) Update(ctx context.Context, uid uint, values map[string]any) error {
	if uid == 0 {
		return gorm.ErrRecordNotFound
	}
	return updateByID[models.User](r.db.WithContext(ctx), "id", uid, values)
}

func (r *UserRepository) UpdateChild(ctx context.Context, parentID uint, childID uint, values map[string]any) error {
	if parentID == 0 || childID == 0 {
		return gorm.ErrRecordNotFound
	}
	if len(values) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ? AND parent_id = ?", childID, parentID).
		Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UserRepository) DeductBalanceIfEnough(ctx context.Context, uid uint, amount float64) error {
	if uid == 0 {
		return gorm.ErrRecordNotFound
	}
	if amount <= 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ? AND balance >= ?", uid, amount).
		Update("balance", gorm.Expr("balance - ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UserRepository) AdjustBalance(ctx context.Context, uid uint, amount float64) error {
	if uid == 0 {
		return gorm.ErrRecordNotFound
	}
	if amount == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", uid).
		Update("balance", gorm.Expr("balance + ?", amount)).
		Error
}

func (r *UserRepository) AdjustBalanceChecked(ctx context.Context, uid uint, amount float64) error {
	if uid == 0 {
		return gorm.ErrRecordNotFound
	}
	if amount == 0 {
		return nil
	}
	query := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", uid)
	if amount < 0 {
		query = query.Where("balance >= ?", -amount)
	}
	result := query.Update("balance", gorm.Expr("balance + ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if amount < 0 {
			return ErrInsufficientBalance
		}
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UserRepository) TransferBalanceToChild(ctx context.Context, parentID uint, childID uint, amount float64) (ChildBalanceTransferResult, error) {
	result := ChildBalanceTransferResult{ParentID: parentID, ChildID: childID}
	if parentID == 0 || childID == 0 {
		return result, gorm.ErrRecordNotFound
	}
	if amount == 0 {
		return result, ErrInsufficientBalance
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if amount > 0 {
			charged, err := transferParentToChild(tx, parentID, childID, amount)
			if err != nil {
				return err
			}
			result.ChargedAmount = charged
			result.CreditedAmount = amount
			return nil
		}
		debited := -amount
		if err := transferChildToParent(tx, parentID, childID, debited); err != nil {
			return err
		}
		result.ChargedAmount = -debited
		result.CreditedAmount = -debited
		return nil
	})
	return result, err
}

func transferParentToChild(tx *gorm.DB, parentID uint, childID uint, amount float64) (float64, error) {
	var parent models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", parentID).First(&parent).Error; err != nil {
		return 0, err
	}
	var child models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND parent_id = ?", childID, parentID).First(&child).Error; err != nil {
		return 0, err
	}
	charged := childRechargeCost(amount, parent.PriceRate, child.PriceRate)
	result := tx.Model(&models.User{}).
		Where("id = ? AND balance >= ?", parentID, charged).
		Update("balance", gorm.Expr("balance - ?", charged))
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		return 0, ErrInsufficientBalance
	}
	result = tx.Model(&models.User{}).
		Where("id = ? AND parent_id = ?", childID, parentID).
		Update("balance", gorm.Expr("balance + ?", amount))
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return charged, nil
}

func childRechargeCost(amount, parentRate, childRate float64) float64 {
	parentRate = positiveRate(parentRate)
	childRate = positiveRate(childRate)
	return roundMoney(amount * parentRate / childRate)
}

func positiveRate(value float64) float64 {
	if value <= 0 {
		return 1
	}
	return value
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func transferChildToParent(tx *gorm.DB, parentID uint, childID uint, amount float64) error {
	result := tx.Model(&models.User{}).
		Where("id = ? AND parent_id = ? AND balance >= ?", childID, parentID, amount).
		Update("balance", gorm.Expr("balance - ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInsufficientBalance
	}
	result = tx.Model(&models.User{}).
		Where("id = ?", parentID).
		Update("balance", gorm.Expr("balance + ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, uid uint) error {
	if uid == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.User](r.db.WithContext(ctx), "id", uid)
}

type ClassRepository struct {
	db *gorm.DB
}

func (r *ClassRepository) List(ctx context.Context, search string, status *string, category string, page, perPage int) (Page[models.CourseClass], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.CourseClass{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR docking_code LIKE ? OR category LIKE ?", like, like, like)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if category = strings.TrimSpace(category); category != "" {
		query = query.Where("category = ?", category)
	}
	return paginate[models.CourseClass](query, page, perPage, "sort DESC, id DESC")
}

func (r *ClassRepository) Find(ctx context.Context, id uint) (models.CourseClass, error) {
	var item models.CourseClass
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (r *ClassRepository) ListByDockingCodes(ctx context.Context, dockingPlatform string, codes []string) (map[string]models.CourseClass, error) {
	dockingPlatform = strings.TrimSpace(dockingPlatform)
	normalized := make([]string, 0, len(codes))
	seen := map[string]struct{}{}
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		normalized = append(normalized, code)
	}
	result := make(map[string]models.CourseClass, len(normalized))
	if dockingPlatform == "" || len(normalized) == 0 {
		return result, nil
	}
	var items []models.CourseClass
	err := r.db.WithContext(ctx).
		Where("docking_platform = ? AND docking_code IN ?", dockingPlatform, normalized).
		Find(&items).
		Error
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.DockingCode] = item
	}
	return result, nil
}

func (r *ClassRepository) ListOnlineForAPI(ctx context.Context, id uint, limit int) ([]models.CourseClass, error) {
	if limit < 1 {
		limit = 500
	}
	if limit > 1000 {
		limit = 1000
	}
	query := r.db.WithContext(ctx).
		Model(&models.CourseClass{}).
		Where("status = ?", "online").
		Order("sort ASC, id DESC").
		Limit(limit)
	if id > 0 {
		query = query.Where("id = ?", id)
	}
	var items []models.CourseClass
	err := query.Find(&items).Error
	return items, err
}

func (r *ClassRepository) Create(ctx context.Context, item *models.CourseClass) error {
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if item.PriceOperator == "" {
		item.PriceOperator = "*"
	}
	if item.Status == "" {
		item.Status = "online"
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *ClassRepository) Update(ctx context.Context, cid uint, values map[string]any) error {
	if cid == 0 {
		return gorm.ErrRecordNotFound
	}
	return updateByID[models.CourseClass](r.db.WithContext(ctx), "id", cid, values)
}

func (r *ClassRepository) BulkUpdateStatus(ctx context.Context, ids []uint, status string) (int64, error) {
	ids = normalizeIDs(ids)
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.CourseClass{}).
		Where("id IN ?", ids).
		Update("status", strings.TrimSpace(status))
	return result.RowsAffected, result.Error
}

func (r *ClassRepository) BulkMove(ctx context.Context, ids []uint, category string) (int64, error) {
	ids = normalizeIDs(ids)
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.CourseClass{}).
		Where("id IN ?", ids).
		Update("category", strings.TrimSpace(category))
	return result.RowsAffected, result.Error
}

func (r *ClassRepository) BulkDelete(ctx context.Context, ids []uint) (int64, error) {
	ids = normalizeIDs(ids)
	if len(ids) == 0 {
		return 0, nil
	}
	var deleted int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		classes, err := findClassesByIDsTx(tx, ids)
		if err != nil {
			return err
		}
		result, err := deleteCourseClassesCascadeTx(tx, classes, false)
		if err != nil {
			return err
		}
		deleted = result.DeletedClasses
		return nil
	})
	return deleted, err
}

func (r *ClassRepository) ReplaceKeyword(ctx context.Context, scope string, scopeID string, oldKeyword string, newKeyword string) (int64, error) {
	query := r.classScopeQuery(ctx, scope, scopeID)
	result := query.Update("name", gorm.Expr("REPLACE(name, ?, ?)", oldKeyword, newKeyword))
	return result.RowsAffected, result.Error
}

func (r *ClassRepository) AddPrefix(ctx context.Context, scope string, scopeID string, prefix string) (int64, error) {
	query := r.classScopeQuery(ctx, scope, scopeID)
	result := query.Update("name", gorm.Expr("CONCAT(?, name)", prefix))
	return result.RowsAffected, result.Error
}

type DuplicateClassGroup struct {
	Key             string   `json:"key"`
	Category        string   `json:"category"`
	DockingPlatform string   `json:"dockingPlatform"`
	DockingCode     string   `json:"dockingCode"`
	KeepID          uint     `json:"keepId"`
	DeleteIDs       []uint   `json:"deleteIds"`
	Names           []string `json:"names"`
	Count           int      `json:"count"`
}

func (r *ClassRepository) DeduplicatePreview(ctx context.Context, scope string, scopeID string, keepStrategy string, limit int) ([]DuplicateClassGroup, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	var items []models.CourseClass
	err := r.classScopeQuery(ctx, scope, scopeID).
		Where("docking_code <> ''").
		Order("docking_platform ASC, docking_code ASC, category ASC, id ASC").
		Find(&items).
		Error
	if err != nil {
		return nil, err
	}
	type bucket struct {
		items []models.CourseClass
	}
	buckets := map[string]bucket{}
	for _, item := range items {
		key := strings.Join([]string{item.DockingPlatform, item.DockingCode, item.Category}, "\x00")
		group := buckets[key]
		group.items = append(group.items, item)
		buckets[key] = group
	}
	groups := make([]DuplicateClassGroup, 0)
	for key, group := range buckets {
		if len(group.items) < 2 {
			continue
		}
		keepIndex := 0
		if keepStrategy == "keep_newer" {
			keepIndex = len(group.items) - 1
		}
		keep := group.items[keepIndex]
		deleteIDs := make([]uint, 0, len(group.items)-1)
		names := make([]string, 0, len(group.items))
		for i, item := range group.items {
			names = append(names, item.Name)
			if i != keepIndex {
				deleteIDs = append(deleteIDs, item.ID)
			}
		}
		groups = append(groups, DuplicateClassGroup{
			Key:             key,
			Category:        keep.Category,
			DockingPlatform: keep.DockingPlatform,
			DockingCode:     keep.DockingCode,
			KeepID:          keep.ID,
			DeleteIDs:       deleteIDs,
			Names:           names,
			Count:           len(group.items),
		})
		if len(groups) >= limit {
			break
		}
	}
	return groups, nil
}

func (r *ClassRepository) DeduplicateApply(ctx context.Context, scope string, scopeID string, keepStrategy string, limit int) (int64, error) {
	groups, err := r.DeduplicatePreview(ctx, scope, scopeID, keepStrategy, limit)
	if err != nil {
		return 0, err
	}
	ids := make([]uint, 0)
	for _, group := range groups {
		ids = append(ids, group.DeleteIDs...)
	}
	return r.BulkDelete(ctx, ids)
}

func (r *ClassRepository) UpdateByDockingCode(ctx context.Context, dockingPlatform string, dockingCode string, values map[string]any) (bool, error) {
	dockingPlatform = strings.TrimSpace(dockingPlatform)
	dockingCode = strings.TrimSpace(dockingCode)
	if dockingPlatform == "" || dockingCode == "" || len(values) == 0 {
		return false, nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.CourseClass{}).
		Where("docking_platform = ? AND docking_code = ?", dockingPlatform, dockingCode).
		Updates(values)
	return result.RowsAffected > 0, result.Error
}

func (r *ClassRepository) MarkMissingDockingOffline(ctx context.Context, dockingPlatform string, keepCodes []string) (int64, error) {
	dockingPlatform = strings.TrimSpace(dockingPlatform)
	if dockingPlatform == "" {
		return 0, nil
	}
	query := r.db.WithContext(ctx).
		Model(&models.CourseClass{}).
		Where("docking_platform = ?", dockingPlatform)
	if len(keepCodes) > 0 {
		query = query.Where("docking_code NOT IN ?", keepCodes)
	}
	result := query.Update("status", "offline")
	return result.RowsAffected, result.Error
}

func (r *ClassRepository) classScopeQuery(ctx context.Context, scope string, scopeID string) *gorm.DB {
	query := r.db.WithContext(ctx).Model(&models.CourseClass{})
	switch strings.TrimSpace(scope) {
	case "category":
		return query.Where("category = ?", strings.TrimSpace(scopeID))
	case "docking":
		return query.Where("docking_platform = ?", strings.TrimSpace(scopeID))
	default:
		return query
	}
}

func (r *ClassRepository) Delete(ctx context.Context, cid uint) error {
	if cid == 0 {
		return gorm.ErrRecordNotFound
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		classes, err := findClassesByIDsTx(tx, []uint{cid})
		if err != nil {
			return err
		}
		if len(classes) == 0 {
			return gorm.ErrRecordNotFound
		}
		_, err = deleteCourseClassesCascadeTx(tx, classes, false)
		return err
	})
}

type CategoryRepository struct {
	db *gorm.DB
}

func (r *CategoryRepository) List(ctx context.Context, search string, status string, page, perPage int) (Page[models.CourseCategory], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.CourseCategory{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR id = ?", like, search)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	return paginate[models.CourseCategory](query, page, perPage, "sort DESC, id DESC")
}

func (r *CategoryRepository) ListActive(ctx context.Context, limit int) ([]models.CourseCategory, error) {
	if limit < 1 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	var items []models.CourseCategory
	err := r.db.WithContext(ctx).
		Model(&models.CourseCategory{}).
		Where("status = ?", "active").
		Order("pinned DESC, sort DESC, id DESC").
		Limit(limit).
		Find(&items).
		Error
	return items, err
}

func (r *CategoryRepository) Find(ctx context.Context, id uint) (models.CourseCategory, error) {
	var item models.CourseCategory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (r *CategoryRepository) FindByName(ctx context.Context, name string) (models.CourseCategory, error) {
	var item models.CourseCategory
	err := r.db.WithContext(ctx).Where("name = ?", strings.TrimSpace(name)).First(&item).Error
	return item, err
}

func (r *CategoryRepository) Create(ctx context.Context, item *models.CourseCategory) error {
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if item.Status == "" {
		item.Status = "active"
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *CategoryRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return updateByID[models.CourseCategory](r.db.WithContext(ctx), "id", id, values)
}

func (r *CategoryRepository) Delete(ctx context.Context, id uint) error {
	_, err := r.DeleteCascade(ctx, id)
	return err
}

func (r *CategoryRepository) DeleteCascade(ctx context.Context, id uint) (CascadeDeleteResult, error) {
	result := CascadeDeleteResult{ID: id}
	if id == 0 {
		return result, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var category models.CourseCategory
		if err := tx.Where("id = ?", id).First(&category).Error; err != nil {
			return err
		}
		var classes []models.CourseClass
		if err := tx.Where("category = ?", strings.TrimSpace(category.Name)).Find(&classes).Error; err != nil {
			return err
		}
		deleted, err := deleteCourseClassesCascadeTx(tx, classes, false)
		if err != nil {
			return err
		}
		result.DeletedClasses = deleted.DeletedClasses
		result.DeletedFavorites = deleted.DeletedFavorites
		result.DeletedSpecialPrices = deleted.DeletedSpecialPrices
		deleteResult := tx.Where("id = ?", id).Delete(&models.CourseCategory{})
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

func findClassesByIDsTx(tx *gorm.DB, ids []uint) ([]models.CourseClass, error) {
	ids = normalizeIDs(ids)
	if len(ids) == 0 {
		return []models.CourseClass{}, nil
	}
	var classes []models.CourseClass
	err := tx.Where("id IN ?", ids).Find(&classes).Error
	return classes, err
}

func deleteCourseClassesCascadeTx(tx *gorm.DB, classes []models.CourseClass, cleanupEmptyCategories bool) (CascadeDeleteResult, error) {
	result := CascadeDeleteResult{}
	classIDs := courseClassIDs(classes)
	if len(classIDs) == 0 {
		return result, nil
	}
	deletedFavorites, err := deleteClassFavoritesTx(tx, classIDs)
	if err != nil {
		return result, err
	}
	result.DeletedFavorites = deletedFavorites
	deletedSpecialPrices, err := deleteClassSpecialPricesTx(tx, classIDs)
	if err != nil {
		return result, err
	}
	result.DeletedSpecialPrices = deletedSpecialPrices
	deletedClasses := tx.Where("id IN ?", classIDs).Delete(&models.CourseClass{})
	if deletedClasses.Error != nil {
		return result, deletedClasses.Error
	}
	result.DeletedClasses = deletedClasses.RowsAffected
	if cleanupEmptyCategories {
		deletedCategories, err := deleteEmptyCategoriesByNamesTx(tx, courseClassCategories(classes))
		if err != nil {
			return result, err
		}
		result.DeletedEmptyCategories = deletedCategories
	}
	return result, nil
}

func deleteClassFavoritesTx(tx *gorm.DB, classIDs []uint) (int64, error) {
	result := tx.Where("class_id IN ?", classIDs).Delete(&models.ClassFavorite{})
	return result.RowsAffected, result.Error
}

func deleteClassSpecialPricesTx(tx *gorm.DB, classIDs []uint) (int64, error) {
	result := tx.Where("class_id IN ?", classIDs).Delete(&models.SpecialPrice{})
	return result.RowsAffected, result.Error
}

func deleteEmptyCategoriesByNamesTx(tx *gorm.DB, names []string) (int64, error) {
	var total int64
	for _, name := range names {
		var count int64
		if err := tx.Model(&models.CourseClass{}).Where("category = ?", name).Count(&count).Error; err != nil {
			return total, err
		}
		if count != 0 {
			continue
		}
		result := tx.Where("name = ?", name).Delete(&models.CourseCategory{})
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
	}
	return total, nil
}

func courseClassIDs(classes []models.CourseClass) []uint {
	ids := make([]uint, 0, len(classes))
	for _, class := range classes {
		if class.ID != 0 {
			ids = append(ids, class.ID)
		}
	}
	return normalizeIDs(ids)
}

func courseClassCategories(classes []models.CourseClass) []string {
	seen := map[string]struct{}{}
	categories := make([]string, 0)
	for _, class := range classes {
		category := strings.TrimSpace(class.Category)
		if category == "" {
			continue
		}
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		categories = append(categories, category)
	}
	return categories
}

type ClassFavoriteRepository struct {
	db *gorm.DB
}

func (r *ClassFavoriteRepository) ListByUser(ctx context.Context, userID uint) ([]models.ClassFavorite, error) {
	if userID == 0 {
		return []models.ClassFavorite{}, nil
	}
	var items []models.ClassFavorite
	err := r.db.WithContext(ctx).
		Model(&models.ClassFavorite{}).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&items).
		Error
	return items, err
}

func (r *ClassFavoriteRepository) IDsByUser(ctx context.Context, userID uint) (map[uint]struct{}, error) {
	items, err := r.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make(map[uint]struct{}, len(items))
	for _, item := range items {
		ids[item.ClassID] = struct{}{}
	}
	return ids, nil
}

func (r *ClassFavoriteRepository) Add(ctx context.Context, userID uint, classID uint) error {
	if userID == 0 || classID == 0 {
		return gorm.ErrRecordNotFound
	}
	item := models.ClassFavorite{UserID: userID, ClassID: classID, CreatedAt: time.Now()}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&item).
		Error
}

func (r *ClassFavoriteRepository) Remove(ctx context.Context, userID uint, classID uint) error {
	if userID == 0 || classID == 0 {
		return gorm.ErrRecordNotFound
	}
	return r.db.WithContext(ctx).
		Where("user_id = ? AND class_id = ?", userID, classID).
		Delete(&models.ClassFavorite{}).
		Error
}

type SpecialPriceRepository struct {
	db *gorm.DB
}

type SpecialPriceRow struct {
	models.SpecialPrice
	UserAccount string `json:"userAccount"`
	ClassName   string `json:"className"`
}

func (r *SpecialPriceRepository) List(ctx context.Context, search string, page, perPage int) (Page[SpecialPriceRow], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).
		Table("special_prices").
		Select("special_prices.*, users.account AS user_account, course_classes.name AS class_name").
		Joins("LEFT JOIN users ON users.id = special_prices.user_id").
		Joins("LEFT JOIN course_classes ON course_classes.id = special_prices.class_id")
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("users.account LIKE ? OR course_classes.name LIKE ? OR special_prices.user_id = ? OR special_prices.class_id = ?", like, like, search, search)
	}
	return paginate[SpecialPriceRow](query, page, perPage, "special_prices.id DESC")
}

func (r *SpecialPriceRepository) FindForUserClass(ctx context.Context, userID uint, classID uint) (models.SpecialPrice, bool, error) {
	if userID == 0 || classID == 0 {
		return models.SpecialPrice{}, false, nil
	}
	var item models.SpecialPrice
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND class_id = ?", userID, classID).
		First(&item).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.SpecialPrice{}, false, nil
		}
		return models.SpecialPrice{}, false, err
	}
	return item, true, nil
}

func (r *SpecialPriceRepository) Find(ctx context.Context, id uint) (models.SpecialPrice, error) {
	var item models.SpecialPrice
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (r *SpecialPriceRepository) Upsert(ctx context.Context, item *models.SpecialPrice) error {
	if item.UserID == 0 || item.ClassID == 0 {
		return gorm.ErrRecordNotFound
	}
	return r.db.WithContext(ctx).
		Where("user_id = ? AND class_id = ?", item.UserID, item.ClassID).
		Assign(map[string]any{
			"mode":  item.Mode,
			"price": item.Price,
		}).
		FirstOrCreate(item).
		Error
}

func (r *SpecialPriceRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return updateByID[models.SpecialPrice](r.db.WithContext(ctx), "id", id, values)
}

func (r *SpecialPriceRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.SpecialPrice](r.db.WithContext(ctx), "id", id)
}

type OrderRepository struct {
	db *gorm.DB
}

func (r *OrderRepository) List(ctx context.Context, search string, status string, flashMode *bool, page, perPage int) (Page[models.Order], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.Order{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("account LIKE ? OR school LIKE ? OR course_name LIKE ? OR id = ?", like, like, like, search)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	if flashMode != nil {
		query = query.Where("flash_mode = ?", *flashMode)
	}
	return paginate[models.Order](query, page, perPage, "id DESC")
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID uint, search string, status string, page, perPage int) (Page[models.Order], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID)
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("account LIKE ? OR school LIKE ? OR course_name LIKE ? OR id = ?", like, like, like, search)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	return paginate[models.Order](query, page, perPage, "id DESC")
}

func (r *OrderRepository) ListPublicByAccount(ctx context.Context, account string, limit int) ([]models.Order, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []models.Order{}, nil
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	var orders []models.Order
	err := r.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("account = ?", account).
		Order("id DESC").
		Limit(limit).
		Find(&orders).
		Error
	return orders, err
}

func (r *OrderRepository) ListByUserAccount(ctx context.Context, userID uint, account string, limit int) ([]models.Order, error) {
	account = strings.TrimSpace(account)
	if userID == 0 || account == "" {
		return []models.Order{}, nil
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var orders []models.Order
	err := r.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("user_id = ? AND account = ?", userID, account).
		Order("id ASC").
		Limit(limit).
		Find(&orders).
		Error
	return orders, err
}

func (r *OrderRepository) CountByUser(ctx context.Context, userID uint, statuses []string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *OrderRepository) CountByUserSince(ctx context.Context, userID uint, since time.Time, statuses []string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ? AND created_at >= ?", userID, since)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *OrderRepository) CountByUserNotStatuses(ctx context.Context, userID uint, statuses []string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID)
	if len(statuses) > 0 {
		query = query.Where("status NOT IN ?", statuses)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *OrderRepository) CountByUserDockingStatuses(ctx context.Context, userID uint, dockingStatuses []string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID)
	if len(dockingStatuses) > 0 {
		query = query.Where("docking_status IN ?", dockingStatuses)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *OrderRepository) CountProcessingPlugin(ctx context.Context, pluginCode string, account string, excludeID uint) (int64, error) {
	pluginCode = strings.TrimSpace(pluginCode)
	if pluginCode == "" {
		return 0, nil
	}
	query := r.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("plugin_code = ? AND status = ?", pluginCode, "processing")
	if account = strings.TrimSpace(account); account != "" {
		query = query.Where("account = ?", account)
	}
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

type DailyOrderPoint struct {
	Date   string `json:"date"`
	Orders int64  `json:"orders"`
}

func (r *OrderRepository) DailyTrendByUser(ctx context.Context, userID uint, days int) ([]DailyOrderPoint, error) {
	if days < 1 {
		days = 7
	}
	if days > 60 {
		days = 60
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
	points := make([]DailyOrderPoint, 0, days)
	for i := 0; i < days; i++ {
		dayStart := start.AddDate(0, 0, i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		point := DailyOrderPoint{Date: dayStart.Format("01-02")}
		if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, dayStart, dayEnd).Count(&point.Orders).Error; err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

func (r *OrderRepository) SumFeesByUser(ctx context.Context, userID uint) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(fee), 0)").
		Scan(&total).
		Error
	return total, err
}

func (r *OrderRepository) Find(ctx context.Context, id uint) (models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&order).Error
	return order, err
}

func (r *OrderRepository) FindByUser(ctx context.Context, id uint, userID uint) (models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&order).Error
	return order, err
}

func (r *OrderRepository) FindSyncCandidate(ctx context.Context, connectorID uint, remoteOrderID string, account string, courseName string, dockingCode string) (models.Order, bool, error) {
	if connectorID == 0 {
		return models.Order{}, false, nil
	}
	remoteOrderID = strings.TrimSpace(remoteOrderID)
	if remoteOrderID != "" {
		order, ok, err := r.findOpenSyncCandidate(ctx, "connector_id = ? AND remote_order_id = ?", connectorID, remoteOrderID)
		if err != nil || ok {
			return order, ok, err
		}
	}
	account = strings.TrimSpace(account)
	courseName = strings.TrimSpace(courseName)
	dockingCode = strings.TrimSpace(dockingCode)
	if account == "" || courseName == "" || dockingCode == "" {
		return models.Order{}, false, nil
	}
	return r.findOpenSyncCandidate(ctx, "connector_id = ? AND account = ? AND course_name = ? AND docking_code = ?", connectorID, account, courseName, dockingCode)
}

func (r *OrderRepository) findOpenSyncCandidate(ctx context.Context, where string, args ...any) (models.Order, bool, error) {
	var order models.Order
	err := r.db.WithContext(ctx).
		Where(where, args...).
		Where("status NOT IN ?", finalOrderStatuses()).
		Order("id DESC").
		First(&order).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Order{}, false, nil
		}
		return models.Order{}, false, err
	}
	return order, true, nil
}

func (r *OrderRepository) UpdateSyncCandidate(ctx context.Context, id uint, values map[string]any) (bool, error) {
	if id == 0 || len(values) == 0 {
		return false, nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("id = ? AND status NOT IN ?", id, finalOrderStatuses()).
		Updates(values)
	return result.RowsAffected > 0, result.Error
}

func finalOrderStatuses() []string {
	return []string{"done", "failed", "cancelled", "refunded"}
}

func (r *OrderRepository) ListQueuedAfter(ctx context.Context, afterID uint, limit int) ([]models.Order, error) {
	return r.ListByStatusesAfter(ctx, afterID, []string{"queued"}, limit)
}

func (r *OrderRepository) ListRecoverableQueueAfter(ctx context.Context, afterID uint, limit int) ([]models.Order, error) {
	if limit < 1 {
		limit = 100
	}
	var orders []models.Order
	err := r.db.WithContext(ctx).
		Where(
			"id > ? AND (status = ? OR (status = ? AND docking_status IN ?))",
			afterID,
			"queued",
			"processing",
			[]string{"pending", "refresh_requested"},
		).
		Order("id ASC").
		Limit(limit).
		Find(&orders).
		Error
	return orders, err
}

func (r *OrderRepository) ListByStatusesAfter(ctx context.Context, afterID uint, statuses []string, limit int) ([]models.Order, error) {
	if limit < 1 {
		limit = 100
	}
	if len(statuses) == 0 {
		statuses = []string{"queued"}
	}
	var orders []models.Order
	err := r.db.WithContext(ctx).
		Where("id > ? AND status IN ?", afterID, statuses).
		Order("id ASC").
		Limit(limit).
		Find(&orders).
		Error
	return orders, err
}

func (r *OrderRepository) Create(ctx context.Context, order *models.Order) error {
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now()
	}
	if order.Status == "" {
		order.Status = "pending"
	}
	if order.DockingStatus == "" {
		order.DockingStatus = "pending"
	}
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *OrderRepository) Update(ctx context.Context, oid uint, values map[string]any) error {
	if oid == 0 {
		return gorm.ErrRecordNotFound
	}
	return updateByID[models.Order](r.db.WithContext(ctx), "id", oid, values)
}

func (r *OrderRepository) Refund(ctx context.Context, oid uint) (models.Order, error) {
	if oid == 0 {
		return models.Order{}, gorm.ErrRecordNotFound
	}
	var order models.Order
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", oid).First(&order).Error; err != nil {
			return err
		}
		if order.Status == "refunded" {
			return ErrOrderAlreadyRefunded
		}
		if order.UserID != 0 && order.Fee > 0 {
			result := tx.Model(&models.User{}).
				Where("id = ?", order.UserID).
				Update("balance", gorm.Expr("balance + ?", order.Fee))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		if err := tx.Model(&models.Order{}).
			Where("id = ?", oid).
			Updates(map[string]any{
				"status":         "refunded",
				"docking_status": "refunded",
				"remarks":        "refunded by admin",
			}).Error; err != nil {
			return err
		}
		order.Status = "refunded"
		order.DockingStatus = "refunded"
		order.Remarks = "refunded by admin"
		return nil
	})
	return order, err
}

func (r *OrderRepository) Delete(ctx context.Context, oid uint) error {
	if oid == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.Order](r.db.WithContext(ctx), "id", oid)
}

type WorkOrderRepository struct {
	db *gorm.DB
}

type WorkOrderRow struct {
	models.WorkOrder
	UserAccount string `json:"userAccount"`
}

func (r *WorkOrderRepository) List(ctx context.Context, search string, status string, userID *uint, page, perPage int) (Page[WorkOrderRow], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).
		Table("work_orders").
		Select("work_orders.*, users.account AS user_account").
		Joins("LEFT JOIN users ON users.id = work_orders.user_id")
	if userID != nil {
		query = query.Where("work_orders.user_id = ? AND work_orders.user_visible = ?", *userID, true)
	}
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("work_orders.title LIKE ? OR work_orders.content LIKE ? OR work_orders.category LIKE ? OR users.account LIKE ? OR work_orders.id = ?", like, like, like, like, search)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("work_orders.status = ?", status)
	}
	return paginate[WorkOrderRow](query, page, perPage, "work_orders.id DESC")
}

func (r *WorkOrderRepository) Find(ctx context.Context, id uint) (models.WorkOrder, error) {
	var item models.WorkOrder
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return item, err
}

func (r *WorkOrderRepository) FindByUser(ctx context.Context, id uint, userID uint) (models.WorkOrder, error) {
	var item models.WorkOrder
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&item).Error
	return item, err
}

func (r *WorkOrderRepository) Create(ctx context.Context, item *models.WorkOrder) error {
	if item.UserID == 0 {
		return gorm.ErrRecordNotFound
	}
	if item.Status == "" {
		item.Status = "待回复"
	}
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = now
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *WorkOrderRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return updateByID[models.WorkOrder](r.db.WithContext(ctx), "id", id, values)
}

func (r *WorkOrderRepository) UpdateByUser(ctx context.Context, id uint, userID uint, values map[string]any) error {
	if id == 0 || userID == 0 {
		return gorm.ErrRecordNotFound
	}
	if len(values) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.WorkOrder{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *WorkOrderRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.WorkOrder](r.db.WithContext(ctx), "id", id)
}

func (r *WorkOrderRepository) DeleteByUser(ctx context.Context, id uint, userID uint) error {
	if id == 0 || userID == 0 {
		return gorm.ErrRecordNotFound
	}
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.WorkOrder{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type RechargeCardRepository struct {
	db *gorm.DB
}

type RechargeCardRow struct {
	models.RechargeCard
	UserAccount string `json:"userAccount"`
}

func (r *RechargeCardRepository) List(ctx context.Context, search string, status string, page, perPage int) (Page[RechargeCardRow], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).
		Table("recharge_cards").
		Select("recharge_cards.*, users.account AS user_account").
		Joins("LEFT JOIN users ON users.id = recharge_cards.user_id")
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("recharge_cards.code LIKE ? OR users.account LIKE ? OR recharge_cards.id = ? OR recharge_cards.user_id = ?", like, like, search, search)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("recharge_cards.status = ?", status)
	}
	return paginate[RechargeCardRow](query, page, perPage, "recharge_cards.id DESC")
}

func (r *RechargeCardRepository) FindByCode(ctx context.Context, code string) (models.RechargeCard, error) {
	var item models.RechargeCard
	err := r.db.WithContext(ctx).Where("code = ?", strings.TrimSpace(code)).First(&item).Error
	return item, err
}

func (r *RechargeCardRepository) CreateBatch(ctx context.Context, items []models.RechargeCard) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		if items[i].Status == "" {
			items[i].Status = "unused"
		}
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

func (r *RechargeCardRepository) Redeem(ctx context.Context, code string, userID uint) (models.RechargeCard, error) {
	code = strings.TrimSpace(code)
	if code == "" || userID == 0 {
		return models.RechargeCard{}, gorm.ErrRecordNotFound
	}
	var redeemed models.RechargeCard
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var card models.RechargeCard
		if err := tx.Where("code = ?", code).First(&card).Error; err != nil {
			return err
		}
		if card.Status != "unused" {
			return ErrRechargeCardUsed
		}
		now := time.Now()
		result := tx.Model(&models.RechargeCard{}).
			Where("id = ? AND status = ?", card.ID, "unused").
			Updates(map[string]any{"status": "used", "user_id": userID, "used_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrRechargeCardUsed
		}
		result = tx.Model(&models.User{}).
			Where("id = ? AND status = ?", userID, "active").
			Update("balance", gorm.Expr("balance + ?", card.Amount))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		card.Status = "used"
		card.UserID = userID
		card.UsedAt = &now
		redeemed = card
		return nil
	})
	return redeemed, err
}

func (r *RechargeCardRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.RechargeCard](r.db.WithContext(ctx), "id", id)
}

func paginate[T any](query *gorm.DB, page, perPage int, order string) (Page[T], error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return Page[T]{}, err
	}

	var items []T
	err := query.Order(order).Offset((page - 1) * perPage).Limit(perPage).Find(&items).Error
	if err != nil {
		return Page[T]{}, err
	}

	return Page[T]{
		Items:   items,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

func updateByID[T any](db *gorm.DB, column string, id uint, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	result := db.Model(new(T)).Where(column+" = ?", id).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func deleteByID[T any](db *gorm.DB, column string, id uint) error {
	result := db.Where(column+" = ?", id).Delete(new(T))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func IsInsufficientBalance(err error) bool {
	return errors.Is(err, ErrInsufficientBalance)
}

func IsRechargeCardUsed(err error) bool {
	return errors.Is(err, ErrRechargeCardUsed)
}

func IsOrderAlreadyRefunded(err error) bool {
	return errors.Is(err, ErrOrderAlreadyRefunded)
}
