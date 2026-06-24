package repository

import "gorm.io/gorm"

type Registry struct {
	Dashboard       *DashboardRepository
	Health          *HealthRepository
	Users           *UserRepository
	InviteCodes     *InviteCodeRepository
	Classes         *ClassRepository
	Categories      *CategoryRepository
	Favorites       *ClassFavoriteRepository
	SpecialPrices   *SpecialPriceRepository
	Orders          *OrderRepository
	OrderEvents     *OrderEventRepository
	WorkOrders      *WorkOrderRepository
	RechargeCards   *RechargeCardRepository
	Recommendations *RecommendationRepository
	Connectors      *ConnectorRepository
	PlatformPlugins *PlatformPluginRepository
	WorkerNodes     *WorkerNodeRepository
	WorkerCommands  *WorkerCommandRepository
	WorkerProxies   *WorkerProxyRepository
	Menus           *MenuRepository
	Settings        *SettingRepository
	SystemJobs      *SystemJobRepository
	Logs            *LogRepository
}

func NewRegistry(db *gorm.DB) *Registry {
	return &Registry{
		Dashboard:       &DashboardRepository{db: db},
		Health:          &HealthRepository{db: db},
		Users:           &UserRepository{db: db},
		InviteCodes:     &InviteCodeRepository{db: db},
		Classes:         &ClassRepository{db: db},
		Categories:      &CategoryRepository{db: db},
		Favorites:       &ClassFavoriteRepository{db: db},
		SpecialPrices:   &SpecialPriceRepository{db: db},
		Orders:          &OrderRepository{db: db},
		OrderEvents:     &OrderEventRepository{db: db},
		WorkOrders:      &WorkOrderRepository{db: db},
		RechargeCards:   &RechargeCardRepository{db: db},
		Recommendations: &RecommendationRepository{db: db},
		Connectors:      &ConnectorRepository{db: db},
		PlatformPlugins: &PlatformPluginRepository{db: db},
		WorkerNodes:     &WorkerNodeRepository{db: db},
		WorkerCommands:  &WorkerCommandRepository{db: db},
		WorkerProxies:   &WorkerProxyRepository{db: db},
		Menus:           &MenuRepository{db: db},
		Settings:        &SettingRepository{db: db},
		SystemJobs:      &SystemJobRepository{db: db},
		Logs:            &LogRepository{db: db},
	}
}

type Page[T any] struct {
	Items   []T   `json:"items"`
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	PerPage int   `json:"perPage"`
}

type CascadeDeleteResult struct {
	ID                     uint  `json:"id"`
	DeletedClasses         int64 `json:"deletedClasses"`
	DeletedFavorites       int64 `json:"deletedFavorites"`
	DeletedSpecialPrices   int64 `json:"deletedSpecialPrices"`
	DeletedEmptyCategories int64 `json:"deletedEmptyCategories"`
}

func normalizePage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func normalizeIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	result := make([]uint, 0, len(ids))
	seen := map[uint]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
