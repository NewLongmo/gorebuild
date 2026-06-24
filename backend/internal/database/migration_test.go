package database

import (
	"os"
	"strings"
	"testing"
)

func TestInitialMigrationContainsRuntimeCriticalSchema(t *testing.T) {
	data, err := os.ReadFile("../../migrations/001_init_schema.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(data)

	requiredFragments := []string{
		"CREATE TABLE IF NOT EXISTS `users`",
		"UNIQUE KEY `uk_users_account` (`account`)",
		"`invite_price_rate` decimal(10,4) NOT NULL DEFAULT 0.0000",
		"KEY `idx_users_invite_code` (`invite_code`)",
		"CREATE TABLE IF NOT EXISTS `invite_codes`",
		"UNIQUE KEY `uk_invite_codes_code` (`code`)",
		"KEY `idx_invite_codes_status` (`status`)",
		"CREATE TABLE IF NOT EXISTS `orders`",
		"`pinned` tinyint(1) NOT NULL DEFAULT 0",
		"CREATE TABLE IF NOT EXISTS `class_favorites`",
		"UNIQUE KEY `uk_class_favorites_user_class` (`user_id`, `class_id`)",
		"`connector_id` bigint unsigned NOT NULL DEFAULT 0",
		"`execution_mode` varchar(32) NOT NULL DEFAULT 'connector'",
		"`plugin_code` varchar(64) NOT NULL DEFAULT ''",
		"`worker_id` varchar(120) NOT NULL DEFAULT ''",
		"`proxy_id` bigint unsigned NOT NULL DEFAULT 0",
		"`flash_mode` tinyint(1) NOT NULL DEFAULT 0",
		"`docking_status` varchar(32) NOT NULL DEFAULT 'pending'",
		"`status` varchar(32) NOT NULL DEFAULT 'pending'",
		"`retry_count` int NOT NULL DEFAULT 0",
		"KEY `idx_orders_connector_id` (`connector_id`)",
		"KEY `idx_orders_flash_status` (`flash_mode`, `status`)",
		"KEY `idx_orders_docking_status` (`docking_status`)",
		"CREATE TABLE IF NOT EXISTS `order_events`",
		"`visible_to_user` tinyint(1) NOT NULL DEFAULT 1",
		"KEY `idx_order_events_order_id` (`order_id`)",
		"`attachment_url` varchar(500) NOT NULL DEFAULT ''",
		"`user_visible` tinyint(1) NOT NULL DEFAULT 1",
		"KEY `idx_work_orders_user_visible` (`user_visible`)",
		"CREATE TABLE IF NOT EXISTS `recommended_classes`",
		"KEY `idx_recommended_classes_visible` (`visible`)",
		"CREATE TABLE IF NOT EXISTS `connectors`",
		"`base_url` varchar(500) NOT NULL DEFAULT ''",
		"`timeout_ms` int NOT NULL DEFAULT 8000",
		"`order_sync_enabled` tinyint(1) NOT NULL DEFAULT 1",
		"`source_sync_enabled` tinyint(1) NOT NULL DEFAULT 1",
		"`price_mode` varchar(32) NOT NULL DEFAULT 'multiplier'",
		"`replace_rules_json` text NOT NULL",
		"CREATE TABLE IF NOT EXISTS `platform_plugins`",
		"`supports_query` tinyint(1) NOT NULL DEFAULT 0",
		"`account_serial` tinyint(1) NOT NULL DEFAULT 1",
		"CREATE TABLE IF NOT EXISTS `worker_nodes`",
		"UNIQUE KEY `uk_worker_nodes_worker_id` (`worker_id`)",
		"CREATE TABLE IF NOT EXISTS `worker_commands`",
		"KEY `idx_worker_commands_status` (`status`)",
		"CREATE TABLE IF NOT EXISTS `worker_proxies`",
		"`proxy_url` varchar(500) NOT NULL DEFAULT ''",
		"KEY `idx_worker_proxies_status` (`status`)",
		"CREATE TABLE IF NOT EXISTS `admin_menus`",
		"`parent_id` bigint unsigned NOT NULL DEFAULT 0",
		"`visible` tinyint(1) NOT NULL DEFAULT 1",
		"CREATE TABLE IF NOT EXISTS `system_jobs`",
		"`last_summary_json` text NOT NULL",
		"KEY `idx_system_jobs_enabled` (`enabled`)",
		"CREATE TABLE IF NOT EXISTS `site_configs`",
		"PRIMARY KEY (`key`)",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing required fragment: %s", fragment)
		}
	}
}
