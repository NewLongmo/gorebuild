package repository

import (
	"context"
	"time"

	"dw0rdwk/backend/internal/models"

	"gorm.io/gorm"
)

type DashboardRepository struct {
	db *gorm.DB
}

type DashboardStats struct {
	Users             int64 `json:"users"`
	Classes           int64 `json:"classes"`
	Orders            int64 `json:"orders"`
	Pending           int64 `json:"pending"`
	FlashOrders       int64 `json:"flashOrders"`
	FlashPending      int64 `json:"flashPending"`
	QueueOrders       int64 `json:"queueOrders"`
	QueueRefreshes    int64 `json:"queueRefreshes"`
	QueueSubmit       int64 `json:"queueSubmit"`
	QueueSubmitFlash  int64 `json:"queueSubmitFlash"`
	QueueRefresh      int64 `json:"queueRefresh"`
	QueueRefreshFlash int64 `json:"queueRefreshFlash"`
	ActiveUsers       int64 `json:"activeUsers"`
	OnlineClasses     int64 `json:"onlineClasses"`
}

type DashboardStatistics struct {
	Summary       DashboardStatisticsSummary `json:"summary"`
	Trend7        []DashboardTrendPoint      `json:"trend7"`
	Trend30       []DashboardTrendPoint      `json:"trend30"`
	UserOrderRank []DashboardRankRow         `json:"userOrderRank"`
	PlatformRank  []DashboardRankRow         `json:"platformRank"`
	RechargeRank  []DashboardRankRow         `json:"rechargeRank"`
	InviteRank    []DashboardRankRow         `json:"inviteRank"`
	SourceStats   []DashboardSourceStat      `json:"sourceStats"`
	GeneratedAt   time.Time                  `json:"generatedAt"`
}

type DashboardStatisticsSummary struct {
	TotalUsers       int64   `json:"totalUsers"`
	TodayNewUsers    int64   `json:"todayNewUsers"`
	TotalOrders      int64   `json:"totalOrders"`
	TodayOrders      int64   `json:"todayOrders"`
	PendingOrders    int64   `json:"pendingOrders"`
	DoneOrders       int64   `json:"doneOrders"`
	FailedOrders     int64   `json:"failedOrders"`
	OnlineClasses    int64   `json:"onlineClasses"`
	ActiveConnectors int64   `json:"activeConnectors"`
	AgentBalance     float64 `json:"agentBalance"`
	TodayRecharge    float64 `json:"todayRecharge"`
	TotalRecharge    float64 `json:"totalRecharge"`
	TodaySpend       float64 `json:"todaySpend"`
	TotalSpend       float64 `json:"totalSpend"`
	TodayRevenue     float64 `json:"todayRevenue"`
	TotalRevenue     float64 `json:"totalRevenue"`
	TodayProfit      float64 `json:"todayProfit"`
	TotalProfit      float64 `json:"totalProfit"`
}

type DashboardTrendPoint struct {
	Date     string  `json:"date"`
	Orders   int64   `json:"orders"`
	Revenue  float64 `json:"revenue"`
	Recharge float64 `json:"recharge"`
	Spend    float64 `json:"spend"`
	Profit   float64 `json:"profit"`
}

type DashboardRankRow struct {
	ID     uint    `json:"id"`
	Name   string  `json:"name"`
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

type DashboardSourceStat struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	Orders      int64  `json:"orders"`
	TodayOrders int64  `json:"todayOrders"`
}

func (r *DashboardRepository) Stats(ctx context.Context) (DashboardStats, error) {
	var stats DashboardStats
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&stats.Users).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("status = ?", "active").Count(&stats.ActiveUsers).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.CourseClass{}).Count(&stats.Classes).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.CourseClass{}).Where("status = ?", "online").Count(&stats.OnlineClasses).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Count(&stats.Orders).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("status IN ?", []string{"pending", "queued", "processing"}).Count(&stats.Pending).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("flash_mode = ?", true).Count(&stats.FlashOrders).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("flash_mode = ? AND status IN ?", true, []string{"pending", "queued", "processing"}).Count(&stats.FlashPending).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

