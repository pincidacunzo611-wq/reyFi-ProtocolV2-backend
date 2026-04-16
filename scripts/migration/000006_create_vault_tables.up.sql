-- 000006_create_vault_tables.up.sql

CREATE TABLE IF NOT EXISTS vaults (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    vault_address   VARCHAR(42)     NOT NULL,
    asset_address   VARCHAR(42)     NOT NULL COMMENT '申赎资产',
    asset_symbol    VARCHAR(32)     NOT NULL DEFAULT '',
    strategy_type   VARCHAR(32)     NOT NULL COMMENT '策略类型',
    name            VARCHAR(64)     NOT NULL,
    symbol          VARCHAR(32)     NOT NULL,
    is_active       TINYINT(1)      NOT NULL DEFAULT 1,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_vault (chain_id, vault_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='金库信息';

CREATE TABLE IF NOT EXISTS vault_snapshots (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id       BIGINT          NOT NULL,
    vault_address  VARCHAR(42)     NOT NULL,
    total_assets   DECIMAL(65,18)  NOT NULL DEFAULT 0,
    total_shares   DECIMAL(65,18)  NOT NULL DEFAULT 0,
    nav            DECIMAL(30,18)  NOT NULL DEFAULT 0 COMMENT '每份额净值',
    tvl_usd        DECIMAL(65,18)  NOT NULL DEFAULT 0,
    apr_7d         DECIMAL(20,8)   DEFAULT NULL,
    apr_30d        DECIMAL(20,8)   DEFAULT NULL,
    max_drawdown   DECIMAL(20,8)   DEFAULT NULL,
    snapshot_time  DATETIME        NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_vault_time (vault_address, snapshot_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='金库净值快照';

CREATE TABLE IF NOT EXISTS vault_user_positions (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    user_address    VARCHAR(42)     NOT NULL,
    vault_address   VARCHAR(42)     NOT NULL,
    shares          DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '持有份额',
    cost_basis      DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '成本基础（USD）',
    deposited_total DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '累计申购',
    withdrawn_total DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '累计赎回',
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_user_vault (chain_id, user_address, vault_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户金库持仓';

CREATE TABLE IF NOT EXISTS vault_harvest_records (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id       BIGINT          NOT NULL,
    vault_address  VARCHAR(42)     NOT NULL,
    strategy       VARCHAR(64)     NOT NULL COMMENT '策略名称',
    profit         DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '本次收益',
    profit_usd     DECIMAL(65,18)  NOT NULL DEFAULT 0,
    nav_before     DECIMAL(30,18)  NOT NULL DEFAULT 0,
    nav_after      DECIMAL(30,18)  NOT NULL DEFAULT 0,
    block_number   BIGINT          NOT NULL,
    block_time     DATETIME        NOT NULL,
    tx_hash        VARCHAR(66)     NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_vault_time (vault_address, block_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='金库收割记录';
