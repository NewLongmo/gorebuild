CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `parent_id` bigint unsigned NOT NULL DEFAULT 0,
  `account` varchar(64) NOT NULL,
  `password_hash` varchar(128) NOT NULL,
  `name` varchar(120) NOT NULL DEFAULT '',
  `balance` decimal(12,2) NOT NULL DEFAULT 0.00,
  `price_rate` decimal(10,4) NOT NULL DEFAULT 1.0000,
  `api_key` varchar(128) NOT NULL DEFAULT '',
  `invite_code` varchar(64) NOT NULL DEFAULT '',
  `invite_price_rate` decimal(10,4) NOT NULL DEFAULT 0.0000,
  `notice` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_seen_at` datetime NULL,
  `last_ip` varchar(64) NOT NULL DEFAULT '',
  `role` varchar(32) NOT NULL DEFAULT 'agent',
  `status` varchar(32) NOT NULL DEFAULT 'active',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_users_account` (`account`),
  KEY `idx_users_parent_id` (`parent_id`),
  KEY `idx_users_invite_code` (`invite_code`),
  KEY `idx_users_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `course_classes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sort` int NOT NULL DEFAULT 10,
  `name` varchar(160) NOT NULL,
  `query_param` varchar(160) NOT NULL DEFAULT '',
  `docking_code` varchar(160) NOT NULL DEFAULT '',
  `price` decimal(12,4) NOT NULL DEFAULT 0.0000,
  `query_platform` varchar(160) NOT NULL DEFAULT '',
  `docking_platform` varchar(160) NOT NULL DEFAULT '',
  `price_operator` varchar(8) NOT NULL DEFAULT '*',
  `description` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `status` varchar(32) NOT NULL DEFAULT 'online',
  `category` varchar(64) NOT NULL DEFAULT '',
  `bridge_enabled` tinyint(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  KEY `idx_course_classes_status` (`status`),
  KEY `idx_course_classes_category` (`category`),
  KEY `idx_course_classes_sort` (`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `course_categories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sort` int NOT NULL DEFAULT 10,
  `name` varchar(120) NOT NULL,
  `status` varchar(32) NOT NULL DEFAULT 'active',
  `pinned` tinyint(1) NOT NULL DEFAULT 0,
  `description` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_course_categories_name` (`name`),
  KEY `idx_course_categories_sort` (`sort`),
  KEY `idx_course_categories_status` (`status`),
  KEY `idx_course_categories_pinned` (`pinned`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `class_favorites` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `class_id` bigint unsigned NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_class_favorites_user_class` (`user_id`, `class_id`),
  KEY `idx_class_favorites_user_id` (`user_id`),
  KEY `idx_class_favorites_class_id` (`class_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `invite_codes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `note` varchar(255) NOT NULL DEFAULT '',
  `max_uses` int NOT NULL DEFAULT 1,
  `used_count` int NOT NULL DEFAULT 0,
  `price_rate` decimal(10,4) NOT NULL DEFAULT 1.0000,
  `status` varchar(32) NOT NULL DEFAULT 'active',
  `created_by` bigint unsigned NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `expires_at` datetime NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_invite_codes_code` (`code`),
  KEY `idx_invite_codes_status` (`status`),
  KEY `idx_invite_codes_created_by` (`created_by`),
  KEY `idx_invite_codes_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL DEFAULT 0,
  `class_id` bigint unsigned NOT NULL DEFAULT 0,
  `connector_id` bigint unsigned NOT NULL DEFAULT 0,
  `execution_mode` varchar(32) NOT NULL DEFAULT 'connector',
  `plugin_code` varchar(64) NOT NULL DEFAULT '',
  `worker_id` varchar(120) NOT NULL DEFAULT '',
  `proxy_id` bigint unsigned NOT NULL DEFAULT 0,
  `remote_order_id` varchar(160) NOT NULL DEFAULT '',
  `platform` varchar(160) NOT NULL DEFAULT '',
  `school` varchar(160) NOT NULL DEFAULT '',
  `student_name` varchar(120) NOT NULL DEFAULT '',
  `account` varchar(160) NOT NULL DEFAULT '',
  `account_password` varchar(160) NOT NULL DEFAULT '',
  `course_id` text NOT NULL,
  `course_name` varchar(255) NOT NULL DEFAULT '',
  `fee` decimal(12,4) NOT NULL DEFAULT 0.0000,
  `docking_code` varchar(160) NOT NULL DEFAULT '',
  `flash_mode` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `source_ip` varchar(64) NOT NULL DEFAULT '',
  `docking_status` varchar(32) NOT NULL DEFAULT 'pending',
  `status` varchar(32) NOT NULL DEFAULT 'pending',
  `progress` varchar(160) NOT NULL DEFAULT '',
  `retry_count` int NOT NULL DEFAULT 0,
  `remarks` varchar(500) NOT NULL DEFAULT '',
  `score` varchar(32) NOT NULL DEFAULT '',
  `duration_minutes` int NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_orders_user_id` (`user_id`),
  KEY `idx_orders_class_id` (`class_id`),
  KEY `idx_orders_connector_id` (`connector_id`),
  KEY `idx_orders_execution_mode` (`execution_mode`),
  KEY `idx_orders_plugin_code` (`plugin_code`),
  KEY `idx_orders_worker_id` (`worker_id`),
  KEY `idx_orders_proxy_id` (`proxy_id`),
  KEY `idx_orders_remote_order_id` (`remote_order_id`),
  KEY `idx_orders_account` (`account`),
  KEY `idx_orders_status` (`status`),
  KEY `idx_orders_flash_status` (`flash_mode`, `status`),
  KEY `idx_orders_docking_status` (`docking_status`),
  KEY `idx_orders_created_at` (`created_at`),
  KEY `idx_orders_course_name` (`course_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `order_events` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL DEFAULT 0,
  `level` varchar(32) NOT NULL DEFAULT 'info',
  `source` varchar(64) NOT NULL DEFAULT 'system',
  `event_type` varchar(80) NOT NULL DEFAULT '',
  `content` varchar(1000) NOT NULL DEFAULT '',
  `progress` varchar(160) NOT NULL DEFAULT '',
  `visible_to_user` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_order_events_order_id` (`order_id`),
  KEY `idx_order_events_user_id` (`user_id`),
  KEY `idx_order_events_level` (`level`),
  KEY `idx_order_events_source` (`source`),
  KEY `idx_order_events_event_type` (`event_type`),
  KEY `idx_order_events_visible` (`visible_to_user`),
  KEY `idx_order_events_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `special_prices` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `class_id` bigint unsigned NOT NULL,
  `mode` int NOT NULL DEFAULT 0,
  `price` decimal(12,4) NOT NULL DEFAULT 0.0000,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_special_prices_user_class` (`user_id`, `class_id`),
  KEY `idx_special_prices_user_id` (`user_id`),
  KEY `idx_special_prices_class_id` (`class_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `work_orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL DEFAULT 0,
  `category` varchar(80) NOT NULL DEFAULT '',
  `title` varchar(160) NOT NULL DEFAULT '',
  `content` text NOT NULL,
  `answer` text NOT NULL,
  `status` varchar(32) NOT NULL DEFAULT '待回复',
  `progress` int NOT NULL DEFAULT 0,
  `attachment_url` varchar(500) NOT NULL DEFAULT '',
  `user_visible` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_work_orders_user_id` (`user_id`),
  KEY `idx_work_orders_category` (`category`),
  KEY `idx_work_orders_status` (`status`),
  KEY `idx_work_orders_user_visible` (`user_visible`),
  KEY `idx_work_orders_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `recharge_cards` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `amount` decimal(12,2) NOT NULL DEFAULT 0.00,
  `status` varchar(32) NOT NULL DEFAULT 'unused',
  `user_id` bigint unsigned NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `used_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_recharge_cards_code` (`code`),
  KEY `idx_recharge_cards_status` (`status`),
  KEY `idx_recharge_cards_user_id` (`user_id`),
  KEY `idx_recharge_cards_used_at` (`used_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `recommended_classes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `class_id` bigint unsigned NOT NULL,
  `title` varchar(160) NOT NULL DEFAULT '',
  `note` text NOT NULL,
  `sort_order` int NOT NULL DEFAULT 10,
  `visible` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_recommended_classes_class_id` (`class_id`),
  KEY `idx_recommended_classes_sort_order` (`sort_order`),
  KEY `idx_recommended_classes_visible` (`visible`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `connectors` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(120) NOT NULL,
  `base_url` varchar(500) NOT NULL DEFAULT '',
  `app_key` varchar(160) NOT NULL DEFAULT '',
  `app_secret` varchar(255) NOT NULL DEFAULT '',
  `kind` varchar(64) NOT NULL DEFAULT 'generic',
  `status` varchar(32) NOT NULL DEFAULT 'active',
  `timeout_ms` int NOT NULL DEFAULT 8000,
  `sort_order` int NOT NULL DEFAULT 10,
  `order_sync_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `source_sync_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `price_mode` varchar(32) NOT NULL DEFAULT 'multiplier',
  `price_value` decimal(12,4) NOT NULL DEFAULT 1.0000,
  `price_rounding` varchar(32) NOT NULL DEFAULT 'none',
  `replace_rules_json` text NOT NULL,
  `category_price_rules_json` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_connectors_status` (`status`),
  KEY `idx_connectors_kind` (`kind`),
  KEY `idx_connectors_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `platform_plugins` (
  `code` varchar(64) NOT NULL,
  `name` varchar(120) NOT NULL,
  `description` varchar(500) NOT NULL DEFAULT '',
  `status` varchar(32) NOT NULL DEFAULT 'disabled',
  `sort_order` int NOT NULL DEFAULT 10,
  `supports_query` tinyint(1) NOT NULL DEFAULT 0,
  `supports_submit` tinyint(1) NOT NULL DEFAULT 1,
  `supports_refresh` tinyint(1) NOT NULL DEFAULT 1,
  `max_concurrency` int NOT NULL DEFAULT 2,
  `account_serial` tinyint(1) NOT NULL DEFAULT 1,
  `config_json` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`code`),
  KEY `idx_platform_plugins_status` (`status`),
  KEY `idx_platform_plugins_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `worker_nodes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `worker_id` varchar(120) NOT NULL,
  `hostname` varchar(160) NOT NULL DEFAULT '',
  `status` varchar(32) NOT NULL DEFAULT 'stopped',
  `accept_new` tinyint(1) NOT NULL DEFAULT 1,
  `max_concurrency` int NOT NULL DEFAULT 1,
  `running_count` int NOT NULL DEFAULT 0,
  `current_order_id` bigint unsigned NOT NULL DEFAULT 0,
  `current_plugin_code` varchar(64) NOT NULL DEFAULT '',
  `message` varchar(500) NOT NULL DEFAULT '',
  `started_at` datetime NULL,
  `heartbeat_at` datetime NULL,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_worker_nodes_worker_id` (`worker_id`),
  KEY `idx_worker_nodes_status` (`status`),
  KEY `idx_worker_nodes_current_order_id` (`current_order_id`),
  KEY `idx_worker_nodes_heartbeat_at` (`heartbeat_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `worker_commands` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `worker_id` varchar(120) NOT NULL DEFAULT '',
  `command` varchar(64) NOT NULL DEFAULT '',
  `status` varchar(32) NOT NULL DEFAULT 'pending',
  `result` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `executed_at` datetime NULL,
  PRIMARY KEY (`id`),
  KEY `idx_worker_commands_worker_id` (`worker_id`),
  KEY `idx_worker_commands_command` (`command`),
  KEY `idx_worker_commands_status` (`status`),
  KEY `idx_worker_commands_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `worker_proxies` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(120) NOT NULL DEFAULT '',
  `proxy_url` varchar(500) NOT NULL DEFAULT '',
  `kind` varchar(32) NOT NULL DEFAULT 'http',
  `status` varchar(32) NOT NULL DEFAULT 'active',
  `max_concurrency` int NOT NULL DEFAULT 1,
  `in_use_count` int NOT NULL DEFAULT 0,
  `use_count` int NOT NULL DEFAULT 0,
  `success_count` int NOT NULL DEFAULT 0,
  `fail_count` int NOT NULL DEFAULT 0,
  `last_used_at` datetime NULL,
  `last_error` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_worker_proxies_status` (`status`),
  KEY `idx_worker_proxies_in_use_count` (`in_use_count`),
  KEY `idx_worker_proxies_last_used_at` (`last_used_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `admin_menus` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `parent_id` bigint unsigned NOT NULL DEFAULT 0,
  `name` varchar(80) NOT NULL,
  `route` varchar(160) NOT NULL DEFAULT '',
  `icon` varchar(80) NOT NULL DEFAULT '',
  `type` varchar(20) NOT NULL DEFAULT 'menu',
  `sort_order` int NOT NULL DEFAULT 10,
  `visible` tinyint(1) NOT NULL DEFAULT 1,
  `permission` varchar(80) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_admin_menus_parent_id` (`parent_id`),
  KEY `idx_admin_menus_type` (`type`),
  KEY `idx_admin_menus_sort_order` (`sort_order`),
  KEY `idx_admin_menus_visible` (`visible`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL DEFAULT 0,
  `type` varchar(80) NOT NULL DEFAULT '',
  `text` text NOT NULL,
  `amount` decimal(12,4) NOT NULL DEFAULT 0.0000,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `source_ip` varchar(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_operation_logs_user_id` (`user_id`),
  KEY `idx_operation_logs_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_jobs` (
  `name` varchar(80) NOT NULL,
  `status` varchar(32) NOT NULL DEFAULT 'idle',
  `enabled` tinyint(1) NOT NULL DEFAULT 1,
  `last_started_at` datetime NULL,
  `last_finished_at` datetime NULL,
  `last_duration_ms` bigint NOT NULL DEFAULT 0,
  `last_error` text NOT NULL,
  `last_summary_json` text NOT NULL,
  `heartbeat_at` datetime NULL,
  PRIMARY KEY (`name`),
  KEY `idx_system_jobs_status` (`status`),
  KEY `idx_system_jobs_enabled` (`enabled`),
  KEY `idx_system_jobs_heartbeat_at` (`heartbeat_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `site_configs` (
  `key` varchar(120) NOT NULL,
  `value` text NOT NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
