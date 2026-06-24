package queue

const (
	OrderSubmit       = "queue:orders"
	OrderSubmitFlash  = "queue:orders:flash"
	OrderRefresh      = "queue:order-refresh"
	OrderRefreshFlash = "queue:order-refresh:flash"
)

func SubmitKey(flashMode bool) string {
	if flashMode {
		return OrderSubmitFlash
	}
	return OrderSubmit
}

func RefreshKey(flashMode bool) string {
	if flashMode {
		return OrderRefreshFlash
	}
	return OrderRefresh
}

func SubmitPriorityKeys() []string {
	return []string{OrderSubmitFlash, OrderSubmit}
}

func RefreshPriorityKeys() []string {
	return []string{OrderRefreshFlash, OrderRefresh}
}

func AllKeys() []string {
	return []string{OrderSubmit, OrderSubmitFlash, OrderRefresh, OrderRefreshFlash}
}