func (r *DashboardRepository) Statistics(ctx context.Context) (DashboardStatistics, error) {
	now := time.Now()
	today := startOfDay(now)
	var result DashboardStatistics
	result.GeneratedAt = now
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&result.Summary.TotalUsers).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("created_at >= ?", today).Count(&result.Summary.TodayNewUsers).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Count(&result.Summary.TotalOrders).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("created_at >= ?", today).Count(&result.Summary.TodayOrders).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("status IN ?", []string{"pending", "queued", "processing"}).Count(&result.Summary.PendingOrders).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("status = ?", "done").Count(&result.Summary.DoneOrders).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("status = ?", "failed").Count(&result.Summary.FailedOrders).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&models.CourseClass{}).Where("status = ?", "online").Count(&result.Summary.OnlineClasses).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Connector{}).Where("status = ?", "active").Count(&result.Summary.ActiveConnectors).Error; err != nil {
		return result, err
	}
	var err error
	if result.Summary.AgentBalance, err = r.sumFloat(ctx, &models.User{}, "COALESCE(SUM(balance), 0)", "role <> ?", "admin"); err != nil {
		return result, err
	}
	if result.Summary.TodayRevenue, err = r.sumFloat(ctx, &models.Order{}, "COALESCE(SUM(fee), 0)", "created_at >= ?", today); err != nil {
		return result, err
	}
	if result.Summary.TotalRevenue, err = r.sumFloat(ctx, &models.Order{}, "COALESCE(SUM(fee), 0)", "1 = ?", 1); err != nil {
		return result, err
	}
	if result.Summary.TodayRecharge, err = r.sumFloat(ctx, &models.OperationLog{}, "COALESCE(SUM(amount), 0)", "amount > 0 AND created_at >= ?", today); err != nil {
		return result, err
	}
	if result.Summary.TotalRecharge, err = r.sumFloat(ctx, &models.OperationLog{}, "COALESCE(SUM(amount), 0)", "amount > 0", nil); err != nil {
		return result, err
	}
	if result.Summary.TodaySpend, err = r.sumFloat(ctx, &models.OperationLog{}, "COALESCE(SUM(-amount), 0)", "amount < 0 AND created_at >= ?", today); err != nil {
		return result, err
	}
	if result.Summary.TotalSpend, err = r.sumFloat(ctx, &models.OperationLog{}, "COALESCE(SUM(-amount), 0)", "amount < 0", nil); err != nil {
		return result, err
	}
	result.Summary.TodayProfit = result.Summary.TodayRevenue
	result.Summary.TotalProfit = result.Summary.TotalRevenue
	if result.Trend7, err = r.trend(ctx, now, 7); err != nil {
		return result, err
	}
	if result.Trend30, err = r.trend(ctx, now, 30); err != nil {
		return result, err
	}
	if result.UserOrderRank, err = r.userOrderRank(ctx, today, 8); err != nil {
		return result, err
	}
	if result.PlatformRank, err = r.platformRank(ctx, today, 8); err != nil {
		return result, err
	}
	if result.RechargeRank, err = r.rechargeRank(ctx, today, 8); err != nil {
		return result, err
	}
	if result.InviteRank, err = r.inviteRank(ctx, 8); err != nil {
		return result, err
	}
	if result.SourceStats, err = r.sourceStats(ctx, today); err != nil {
		return result, err
	}
	return result, nil
}

func (r *DashboardRepository) sumFloat(ctx context.Context, model any, selectExpr string, where string, args ...any) (float64, error) {
	var total float64
	query := r.db.WithContext(ctx).Model(model).Select(selectExpr)
	if stringsTrim(where) != "" {
		filtered := make([]any, 0, len(args))
		for _, arg := range args {
			if arg != nil {
				filtered = append(filtered, arg)
			}
		}
		query = query.Where(where, filtered...)
	}
	err := query.Scan(&total).Error
	return total, err
}

func (r *DashboardRepository) trend(ctx context.Context, now time.Time, days int) ([]DashboardTrendPoint, error) {
	points := make([]DashboardTrendPoint, 0, days)
	first := startOfDay(now).AddDate(0, 0, -(days - 1))
	for i := 0; i < days; i++ {
		start := first.AddDate(0, 0, i)
		end := start.AddDate(0, 0, 1)
		var point DashboardTrendPoint
		point.Date = start.Format("01-02")
		if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("created_at >= ? AND created_at < ?", start, end).Count(&point.Orders).Error; err != nil {
			return nil, err
		}
		var err error
		if point.Revenue, err = r.sumFloat(ctx, &models.Order{}, "COALESCE(SUM(fee), 0)", "created_at >= ? AND created_at < ?", start, end); err != nil {
			return nil, err
		}
		if point.Recharge, err = r.sumFloat(ctx, &models.OperationLog{}, "COALESCE(SUM(amount), 0)", "amount > 0 AND created_at >= ? AND created_at < ?", start, end); err != nil {
			return nil, err
		}
		if point.Spend, err = r.sumFloat(ctx, &models.OperationLog{}, "COALESCE(SUM(-amount), 0)", "amount < 0 AND created_at >= ? AND created_at < ?", start, end); err != nil {
			return nil, err
		}
		point.Profit = point.Revenue
		points = append(points, point)
	}
	return points, nil
}

