-- 000008_create_governance_tables.up.sql

CREATE TABLE IF NOT EXISTS gov_proposals (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    proposal_id     VARCHAR(66)     NOT NULL,
    proposer        VARCHAR(42)     NOT NULL,
    title           VARCHAR(255)    NOT NULL DEFAULT '',
    description     TEXT            DEFAULT NULL,
    start_block     BIGINT          NOT NULL DEFAULT 0,
    end_block       BIGINT          NOT NULL DEFAULT 0,
    start_time      DATETIME        DEFAULT NULL,
    end_time        DATETIME        DEFAULT NULL,
    for_votes       DECIMAL(65,18)  NOT NULL DEFAULT 0,
    against_votes   DECIMAL(65,18)  NOT NULL DEFAULT 0,
    abstain_votes   DECIMAL(65,18)  NOT NULL DEFAULT 0,
    quorum          DECIMAL(65,18)  NOT NULL DEFAULT 0,
    status          VARCHAR(16)     NOT NULL DEFAULT 'pending'
                    COMMENT 'pending / active / queued / executed / canceled / defeated',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_proposal (chain_id, proposal_id),
    INDEX idx_status_time (status, end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='治理提案';

CREATE TABLE IF NOT EXISTS gov_votes (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id      BIGINT          NOT NULL,
    proposal_id   VARCHAR(66)     NOT NULL,
    voter_address VARCHAR(42)     NOT NULL,
    support       TINYINT         NOT NULL COMMENT '0=反对, 1=赞成, 2=弃权',
    weight        DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '投票权重',
    reason        TEXT            DEFAULT NULL,
    block_number  BIGINT          NOT NULL,
    block_time    DATETIME        NOT NULL,
    tx_hash       VARCHAR(66)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_proposal_voter (chain_id, proposal_id, voter_address),
    INDEX idx_voter_time (voter_address, block_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='投票记录';

CREATE TABLE IF NOT EXISTS gov_ve_locks (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    user_address    VARCHAR(42)     NOT NULL,
    lock_id         VARCHAR(66)     DEFAULT NULL COMMENT '锁仓 NFT ID（如有）',
    locked_amount   DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '锁仓的 REY 数量',
    ve_balance      DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '当前 veREY 余额',
    lock_start      DATETIME        NOT NULL,
    lock_end        DATETIME        NOT NULL,
    status          VARCHAR(16)     NOT NULL DEFAULT 'active' COMMENT 'active / expired / withdrawn',
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_user_status (user_address, status),
    INDEX idx_lock_end (lock_end)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='veREY 锁仓记录';

CREATE TABLE IF NOT EXISTS gov_gauge_votes (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    user_address    VARCHAR(42)     NOT NULL,
    pool_address    VARCHAR(42)     NOT NULL COMMENT '投票给的池子',
    weight          DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '投票权重',
    epoch           INT             NOT NULL COMMENT '投票周期',
    block_number    BIGINT          NOT NULL,
    block_time      DATETIME        NOT NULL,
    tx_hash         VARCHAR(66)     NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_pool_epoch (pool_address, epoch),
    INDEX idx_user_epoch (user_address, epoch)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Gauge 权重投票';

CREATE TABLE IF NOT EXISTS gov_bribe_rewards (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id        BIGINT          NOT NULL,
    user_address    VARCHAR(42)     NOT NULL,
    pool_address    VARCHAR(42)     NOT NULL,
    reward_token    VARCHAR(42)     NOT NULL,
    reward_amount   DECIMAL(65,18)  NOT NULL DEFAULT 0,
    epoch           INT             NOT NULL,
    claimed         TINYINT(1)      NOT NULL DEFAULT 0,
    block_number    BIGINT          DEFAULT NULL,
    block_time      DATETIME        DEFAULT NULL,
    tx_hash         VARCHAR(66)     DEFAULT NULL,
    PRIMARY KEY (id),
    INDEX idx_user_claimed (user_address, claimed)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Bribe 奖励记录';
