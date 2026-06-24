package database

import (
	"time"

	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/password"
	"dw0rdwk/backend/internal/platforms"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SeedDefaults(db *gorm.DB, cfg config.BootstrapConfig) error {
	now := time.Now()
	passwordHash, err := password.Hash(cfg.AdminPassword)
	if err != nil {
		return err
	}
	admin := models.User{
		Account:      cfg.AdminAccount,
		PasswordHash: passwordHash,
		Name:         cfg.AdminName,
		PriceRate:    1,
		Role:         "admin",
		Status:       "active",
		CreatedAt:    now,
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account"}},
		DoNothing: true,
	}).Create(&admin).Error; err != nil {
		return err
	}

	settings := []models.SiteConfig{
		{Key: "site_name", Value: "DW0RDWK"},
		{Key: "site_notice", Value: ""},
		{Key: "popup_notice", Value: ""},
		{Key: "notice_url", Value: ""},
		{Key: "order_tips", Value: ""},
		{Key: "open_query_notice", Value: ""},
		{Key: "recharge_bonus_rules", Value: "[]"},
		{Key: "order_auto_refresh", Value: "false"},
		{Key: "dashboard_cache_seconds", Value: "30"},
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&settings).Error; err != nil {
		return err
	}
	if err := seedSystemJobs(db, now); err != nil {
		return err
	}
	if err := seedPlatformPlugins(db, now); err != nil {
		return err
	}
	return seedAdminMenus(db, now)
}

func seedSystemJobs(db *gorm.DB, now time.Time) error {
	jobs := []models.SystemJob{
		{Name: "29wk_order_sync", Status: "idle", Enabled: true, HeartbeatAt: &now},
		{Name: "29wk_price_sync", Status: "idle", Enabled: true, HeartbeatAt: &now},
		{Name: "order_queue_recover", Status: "idle", Enabled: true, HeartbeatAt: &now},
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&jobs).Error
}

func seedAdminMenus(db *gorm.DB, now time.Time) error {
	var count int64
	if err := db.Model(&models.AdminMenu{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ensureAdminMenuSeeds(db, now, defaultAdminMenuSeeds())
	}

	return ensureAdminMenuSeeds(db, now, defaultAdminMenuSeeds())
}

func ensureAdminMenuSeeds(db *gorm.DB, now time.Time, seeds []adminMenuSeed) error {
	for _, seed := range seeds {
		parent := models.AdminMenu{
			Name:       seed.Name,
			Route:      seed.Route,
			Icon:       seed.Icon,
			Type:       seed.Type,
			SortOrder:  seed.SortOrder,
			Visible:    true,
			Permission: "admin",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		parentID, err := ensureAdminMenu(db, &parent)
		if err != nil {
			return err
		}
		for _, childSeed := range seed.Children {
			child := models.AdminMenu{
				ParentID:   parentID,
				Name:       childSeed.Name,
				Route:      childSeed.Route,
				Icon:       childSeed.Icon,
				Type:       childSeed.Type,
				SortOrder:  childSeed.SortOrder,
				Visible:    true,
				Permission: "admin",
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if _, err := ensureAdminMenu(db, &child); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureAdminMenu(db *gorm.DB, item *models.AdminMenu) (uint, error) {
	var existing models.AdminMenu
	err := db.Where("parent_id = ? AND name = ?", item.ParentID, item.Name).First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	if err := db.Create(item).Error; err != nil {
		return 0, err
	}
	return item.ID, nil
}

func seedPlatformPlugins(db *gorm.DB, now time.Time) error {
	items := platforms.DefaultPluginSeeds()
	for i := range items {
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = now
		}
		if items[i].UpdatedAt.IsZero() {
			items[i].UpdatedAt = now
		}
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoNothing: true,
	}).Create(&items).Error
}

type adminMenuSeed struct {
	Name      string
	Route     string
	Icon      string
	Type      string
	SortOrder int
	Children  []adminMenuSeed
}

func defaultAdminMenuSeeds() []adminMenuSeed {
	return []adminMenuSeed{
		{Name: "首页", Route: "/dashboard", Icon: "home", Type: "menu", SortOrder: 10},
		{
			Name:      "系统管理",
			Icon:      "settings",
			Type:      "dir",
			SortOrder: 20,
			Children: []adminMenuSeed{
				{Name: "系统配置", Route: "/settings", Icon: "settings", Type: "menu", SortOrder: 10},
				{Name: "系统公告", Route: "/settings", Icon: "notification", Type: "menu", SortOrder: 20},
				{Name: "常见问答", Route: "/support", Icon: "question", Type: "menu", SortOrder: 30},
				{Name: "用户管理", Route: "/users", Icon: "user", Type: "menu", SortOrder: 40},
				{Name: "角色管理", Route: "/agents", Icon: "team", Type: "menu", SortOrder: 50},
				{Name: "菜单管理", Route: "/menus", Icon: "menu", Type: "menu", SortOrder: 60},
				{Name: "卡密管理", Route: "/recharge-cards", Icon: "credit-card", Type: "menu", SortOrder: 70},
				{Name: "运行监控", Route: "/system-jobs", Icon: "monitor", Type: "menu", SortOrder: 80},
				{Name: "直跑插件", Route: "/platform-runtime", Icon: "plugin", Type: "menu", SortOrder: 90},
			},
		},
		{
			Name:      "网课管理",
			Icon:      "database",
			Type:      "dir",
			SortOrder: 30,
			Children: []adminMenuSeed{
				{Name: "货源管理", Route: "/connectors", Icon: "api", Type: "menu", SortOrder: 10},
				{Name: "分类管理", Route: "/categories", Icon: "database", Type: "menu", SortOrder: 20},
				{Name: "平台管理", Route: "/classes", Icon: "cluster", Type: "menu", SortOrder: 30},
				{Name: "费率管理", Route: "/special-prices", Icon: "dollar", Type: "menu", SortOrder: 40},
				{Name: "任务管理", Route: "/orders", Icon: "shopping-cart", Type: "menu", SortOrder: 50},
				{Name: "密价管理", Route: "/special-prices", Icon: "dollar", Type: "menu", SortOrder: 60},
				{Name: "推荐下单", Route: "/recommendations", Icon: "star", Type: "menu", SortOrder: 70},
			},
		},
		{Name: "数据统计", Route: "/statistics", Icon: "bar-chart", Type: "menu", SortOrder: 40},
		{Name: "订单提交", Route: "/order-submit", Icon: "plus", Type: "menu", SortOrder: 50},
		{Name: "订单列表", Route: "/orders", Icon: "shopping-cart", Type: "menu", SortOrder: 60},
		{Name: "代理列表", Route: "/agents", Icon: "team", Type: "menu", SortOrder: 70},
		{Name: "充值中心", Route: "/recharge-cards", Icon: "credit-card", Type: "menu", SortOrder: 80},
	}
}
