-- 000007_create_bonds_tables.up.sql

CREATE TABLE IF NOT EXISTS bond_markets (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id            BIGINT          NOT NULL,
    market_address      VARCHAR(42)     NOT NULL,
    payout_token        VARCHAR(42)     NOT NULL COMMENT '兑付资产（通常是协议代币）',
    payout_symbol       VARCHAR(32)     NOT NULL DEFAULT '',
    quote_token         VARCHAR(42)     NOT NULL COMMENT '支付资产（如 USDC、LP）',
    quote_symbol        VARCHAR(32)     NOT NULL DEFAULT '',
    discount_rate       DECIMAL(20,8)   NOT NULL DEFAULT 0 COMMENT '当前折价率',
    vesting_seconds     BIGINT          NOT NULL COMMENT '归属期（秒）',
    total_capacity      DECIMAL(65,18)  NOT NULL DEFAULT 0,
    remaining_capacity  DECIMAL(65,18)  NOT NULL DEFAULT 0,
    status              VARCHAR(16)     NOT NULL DEFAULT 'active' COMMENT 'active / ended',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_market (chain_id, market_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='债券市场';

CREATE TABLE IF NOT EXISTS bond_positions (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id         BIGINT          NOT NULL,
    bond_nft_id      VARCHAR(66)     NOT NULL COMMENT '债券 NFT ID',
    user_address     VARCHAR(42)     NOT NULL,
    market_address   VARCHAR(42)     NOT NULL,
    paid_amount      DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '支付金额',
    payout_amount    DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '总兑付金额',
    claimed_amount   DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '已领取金额',
    vesting_start    DATETIME        NOT NULL,
    vesting_end      DATETIME        NOT NULL,
    status           VARCHAR(16)     NOT NULL DEFAULT 'vesting' COMMENT 'vesting / claimable / claimed',
    block_number     BIGINT          NOT NULL,
    block_time       DATETIME        NOT NULL,
    tx_hash          VARCHAR(66)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_nft (chain_id, bond_nft_id),
    INDEX idx_user_status (user_address, status),
    INDEX idx_vesting_end (vesting_end)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户债券仓位';

CREATE TABLE IF NOT EXISTS bond_redemptions (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id       BIGINT          NOT NULL,
    bond_nft_id    VARCHAR(66)     NOT NULL,
    user_address   VARCHAR(42)     NOT NULL,
    claim_amount   DECIMAL(65,18)  NOT NULL DEFAULT 0,
    block_number   BIGINT          NOT NULL,
    block_time     DATETIME        NOT NULL,
    tx_hash        VARCHAR(66)     NOT NULL,
    log_index      INT             NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_tx_log (chain_id, tx_hash, log_index),
    INDEX idx_user_time (user_address, block_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='债券兑付记录';
