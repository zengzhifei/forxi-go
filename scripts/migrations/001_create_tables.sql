-- Forxi-Go Auth System Database Schema
-- MySQL 8.0+ required

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '用户全局唯一ID（雪花算法）',
    `email` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '邮箱地址',
    `password_hash` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '密码哈希',
    `nickname` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '用户昵称',
    `avatar` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '用户头像URL',
    `bio` TEXT COMMENT '用户简介',
    `role` VARCHAR(20) NOT NULL DEFAULT 'user' COMMENT '用户角色: user/admin/super_admin',
    `email_verified` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '邮箱是否已验证: 0-未验证 1-已验证',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '账号状态: active/inactive/banned',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_id` (`user_id`),
    UNIQUE KEY `uk_email` (`email`),
    KEY `idx_role` (`role`),
    KEY `idx_status` (`status`),
    KEY `idx_email_verified` (`email_verified`),
    KEY `idx_created_at` (`created_at`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 第三方登录关联表
CREATE TABLE IF NOT EXISTS `oauth_accounts` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT 'OAuth账号ID',
    `user_id` BIGINT NOT NULL COMMENT '关联的用户ID（对应users表的user_id）',
    `provider` VARCHAR(50) NOT NULL COMMENT 'OAuth提供商: wechat/github',
    `provider_user_id` VARCHAR(255) NOT NULL COMMENT '第三方平台的用户ID',
    `access_token` TEXT COMMENT '访问令牌',
    `refresh_token` TEXT COMMENT '刷新令牌',
    `expires_at` DATETIME DEFAULT NULL COMMENT '令牌过期时间',
    `raw_data` JSON COMMENT '第三方返回的原始数据',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_provider_userid` (`provider`, `provider_user_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_provider` (`provider`),
    KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='第三方登录关联表';

-- 登录日志表
CREATE TABLE IF NOT EXISTS `login_logs` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '登录日志ID',
    `user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '用户ID（对应users表的user_id）',
    `ip_address` VARCHAR(45) NOT NULL DEFAULT '' COMMENT 'IP地址',
    `user_agent` TEXT COMMENT '用户代理信息',
    `device_type` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '设备类型: mobile/tablet/desktop',
    `login_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
    `login_method` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '登录方式: email/wechat/github',
    `status` VARCHAR(20) NOT NULL DEFAULT 'success' COMMENT '登录状态: success/failed',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_ip_address` (`ip_address`),
    KEY `idx_login_at` (`login_at`),
    KEY `idx_status` (`status`),
    KEY `idx_login_method` (`login_method`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='登录日志表';
