-- 000005_create_options_tables.up.sql

CREATE TABLE IF NOT EXISTS options_markets (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id            BIGINT          NOT NULL,
    market_address      VARCHAR(42)     NOT NULL,
    underlying_address  VARCHAR(42)     NOT NULL COMMENT '标的资产',
    underlying_symbol   VARCHAR(32)     NOT NULL DEFAULT '',
    settlement_asset    VARCHAR(42)     NOT NULL COMMENT '结算资产',
    is_active           TINYINT(1)      NOT NULL DEFAULT 1,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_market (chain_id, market_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='期权市场配置';

CREATE TABLE IF NOT EXISTS options_positions (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id            BIGINT          NOT NULL,
    option_id           VARCHAR(66)     NOT NULL,
    user_address        VARCHAR(42)     NOT NULL,
    market_address      VARCHAR(42)     NOT NULL,
    underlying_address  VARCHAR(42)     NOT NULL,
    strike_price        DECIMAL(65,18)  NOT NULL,
    premium             DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '权利金',
    size                DECIMAL(65,18)  NOT NULL DEFAULT 0,
    option_type         VARCHAR(8)      NOT NULL COMMENT 'call / put',
    expiry_time         DATETIME        NOT NULL,
    settlement_price    DECIMAL(65,18)  DEFAULT NULL COMMENT '结算价（到期后填入）',
    pnl                 DECIMAL(65,18)  DEFAULT NULL COMMENT '盈亏（行权/到期后计算）',
    status              VARCHAR(16)     NOT NULL DEFAULT 'open' COMMENT 'open / exercised / expired / settled',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_option (chain_id, option_id),
    INDEX idx_user_status (user_address, status),
    INDEX idx_expiry (expiry_time, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='期权仓位';

CREATE TABLE IF NOT EXISTS options_settlements (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id         BIGINT          NOT NULL,
    option_id        VARCHAR(66)     NOT NULL,
    user_address     VARCHAR(42)     NOT NULL,
    action           VARCHAR(16)     NOT NULL COMMENT 'exercise / expire / settle',
    settlement_price DECIMAL(65,18)  NOT NULL DEFAULT 0,
    payout_amount    DECIMAL(65,18)  NOT NULL DEFAULT 0,
    block_number     BIGINT          NOT NULL,
    block_time       DATETIME        NOT NULL,
    tx_hash          VARCHAR(66)     NOT NULL,
    log_index        INT             NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_tx_log (chain_id, tx_hash, log_index)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='期权结算/行权记录';

CREATE TABLE IF NOT EXISTS options_vol_surfaces (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id         BIGINT          NOT NULL,
    underlying       VARCHAR(42)     NOT NULL,
    strike_price     DECIMAL(65,18)  NOT NULL,
    expiry_time      DATETIME        NOT NULL,
    implied_vol      DECIMAL(20,8)   NOT NULL COMMENT '隐含波动率',
    historical_vol   DECIMAL(20,8)   DEFAULT NULL COMMENT '历史波动率',
    snapshot_time    DATETIME        NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_underlying_time (underlying, snapshot_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='波动率面快照';
