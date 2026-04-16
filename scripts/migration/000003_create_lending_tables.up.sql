-- 000003_create_lending_tables.up.sql

CREATE TABLE IF NOT EXISTS lending_markets (
    id                      BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id                BIGINT          NOT NULL,
    asset_address           VARCHAR(42)     NOT NULL,
    asset_symbol            VARCHAR(32)     NOT NULL DEFAULT '',
    asset_decimals          TINYINT UNSIGNED NOT NULL DEFAULT 18,
    rtoken_address          VARCHAR(42)     NOT NULL COMMENT '存款凭证代币',
    debt_token_address      VARCHAR(42)     NOT NULL COMMENT '债务代币',
    collateral_factor       DECIMAL(20,8)   NOT NULL DEFAULT 0 COMMENT '抵押系数 (如 0.75)',
    liquidation_threshold   DECIMAL(20,8)   NOT NULL DEFAULT 0 COMMENT '清算阈值 (如 0.80)',
    liquidation_penalty     DECIMAL(20,8)   NOT NULL DEFAULT 0 COMMENT '清算罚金 (如 0.05)',
    reserve_factor          DECIMAL(20,8)   NOT NULL DEFAULT 0 COMMENT '储备系数',
    is_active               TINYINT(1)      NOT NULL DEFAULT 1,
    created_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_asset (chain_id, asset_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='借贷市场配置';

CREATE TABLE IF NOT EXISTS lending_market_snapshots (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id         BIGINT          NOT NULL,
    asset_address    VARCHAR(42)     NOT NULL,
    total_supply     DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '总存款',
    total_borrow     DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '总借款',
    utilization_rate DECIMAL(20,8)   NOT NULL DEFAULT 0,
    supply_apr       DECIMAL(20,8)   NOT NULL DEFAULT 0,
    borrow_apr       DECIMAL(20,8)   NOT NULL DEFAULT 0,
    tvl_usd          DECIMAL(65,18)  NOT NULL DEFAULT 0,
    snapshot_time    DATETIME        NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_asset_time (asset_address, snapshot_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='借贷市场快照';

CREATE TABLE IF NOT EXISTS lending_user_positions (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id           BIGINT          NOT NULL,
    user_address       VARCHAR(42)     NOT NULL,
    asset_address      VARCHAR(42)     NOT NULL,
    supplied_amount    DECIMAL(65,18)  NOT NULL DEFAULT 0,
    borrowed_amount    DECIMAL(65,18)  NOT NULL DEFAULT 0,
    collateral_enabled TINYINT(1)      NOT NULL DEFAULT 0,
    health_factor      DECIMAL(20,8)   NOT NULL DEFAULT 0,
    updated_at         DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_user_asset (chain_id, user_address, asset_address),
    INDEX idx_health (health_factor)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户借贷头寸';

CREATE TABLE IF NOT EXISTS lending_liquidations (
    id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id             BIGINT          NOT NULL,
    borrower_address     VARCHAR(42)     NOT NULL COMMENT '被清算用户',
    liquidator_address   VARCHAR(42)     NOT NULL COMMENT '清算人',
    collateral_asset     VARCHAR(42)     NOT NULL COMMENT '被没收的抵押资产',
    debt_asset           VARCHAR(42)     NOT NULL COMMENT '被偿还的债务资产',
    collateral_amount    DECIMAL(65,18)  NOT NULL DEFAULT 0,
    debt_amount          DECIMAL(65,18)  NOT NULL DEFAULT 0,
    penalty_amount       DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '罚金',
    health_factor_before DECIMAL(20,8)   NOT NULL DEFAULT 0,
    block_number         BIGINT          NOT NULL,
    block_time           DATETIME        NOT NULL,
    tx_hash              VARCHAR(66)     NOT NULL,
    log_index            INT             NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_tx_log (chain_id, tx_hash, log_index),
    INDEX idx_borrower_time (borrower_address, block_time DESC),
    INDEX idx_liquidator_time (liquidator_address, block_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='清算记录';
