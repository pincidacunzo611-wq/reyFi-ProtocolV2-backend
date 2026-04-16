-- 000004_create_futures_tables.up.sql

CREATE TABLE IF NOT EXISTS futures_markets (
    id                      BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id                BIGINT          NOT NULL,
    market_address          VARCHAR(42)     NOT NULL,
    market_name             VARCHAR(32)     NOT NULL COMMENT '如 BTC-USD / ETH-USD',
    base_asset              VARCHAR(42)     NOT NULL COMMENT '标的资产',
    quote_asset             VARCHAR(42)     NOT NULL COMMENT '报价资产',
    max_leverage            DECIMAL(10,2)   NOT NULL DEFAULT 20,
    maintenance_margin_rate DECIMAL(20,8)   NOT NULL DEFAULT 0.005 COMMENT '维持保证金率',
    taker_fee_rate          DECIMAL(20,8)   NOT NULL DEFAULT 0.001,
    maker_fee_rate          DECIMAL(20,8)   NOT NULL DEFAULT 0.0005,
    funding_interval        INT             NOT NULL DEFAULT 28800 COMMENT '资金费率结算间隔（秒）',
    is_active               TINYINT(1)      NOT NULL DEFAULT 1,
    created_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_market (chain_id, market_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='永续期货市场配置';

CREATE TABLE IF NOT EXISTS futures_positions (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id          BIGINT          NOT NULL,
    position_id       VARCHAR(66)     NOT NULL COMMENT '链上仓位 ID',
    user_address      VARCHAR(42)     NOT NULL,
    market_address    VARCHAR(42)     NOT NULL,
    side              VARCHAR(8)      NOT NULL COMMENT 'long / short',
    size              DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '仓位大小',
    entry_price       DECIMAL(65,18)  NOT NULL DEFAULT 0,
    mark_price        DECIMAL(65,18)  NOT NULL DEFAULT 0,
    margin            DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '保证金',
    leverage          DECIMAL(20,8)   NOT NULL DEFAULT 1,
    unrealized_pnl    DECIMAL(65,18)  NOT NULL DEFAULT 0,
    realized_pnl      DECIMAL(65,18)  NOT NULL DEFAULT 0,
    liquidation_price DECIMAL(65,18)  NOT NULL DEFAULT 0,
    status            VARCHAR(16)     NOT NULL DEFAULT 'open' COMMENT 'open / closed / liquidated',
    opened_at         DATETIME        NOT NULL,
    closed_at         DATETIME        DEFAULT NULL,
    updated_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_position (chain_id, position_id),
    INDEX idx_user_status (user_address, status),
    INDEX idx_market_status (market_address, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='永续期货仓位';

CREATE TABLE IF NOT EXISTS futures_funding_records (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    market_address  VARCHAR(42)     NOT NULL,
    funding_rate    DECIMAL(20,12)  NOT NULL COMMENT '本期资金费率',
    cumulative_rate DECIMAL(30,12)  NOT NULL DEFAULT 0 COMMENT '累计费率',
    long_pay        DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '多头支付总额',
    short_pay       DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '空头支付总额',
    settlement_time DATETIME        NOT NULL,
    block_number    BIGINT          NOT NULL,
    tx_hash         VARCHAR(66)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_market_time (chain_id, market_address, settlement_time),
    INDEX idx_market_block (market_address, block_number DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资金费率结算记录';

CREATE TABLE IF NOT EXISTS futures_liquidations (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id          BIGINT          NOT NULL,
    position_id       VARCHAR(66)     NOT NULL,
    user_address      VARCHAR(42)     NOT NULL,
    market_address    VARCHAR(42)     NOT NULL,
    liquidator        VARCHAR(42)     NOT NULL,
    side              VARCHAR(8)      NOT NULL,
    size              DECIMAL(65,18)  NOT NULL DEFAULT 0,
    liquidation_price DECIMAL(65,18)  NOT NULL DEFAULT 0,
    penalty           DECIMAL(65,18)  NOT NULL DEFAULT 0,
    block_number      BIGINT          NOT NULL,
    block_time        DATETIME        NOT NULL,
    tx_hash           VARCHAR(66)     NOT NULL,
    log_index         INT             NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_tx_log (chain_id, tx_hash, log_index),
    INDEX idx_user_time (user_address, block_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='永续期货清算记录';
