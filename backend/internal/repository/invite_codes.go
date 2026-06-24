package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"dw0rdwk/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInviteCodeInvalid   = errors.New("invite code is invalid")
	ErrInviteCodeExpired   = errors.New("invite code is expired")
	ErrInviteCodeExhausted = errors.New("invite code has no remaining uses")
)

type InviteCodeRepository struct {
	db *gorm.DB
}

func NormalizeInviteCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func (r *InviteCodeRepository) List(ctx context.Context, search string, status string, page, perPage int) (Page[models.InviteCode], error) {
	page, perPage = normalizePage(page, perPage)
	query := r.db.WithContext(ctx).Model(&models.InviteCode{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + strings.ToUpper(search) + "%"
		noteLike := "%" + search + "%"
		query = query.Where("code LIKE ? OR note LIKE ? OR id = ?", like, noteLike, search)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	return paginate[models.InviteCode](query, page, perPage, "id DESC")
}

func (r *InviteCodeRepository) Find(ctx context.Context, id uint) (models.InviteCode, error) {
	var invite models.InviteCode
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&invite).Error
	return invite, err
}

func (r *InviteCodeRepository) CodeExists(ctx context.Context, code string) (bool, error) {
	code = NormalizeInviteCode(code)
	if code == "" {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.InviteCode{}).
		Where("code = ?", code).
		Count(&count).
		Error
	return count > 0, err
}

func (r *InviteCodeRepository) Create(ctx context.Context, invite *models.InviteCode) error {
	invite.Code = NormalizeInviteCode(invite.Code)
	if invite.Code == "" {
		return ErrInviteCodeInvalid
	}
	if invite.MaxUses == 0 {
		invite.MaxUses = 1
	}
	if invite.PriceRate == 0 {
		invite.PriceRate = 1
	}
	if invite.Status == "" {
		invite.Status = "active"
	}
	now := time.Now()
	if invite.CreatedAt.IsZero() {
		invite.CreatedAt = now
	}
	if invite.UpdatedAt.IsZero() {
		invite.UpdatedAt = now
	}
	return r.db.WithContext(ctx).Create(invite).Error
}

func (r *InviteCodeRepository) Update(ctx context.Context, id uint, values map[string]any) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	if len(values) == 0 {
		return nil
	}
	values["updated_at"] = time.Now()
	return updateByID[models.InviteCode](r.db.WithContext(ctx), "id", id, values)
}

func (r *InviteCodeRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return deleteByID[models.InviteCode](r.db.WithContext(ctx), "id", id)
}

func (r *InviteCodeRepository) RegisterUser(ctx context.Context, code string, user *models.User) error {
	code = NormalizeInviteCode(code)
	if code == "" {
		return ErrInviteCodeInvalid
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invite models.InviteCode
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ?", code).
			First(&invite).
			Error
		if err == nil {
			return registerUserWithAdminInvite(tx, invite, user)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return registerUserWithAgentInvite(tx, code, user)
	})
}

func registerUserWithAdminInvite(tx *gorm.DB, invite models.InviteCode, user *models.User) error {
	if err := validateInviteForUse(invite, time.Now()); err != nil {
		return err
	}
	user.InviteCode = invite.Code
	user.Role = "agent"
	user.Status = "active"
	user.PriceRate = invite.PriceRate
	user.Balance = 0
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if err := tx.Create(user).Error; err != nil {
		return err
	}
	result := tx.Model(&models.InviteCode{}).
		Where("id = ? AND used_count = ?", invite.ID, invite.UsedCount).
		Updates(map[string]any{
			"used_count": invite.UsedCount + 1,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInviteCodeExhausted
	}
	return nil
}

func registerUserWithAgentInvite(tx *gorm.DB, code string, user *models.User) error {
	var parent models.User
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("invite_code = ? AND role = ? AND status = ?", code, "agent", "active").
		Order("id ASC").
		First(&parent).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInviteCodeInvalid
		}
		return err
	}
	priceRate := parent.InvitePriceRate
	if priceRate == 0 {
		priceRate = parent.PriceRate
	}
	if priceRate == 0 {
		priceRate = 1
	}
	user.ParentID = parent.ID
	user.InviteCode = code
	user.Role = "agent"
	user.Status = "active"
	user.PriceRate = priceRate
	user.Balance = 0
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	return tx.Create(user).Error
}

func validateInviteForUse(invite models.InviteCode, now time.Time) error {
	if invite.Status != "active" {
		return ErrInviteCodeInvalid
	}
	if invite.ExpiresAt != nil && !invite.ExpiresAt.After(now) {
		return ErrInviteCodeExpired
	}
	if invite.MaxUses < 1 || invite.UsedCount >= invite.MaxUses {
		return ErrInviteCodeExhausted
	}
	return nil
}

func IsInviteCodeInvalid(err error) bool {
	return errors.Is(err, ErrInviteCodeInvalid)
}

func IsInviteCodeExpired(err error) bool {
	return errors.Is(err, ErrInviteCodeExpired)
}

func IsInviteCodeExhausted(err error) bool {
	return errors.Is(err, ErrInviteCodeExhausted)
}
