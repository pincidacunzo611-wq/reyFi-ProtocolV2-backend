-- 000009_create_user_tables.up.sql

CREATE TABLE IF NOT EXISTS users (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    wallet_address  VARCHAR(42)     NOT NULL,
    nickname        VARCHAR(64)     DEFAULT NULL,
    avatar_url      VARCHAR(255)    DEFAULT NULL,
    status          VARCHAR(16)     NOT NULL DEFAULT 'active' COMMENT 'active / banned',
    last_login_at   DATETIME        DEFAULT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_wallet (wallet_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

CREATE TABLE IF NOT EXISTS user_nonces (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    wallet_address  VARCHAR(42)     NOT NULL,
    nonce           VARCHAR(128)    NOT NULL,
    is_used         TINYINT(1)      NOT NULL DEFAULT 0,
    expires_at      DATETIME        NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_wallet_used (wallet_address, is_used),
    INDEX idx_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='登录 Nonce';

CREATE TABLE IF NOT EXISTS user_sessions (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    wallet_address  VARCHAR(42)     NOT NULL,
    refresh_token   VARCHAR(255)    NOT NULL,
    user_agent      VARCHAR(512)    DEFAULT NULL,
    ip_address      VARCHAR(45)     DEFAULT NULL,
    expires_at      DATETIME        NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_user (user_id),
    INDEX idx_refresh (refresh_token),
    INDEX idx_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户会话';

CREATE TABLE IF NOT EXISTS user_asset_snapshots (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    wallet_address  VARCHAR(42)     NOT NULL,
    total_asset_usd DECIMAL(65,18)  NOT NULL DEFAULT 0,
    total_debt_usd  DECIMAL(65,18)  NOT NULL DEFAULT 0,
    net_value_usd   DECIMAL(65,18)  NOT NULL DEFAULT 0,
    pnl_24h         DECIMAL(65,18)  NOT NULL DEFAULT 0,
    allocation_json JSON            DEFAULT NULL COMMENT '各模块资产分布',
    snapshot_time   DATETIME        NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_wallet_time (wallet_address, snapshot_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户资产快照';

CREATE TABLE IF NOT EXISTS user_activity_stream (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    wallet_address  VARCHAR(42)     NOT NULL,
    module          VARCHAR(32)     NOT NULL COMMENT '来源模块',
    action          VARCHAR(32)     NOT NULL COMMENT 'swap / deposit / borrow / vote 等',
    summary         VARCHAR(255)    NOT NULL COMMENT '人类可读摘要',
    detail_json     JSON            DEFAULT NULL COMMENT '详细参数',
    amount_usd      DECIMAL(65,18)  DEFAULT NULL,
    tx_hash         VARCHAR(66)     DEFAULT NULL,
    block_time      DATETIME        NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_wallet_time (wallet_address, block_time DESC),
    INDEX idx_module_action (module, action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户活动流';
