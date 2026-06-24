package models

import "time"

type User struct {
	ID              uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ParentID        uint       `gorm:"column:parent_id;not null;default:0;index:idx_users_parent_id" json:"parentId"`
	Account         string     `gorm:"column:account;size:64;not null;uniqueIndex:uk_users_account" json:"account"`
	PasswordHash    string     `gorm:"column:password_hash;size:128;not null" json:"-"`
	Name            string     `gorm:"column:name;size:120;not null;default:''" json:"name"`
	Balance         float64    `gorm:"column:balance;type:decimal(12,2);not null;default:0" json:"balance"`
	PriceRate       float64    `gorm:"column:price_rate;type:decimal(10,4);not null;default:1" json:"priceRate"`
	APIKey          string     `gorm:"column:api_key;size:128;not null;default:''" json:"-"`
	InviteCode      string     `gorm:"column:invite_code;size:64;not null;default:'';index:idx_users_invite_code" json:"inviteCode"`
	InvitePriceRate float64    `gorm:"column:invite_price_rate;type:decimal(10,4);not null;default:0" json:"invitePriceRate"`
	Notice          string     `gorm:"column:notice;type:text;not null" json:"notice"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	LastSeenAt      *time.Time `gorm:"column:last_seen_at" json:"lastSeenAt"`
	LastIP          string     `gorm:"column:last_ip;size:64;not null;default:''" json:"lastIp"`
	Role            string     `gorm:"column:role;size:32;not null;default:agent" json:"role"`
	Status          string     `gorm:"column:status;size:32;not null;default:active;index:idx_users_status" json:"status"`
}

func (User) TableName() string {
	return "users"
}

type InviteCode struct {
	ID        uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code      string     `gorm:"column:code;size:64;not null;uniqueIndex:uk_invite_codes_code" json:"code"`
	Note      string     `gorm:"column:note;size:255;not null;default:''" json:"note"`
	MaxUses   int        `gorm:"column:max_uses;not null;default:1" json:"maxUses"`
	UsedCount int        `gorm:"column:used_count;not null;default:0" json:"usedCount"`
	PriceRate float64    `gorm:"column:price_rate;type:decimal(10,4);not null;default:1" json:"priceRate"`
	Status    string     `gorm:"column:status;size:32;not null;default:active;index:idx_invite_codes_status" json:"status"`
	CreatedBy uint       `gorm:"column:created_by;not null;default:0;index:idx_invite_codes_created_by" json:"createdBy"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
	ExpiresAt *time.Time `gorm:"column:expires_at;index:idx_invite_codes_expires_at" json:"expiresAt"`
}

func (InviteCode) TableName() string {
	return "invite_codes"
}