func (r *DashboardRepository) userOrderRank(ctx context.Context, since time.Time, limit int) ([]DashboardRankRow, error) {
	rows := []DashboardRankRow{}
	err := r.db.WithContext(ctx).
		Table("orders").
		Select("orders.user_id AS id, COALESCE(users.account, '') AS name, COUNT(*) AS count, COALESCE(SUM(orders.fee), 0) AS amount").
		Joins("LEFT JOIN users ON users.id = orders.user_id").
		Where("orders.created_at >= ?", since).
		Group("orders.user_id, users.account").
		Order("count DESC, amount DESC").
		Limit(limit).
		Scan(&rows).
		Error
	return rows, err
}

func (r *DashboardRepository) platformRank(ctx context.Context, since time.Time, limit int) ([]DashboardRankRow, error) {
	rows := []DashboardRankRow{}
	err := r.db.WithContext(ctx).
		Table("orders").
		Select("0 AS id, platform AS name, COUNT(*) AS count, COALESCE(SUM(fee), 0) AS amount").
		Where("created_at >= ?", since).
		Group("platform").
		Order("count DESC, amount DESC").
		Limit(limit).
		Scan(&rows).
		Error
	return rows, err
}

func (r *DashboardRepository) rechargeRank(ctx context.Context, since time.Time, limit int) ([]DashboardRankRow, error) {
	rows := []DashboardRankRow{}
	err := r.db.WithContext(ctx).
		Table("operation_logs").
		Select("operation_logs.user_id AS id, COALESCE(users.account, '') AS name, COUNT(*) AS count, COALESCE(SUM(operation_logs.amount), 0) AS amount").
		Joins("LEFT JOIN users ON users.id = operation_logs.user_id").
		Where("operation_logs.amount > 0 AND operation_logs.created_at >= ?", since).
		Group("operation_logs.user_id, users.account").
		Order("amount DESC, count DESC").
		Limit(limit).
		Scan(&rows).
		Error
	return rows, err
}

func (r *DashboardRepository) inviteRank(ctx context.Context, limit int) ([]DashboardRankRow, error) {
	rows := []DashboardRankRow{}
	err := r.db.WithContext(ctx).
		Table("users AS child").
		Select("parent.id AS id, parent.account AS name, COUNT(child.id) AS count, 0 AS amount").
		Joins("JOIN users AS parent ON parent.id = child.parent_id").
		Where("child.parent_id <> 0").
		Group("parent.id, parent.account").
		Order("count DESC").
		Limit(limit).
		Scan(&rows).
		Error
	return rows, err
}

func (r *DashboardRepository) sourceStats(ctx context.Context, since time.Time) ([]DashboardSourceStat, error) {
	var connectors []models.Connector
	if err := r.db.WithContext(ctx).Model(&models.Connector{}).Order("sort_order ASC, id ASC").Find(&connectors).Error; err != nil {
		return nil, err
	}
	stats := make([]DashboardSourceStat, 0, len(connectors))
	for _, connector := range connectors {
		item := DashboardSourceStat{ID: connector.ID, Name: connector.Name, Kind: connector.Kind, Status: connector.Status}
		if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("connector_id = ?", connector.ID).Count(&item.Orders).Error; err != nil {
			return nil, err
		}
		if err := r.db.WithContext(ctx).Model(&models.Order{}).Where("connector_id = ? AND created_at >= ?", connector.ID, since).Count(&item.TodayOrders).Error; err != nil {
			return nil, err
		}
		stats = append(stats, item)
	}
	return stats, nil
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func stringsTrim(value string) string {
	if value == "" {
		return ""
	}
	for len(value) > 0 && (value[0] == ' ' || value[0] == '\t' || value[0] == '\n' || value[0] == '\r') {
		value = value[1:]
	}
	for len(value) > 0 {
		last := value[len(value)-1]
		if last != ' ' && last != '\t' && last != '\n' && last != '\r' {
			break
		}
		value = value[:len(value)-1]
	}
	return value
}
