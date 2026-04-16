-- 000002_create_dex_tables.up.sql

CREATE TABLE IF NOT EXISTS dex_pairs (
    id               BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    chain_id         BIGINT           NOT NULL,
    pair_address     VARCHAR(42)      NOT NULL,
    token0_address   VARCHAR(42)      NOT NULL,
    token1_address   VARCHAR(42)      NOT NULL,
    token0_symbol    VARCHAR(32)      NOT NULL DEFAULT '',
    token1_symbol    VARCHAR(32)      NOT NULL DEFAULT '',
    token0_decimals  TINYINT UNSIGNED NOT NULL DEFAULT 18,
    token1_decimals  TINYINT UNSIGNED NOT NULL DEFAULT 18,
    lp_token_address VARCHAR(42)      DEFAULT NULL,
    fee_bps          INT              NOT NULL DEFAULT 30 COMMENT '费率基点，30 = 0.3%',
    created_block    BIGINT           NOT NULL DEFAULT 0,
    is_active        TINYINT(1)       NOT NULL DEFAULT 1,
    created_at       DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_pair (chain_id, pair_address),
    INDEX idx_tokens (token0_address, token1_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='DEX 交易对';

CREATE TABLE IF NOT EXISTS dex_pair_snapshots (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id      BIGINT          NOT NULL,
    pair_address  VARCHAR(42)     NOT NULL,
    reserve0      DECIMAL(65,18)  NOT NULL DEFAULT 0,
    reserve1      DECIMAL(65,18)  NOT NULL DEFAULT 0,
    price0        DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT 'token0 以 token1 计价',
    price1        DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT 'token1 以 token0 计价',
    total_supply  DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT 'LP 总量',
    tvl_usd       DECIMAL(65,18)  NOT NULL DEFAULT 0,
    snapshot_time DATETIME        NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_pair_time (pair_address, snapshot_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='交易对快照';

CREATE TABLE IF NOT EXISTS dex_trades (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    pair_address    VARCHAR(42)     NOT NULL,
    trader_address  VARCHAR(42)     NOT NULL COMMENT '交易发起者（to）',
    sender_address  VARCHAR(42)     NOT NULL COMMENT 'msg.sender（通常是 Router）',
    direction       VARCHAR(8)      NOT NULL COMMENT 'buy / sell（相对于 token0）',
    amount0_in      DECIMAL(65,18)  NOT NULL DEFAULT 0,
    amount1_in      DECIMAL(65,18)  NOT NULL DEFAULT 0,
    amount0_out     DECIMAL(65,18)  NOT NULL DEFAULT 0,
    amount1_out     DECIMAL(65,18)  NOT NULL DEFAULT 0,
    amount_usd      DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '折算美元',
    price           DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '成交价',
    block_number    BIGINT          NOT NULL,
    block_time      DATETIME        NOT NULL,
    tx_hash         VARCHAR(66)     NOT NULL,
    log_index       INT             NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_tx_log (chain_id, tx_hash, log_index),
    INDEX idx_pair_time (pair_address, block_time DESC),
    INDEX idx_trader_time (trader_address, block_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='DEX 成交记录';

CREATE TABLE IF NOT EXISTS dex_liquidity_events (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id       BIGINT          NOT NULL,
    pair_address   VARCHAR(42)     NOT NULL,
    user_address   VARCHAR(42)     NOT NULL,
    event_type     VARCHAR(8)      NOT NULL COMMENT 'mint / burn',
    amount0        DECIMAL(65,18)  NOT NULL DEFAULT 0,
    amount1        DECIMAL(65,18)  NOT NULL DEFAULT 0,
    lp_amount      DECIMAL(65,18)  NOT NULL DEFAULT 0,
    amount_usd     DECIMAL(65,18)  NOT NULL DEFAULT 0,
    block_number   BIGINT          NOT NULL,
    block_time     DATETIME        NOT NULL,
    tx_hash        VARCHAR(66)     NOT NULL,
    log_index      INT             NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_tx_log (chain_id, tx_hash, log_index),
    INDEX idx_user_time (user_address, block_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流动性事件（加/减）';

CREATE TABLE IF NOT EXISTS dex_liquidity_positions (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    user_address    VARCHAR(42)     NOT NULL,
    pair_address    VARCHAR(42)     NOT NULL,
    lp_balance      DECIMAL(65,18)  NOT NULL DEFAULT 0,
    share_ratio     DECIMAL(30,18)  NOT NULL DEFAULT 0 COMMENT '份额占比',
    deposited_usd   DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '累计投入',
    pending_reward  DECIMAL(65,18)  NOT NULL DEFAULT 0,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_user_pair (chain_id, user_address, pair_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户 LP 头寸';

CREATE TABLE IF NOT EXISTS dex_pair_stats_daily (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id      BIGINT          NOT NULL,
    stat_date     DATE            NOT NULL,
    pair_address  VARCHAR(42)     NOT NULL,
    volume_usd    DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '日成交额',
    fees_usd      DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '手续费（USD）',
    tx_count      BIGINT          NOT NULL DEFAULT 0 COMMENT '交易笔数',
    open_price    DECIMAL(65,18)  NOT NULL DEFAULT 0,
    high_price    DECIMAL(65,18)  NOT NULL DEFAULT 0,
    low_price     DECIMAL(65,18)  NOT NULL DEFAULT 0,
    close_price   DECIMAL(65,18)  NOT NULL DEFAULT 0,
    tvl_usd       DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '当日最后 TVL',
    PRIMARY KEY (id),
    UNIQUE INDEX uk_date_pair (chain_id, stat_date, pair_address),
    INDEX idx_pair_date (pair_address, stat_date DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='交易对日统计（K 线/聚合）';