type CourseClass struct {
	ID              uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Sort            int       `gorm:"column:sort;not null;default:10;index:idx_course_classes_sort" json:"sort"`
	Name            string    `gorm:"column:name;size:160;not null" json:"name"`
	QueryParam      string    `gorm:"column:query_param;size:160;not null;default:''" json:"queryParam"`
	DockingCode     string    `gorm:"column:docking_code;size:160;not null;default:''" json:"dockingCode"`
	Price           float64   `gorm:"column:price;type:decimal(12,4);not null;default:0" json:"price"`
	QueryPlatform   string    `gorm:"column:query_platform;size:160;not null;default:''" json:"queryPlatform"`
	DockingPlatform string    `gorm:"column:docking_platform;size:160;not null;default:''" json:"dockingPlatform"`
	PriceOperator   string    `gorm:"column:price_operator;size:8;not null;default:*" json:"priceOperator"`
	Description     string    `gorm:"column:description;size:500;not null;default:''" json:"description"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	Status          string    `gorm:"column:status;size:32;not null;default:online;index:idx_course_classes_status" json:"status"`
	Category        string    `gorm:"column:category;size:64;not null;default:'';index:idx_course_classes_category" json:"category"`
	BridgeEnabled   bool      `gorm:"column:bridge_enabled;not null;default:true" json:"bridgeEnabled"`
}

func (CourseClass) TableName() string {
	return "course_classes"
}

type CourseCategory struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Sort        int       `gorm:"column:sort;not null;default:10;index:idx_course_categories_sort" json:"sort"`
	Name        string    `gorm:"column:name;size:120;not null;uniqueIndex:uk_course_categories_name" json:"name"`
	Status      string    `gorm:"column:status;size:32;not null;default:active;index:idx_course_categories_status" json:"status"`
	Pinned      bool      `gorm:"column:pinned;not null;default:false;index:idx_course_categories_pinned" json:"pinned"`
	Description string    `gorm:"column:description;type:text;not null" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (CourseCategory) TableName() string {
	return "course_categories"
}

type ClassFavorite struct {
	ID        uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"column:user_id;not null;index:idx_class_favorites_user_id;uniqueIndex:uk_class_favorites_user_class,priority:1" json:"userId"`
	ClassID   uint      `gorm:"column:class_id;not null;index:idx_class_favorites_class_id;uniqueIndex:uk_class_favorites_user_class,priority:2" json:"classId"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
}

func (ClassFavorite) TableName() string {
	return "class_favorites"
}

type SpecialPrice struct {
	ID        uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"column:user_id;not null;index:idx_special_prices_user_id;uniqueIndex:uk_special_prices_user_class,priority:1" json:"userId"`
	ClassID   uint      `gorm:"column:class_id;not null;index:idx_special_prices_class_id;uniqueIndex:uk_special_prices_user_class,priority:2" json:"classId"`
	Mode      int       `gorm:"column:mode;not null;default:0" json:"mode"`
	Price     float64   `gorm:"column:price;type:decimal(12,4);not null;default:0" json:"price"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (SpecialPrice) TableName() string {
	return "special_prices"
}

type Order struct {
	ID              uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID          uint      `gorm:"column:user_id;not null;default:0;index:idx_orders_user_id" json:"userId"`
	ClassID         uint      `gorm:"column:class_id;not null;default:0;index:idx_orders_class_id" json:"classId"`
	ConnectorID     uint      `gorm:"column:connector_id;not null;default:0;index:idx_orders_connector_id" json:"connectorId"`
	ExecutionMode   string    `gorm:"column:execution_mode;size:32;not null;default:connector;index:idx_orders_execution_mode" json:"executionMode"`
	PluginCode      string    `gorm:"column:plugin_code;size:64;not null;default:'';index:idx_orders_plugin_code" json:"pluginCode"`
	WorkerID        string    `gorm:"column:worker_id;size:120;not null;default:'';index:idx_orders_worker_id" json:"workerId"`
	ProxyID         uint      `gorm:"column:proxy_id;not null;default:0;index:idx_orders_proxy_id" json:"proxyId"`
	RemoteOrderID   string    `gorm:"column:remote_order_id;size:160;not null;default:'';index:idx_orders_remote_order_id" json:"remoteOrderId"`
	Platform        string    `gorm:"column:platform;size:160;not null;default:''" json:"platform"`
	School          string    `gorm:"column:school;size:160;not null;default:''" json:"school"`
	StudentName     string    `gorm:"column:student_name;size:120;not null;default:''" json:"studentName"`
	Account         string    `gorm:"column:account;size:160;not null;default:'';index:idx_orders_account" json:"account"`
	AccountPassword string    `gorm:"column:account_password;size:160;not null;default:''" json:"-"`
	CourseID        string    `gorm:"column:course_id;type:text;not null" json:"courseId"`
	CourseName      string    `gorm:"column:course_name;size:255;not null;default:'';index:idx_orders_course_name" json:"courseName"`
	Fee             float64   `gorm:"column:fee;type:decimal(12,4);not null;default:0" json:"fee"`
	DockingCode     string    `gorm:"column:docking_code;size:160;not null;default:''" json:"dockingCode"`
	FlashMode       bool      `gorm:"column:flash_mode;not null;default:false;index:idx_orders_flash_status,priority:1" json:"flashMode"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime;index:idx_orders_created_at" json:"createdAt"`
	SourceIP        string    `gorm:"column:source_ip;size:64;not null;default:''" json:"sourceIp"`
	DockingStatus   string    `gorm:"column:docking_status;size:32;not null;default:pending;index:idx_orders_docking_status" json:"dockingStatus"`
	Status          string    `gorm:"column:status;size:32;not null;default:pending;index:idx_orders_status;index:idx_orders_flash_status,priority:2" json:"status"`
	Progress        string    `gorm:"column:progress;size:160;not null;default:''" json:"progress"`
	RetryCount      int       `gorm:"column:retry_count;not null;default:0" json:"retryCount"`
	Remarks         string    `gorm:"column:remarks;size:500;not null;default:''" json:"remarks"`
	Score           string    `gorm:"column:score;size:32;not null;default:''" json:"score"`
	DurationMinutes int       `gorm:"column:duration_minutes;not null;default:0" json:"durationMinutes"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderEvent struct {
	ID            uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID       uint      `gorm:"column:order_id;not null;index:idx_order_events_order_id" json:"orderId"`
	UserID        uint      `gorm:"column:user_id;not null;default:0;index:idx_order_events_user_id" json:"userId"`
	Level         string    `gorm:"column:level;size:32;not null;default:info;index:idx_order_events_level" json:"level"`
	Source        string    `gorm:"column:source;size:64;not null;default:system;index:idx_order_events_source" json:"source"`
	EventType     string    `gorm:"column:event_type;size:80;not null;default:'';index:idx_order_events_event_type" json:"eventType"`
	Content       string    `gorm:"column:content;size:1000;not null;default:''" json:"content"`
	Progress      string    `gorm:"column:progress;size:160;not null;default:''" json:"progress"`
	VisibleToUser bool      `gorm:"column:visible_to_user;not null;default:true;index:idx_order_events_visible" json:"visibleToUser"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;autoCreateTime;index:idx_order_events_created_at" json:"createdAt"`
}

func (OrderEvent) TableName() string {
	return "order_events"
}

type WorkOrder struct {
	ID            uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID        uint      `gorm:"column:user_id;not null;default:0;index:idx_work_orders_user_id" json:"userId"`
	Category      string    `gorm:"column:category;size:80;not null;default:'';index:idx_work_orders_category" json:"category"`
	Title         string    `gorm:"column:title;size:160;not null;default:''" json:"title"`
	Content       string    `gorm:"column:content;type:text;not null" json:"content"`
	Answer        string    `gorm:"column:answer;type:text;not null" json:"answer"`
	Status        string    `gorm:"column:status;size:32;not null;default:'待回复';index:idx_work_orders_status" json:"status"`
	Progress      int       `gorm:"column:progress;not null;default:0" json:"progress"`
	AttachmentURL string    `gorm:"column:attachment_url;size:500;not null;default:''" json:"attachmentUrl"`
	UserVisible   bool      `gorm:"column:user_visible;not null;default:true;index:idx_work_orders_user_visible" json:"userVisible"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;autoCreateTime;index:idx_work_orders_created_at" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (WorkOrder) TableName() string {
	return "work_orders"
}

type RechargeCard struct {
	ID        uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code      string     `gorm:"column:code;size:64;not null;uniqueIndex:uk_recharge_cards_code" json:"code"`
	Amount    float64    `gorm:"column:amount;type:decimal(12,2);not null;default:0" json:"amount"`
	Status    string     `gorm:"column:status;size:32;not null;default:unused;index:idx_recharge_cards_status" json:"status"`
	UserID    uint       `gorm:"column:user_id;not null;default:0;index:idx_recharge_cards_user_id" json:"userId"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UsedAt    *time.Time `gorm:"column:used_at;index:idx_recharge_cards_used_at" json:"usedAt"`
}

func (RechargeCard) TableName() string {
	return "recharge_cards"
}

type RecommendedClass struct {
	ID        uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ClassID   uint      `gorm:"column:class_id;not null;index:idx_recommended_classes_class_id" json:"classId"`
	Title     string    `gorm:"column:title;size:160;not null;default:''" json:"title"`
	Note      string    `gorm:"column:note;type:text;not null" json:"note"`
	SortOrder int       `gorm:"column:sort_order;not null;default:10;index:idx_recommended_classes_sort_order" json:"sortOrder"`
	Visible   bool      `gorm:"column:visible;not null;default:true;index:idx_recommended_classes_visible" json:"visible"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (RecommendedClass) TableName() string {
	return "recommended_classes"
}

type Connector struct {
	ID                     uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name                   string    `gorm:"column:name;size:120;not null" json:"name"`
	BaseURL                string    `gorm:"column:base_url;size:500;not null;default:''" json:"baseUrl"`
	AppKey                 string    `gorm:"column:app_key;size:160;not null;default:''" json:"appKey"`
	AppSecret              string    `gorm:"column:app_secret;size:255;not null;default:''" json:"-"`
	Kind                   string    `gorm:"column:kind;size:64;not null;default:generic;index:idx_connectors_kind" json:"kind"`
	Status                 string    `gorm:"column:status;size:32;not null;default:active;index:idx_connectors_status" json:"status"`
	TimeoutMS              int       `gorm:"column:timeout_ms;not null;default:8000" json:"timeoutMs"`
	SortOrder              int       `gorm:"column:sort_order;not null;default:10;index:idx_connectors_sort_order" json:"sortOrder"`
	OrderSyncEnabled       bool      `gorm:"column:order_sync_enabled;not null;default:true" json:"orderSyncEnabled"`
	SourceSyncEnabled      bool      `gorm:"column:source_sync_enabled;not null;default:true" json:"sourceSyncEnabled"`
	PriceMode              string    `gorm:"column:price_mode;size:32;not null;default:multiplier" json:"priceMode"`
	PriceValue             float64   `gorm:"column:price_value;type:decimal(12,4);not null;default:1.0000" json:"priceValue"`
	PriceRounding          string    `gorm:"column:price_rounding;size:32;not null;default:none" json:"priceRounding"`
	ReplaceRulesJSON       string    `gorm:"column:replace_rules_json;type:text;not null" json:"replaceRulesJson"`
	CategoryPriceRulesJSON string    `gorm:"column:category_price_rules_json;type:text;not null" json:"categoryPriceRulesJson"`
	CreatedAt              time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
}

func (Connector) TableName() string {
	return "connectors"
}

type PlatformPlugin struct {
	Code            string    `gorm:"column:code;size:64;primaryKey" json:"code"`
	Name            string    `gorm:"column:name;size:120;not null" json:"name"`
	Description     string    `gorm:"column:description;size:500;not null;default:''" json:"description"`
	Status          string    `gorm:"column:status;size:32;not null;default:disabled;index:idx_platform_plugins_status" json:"status"`
	SortOrder       int       `gorm:"column:sort_order;not null;default:10;index:idx_platform_plugins_sort_order" json:"sortOrder"`
	SupportsQuery   bool      `gorm:"column:supports_query;not null;default:false" json:"supportsQuery"`
	SupportsSubmit  bool      `gorm:"column:supports_submit;not null;default:true" json:"supportsSubmit"`
	SupportsRefresh bool      `gorm:"column:supports_refresh;not null;default:true" json:"supportsRefresh"`
	MaxConcurrency  int       `gorm:"column:max_concurrency;not null;default:2" json:"maxConcurrency"`
	AccountSerial   bool      `gorm:"column:account_serial;not null;default:true" json:"accountSerial"`
	ConfigJSON      string    `gorm:"column:config_json;type:text;not null" json:"configJson"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (PlatformPlugin) TableName() string {
	return "platform_plugins"
}

type WorkerNode struct {
	ID                uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WorkerID          string     `gorm:"column:worker_id;size:120;not null;uniqueIndex:uk_worker_nodes_worker_id" json:"workerId"`
	Hostname          string     `gorm:"column:hostname;size:160;not null;default:''" json:"hostname"`
	Status            string     `gorm:"column:status;size:32;not null;default:stopped;index:idx_worker_nodes_status" json:"status"`
	AcceptNew         bool       `gorm:"column:accept_new;not null;default:true" json:"acceptNew"`
	MaxConcurrency    int        `gorm:"column:max_concurrency;not null;default:1" json:"maxConcurrency"`
	RunningCount      int        `gorm:"column:running_count;not null;default:0" json:"runningCount"`
	CurrentOrderID    uint       `gorm:"column:current_order_id;not null;default:0;index:idx_worker_nodes_current_order_id" json:"currentOrderId"`
	CurrentPluginCode string     `gorm:"column:current_plugin_code;size:64;not null;default:''" json:"currentPluginCode"`
	Message           string     `gorm:"column:message;size:500;not null;default:''" json:"message"`
	StartedAt         *time.Time `gorm:"column:started_at" json:"startedAt"`
	HeartbeatAt       *time.Time `gorm:"column:heartbeat_at;index:idx_worker_nodes_heartbeat_at" json:"heartbeatAt"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (WorkerNode) TableName() string {
	return "worker_nodes"
}

type WorkerCommand struct {
	ID         uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WorkerID   string     `gorm:"column:worker_id;size:120;not null;default:'';index:idx_worker_commands_worker_id" json:"workerId"`
	Command    string     `gorm:"column:command;size:64;not null;default:'';index:idx_worker_commands_command" json:"command"`
	Status     string     `gorm:"column:status;size:32;not null;default:pending;index:idx_worker_commands_status" json:"status"`
	Result     string     `gorm:"column:result;size:500;not null;default:''" json:"result"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;autoCreateTime;index:idx_worker_commands_created_at" json:"createdAt"`
	ExecutedAt *time.Time `gorm:"column:executed_at" json:"executedAt"`
}

func (WorkerCommand) TableName() string {
	return "worker_commands"
}

type WorkerProxy struct {
	ID             uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name           string     `gorm:"column:name;size:120;not null;default:''" json:"name"`
	ProxyURL       string     `gorm:"column:proxy_url;size:500;not null;default:''" json:"-"`
	Kind           string     `gorm:"column:kind;size:32;not null;default:http" json:"kind"`
	Status         string     `gorm:"column:status;size:32;not null;default:active;index:idx_worker_proxies_status" json:"status"`
	MaxConcurrency int        `gorm:"column:max_concurrency;not null;default:1" json:"maxConcurrency"`
	InUseCount     int        `gorm:"column:in_use_count;not null;default:0;index:idx_worker_proxies_in_use_count" json:"inUseCount"`
	UseCount       int        `gorm:"column:use_count;not null;default:0" json:"useCount"`
	SuccessCount   int        `gorm:"column:success_count;not null;default:0" json:"successCount"`
	FailCount      int        `gorm:"column:fail_count;not null;default:0" json:"failCount"`
	LastUsedAt     *time.Time `gorm:"column:last_used_at;index:idx_worker_proxies_last_used_at" json:"lastUsedAt"`
	LastError      string     `gorm:"column:last_error;size:500;not null;default:''" json:"lastError"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (WorkerProxy) TableName() string {
	return "worker_proxies"
}

type AdminMenu struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ParentID   uint      `gorm:"column:parent_id;not null;default:0;index:idx_admin_menus_parent_id" json:"parentId"`
	Name       string    `gorm:"column:name;size:80;not null" json:"name"`
	Route      string    `gorm:"column:route;size:160;not null;default:''" json:"route"`
	Icon       string    `gorm:"column:icon;size:80;not null;default:''" json:"icon"`
	Type       string    `gorm:"column:type;size:20;not null;default:menu;index:idx_admin_menus_type" json:"type"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:10;index:idx_admin_menus_sort_order" json:"sortOrder"`
	Visible    bool      `gorm:"column:visible;not null;default:true;index:idx_admin_menus_visible" json:"visible"`
	Permission string    `gorm:"column:permission;size:80;not null;default:''" json:"permission"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (AdminMenu) TableName() string {
	return "admin_menus"
}

type OperationLog struct {
	ID        uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"column:user_id;not null;default:0;index:idx_operation_logs_user_id" json:"userId"`
	Type      string    `gorm:"column:type;size:80;not null;default:'';index:idx_operation_logs_type" json:"type"`
	Text      string    `gorm:"column:text;type:text;not null" json:"text"`
	Amount    float64   `gorm:"column:amount;type:decimal(12,4);not null;default:0" json:"amount"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"createdAt"`
	SourceIP  string    `gorm:"column:source_ip;size:64;not null;default:''" json:"sourceIp"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}

type SystemJob struct {
	Name            string     `gorm:"column:name;size:80;primaryKey" json:"name"`
	Status          string     `gorm:"column:status;size:32;not null;default:idle;index:idx_system_jobs_status" json:"status"`
	Enabled         bool       `gorm:"column:enabled;not null;default:true;index:idx_system_jobs_enabled" json:"enabled"`
	LastStartedAt   *time.Time `gorm:"column:last_started_at" json:"lastStartedAt"`
	LastFinishedAt  *time.Time `gorm:"column:last_finished_at" json:"lastFinishedAt"`
	LastDurationMS  int64      `gorm:"column:last_duration_ms;not null;default:0" json:"lastDurationMs"`
	LastError       string     `gorm:"column:last_error;type:text;not null" json:"lastError"`
	LastSummaryJSON string     `gorm:"column:last_summary_json;type:text;not null" json:"lastSummaryJson"`
	HeartbeatAt     *time.Time `gorm:"column:heartbeat_at;index:idx_system_jobs_heartbeat_at" json:"heartbeatAt"`
}

func (SystemJob) TableName() string {
	return "system_jobs"
}

type SiteConfig struct {
	Key   string `gorm:"column:key;size:120;primaryKey" json:"key"`
	Value string `gorm:"column:value;type:text;not null" json:"value"`
}

func (SiteConfig) TableName() string {
	return "site_configs"
}
