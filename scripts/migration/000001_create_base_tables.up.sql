-- 000001_create_base_tables.up.sql
-- 链上同步层：区块记录、同步游标、原始事件

-- 已同步区块记录
CREATE TABLE IF NOT EXISTS chain_blocks (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id     BIGINT          NOT NULL,
    block_number BIGINT          NOT NULL,
    block_hash   VARCHAR(66)     NOT NULL,
    parent_hash  VARCHAR(66)     NOT NULL,
    block_time   DATETIME        NOT NULL,
    tx_count     INT             NOT NULL DEFAULT 0,
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_block (chain_id, block_number),
    INDEX idx_block_time (block_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='已同步区块记录';

-- 链上同步游标
CREATE TABLE IF NOT EXISTS chain_sync_cursors (
    id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    module               VARCHAR(32)     NOT NULL COMMENT '模块名，如 dex / lending',
    contract_address     VARCHAR(42)     DEFAULT NULL COMMENT '合约地址，可空表示模块级游标',
    chain_id             BIGINT          NOT NULL,
    last_scanned_block   BIGINT          NOT NULL DEFAULT 0,
    last_confirmed_block BIGINT          NOT NULL DEFAULT 0,
    status               VARCHAR(16)     NOT NULL DEFAULT 'running' COMMENT 'running / paused / error',
    error_message        VARCHAR(512)    DEFAULT NULL COMMENT '最近一次错误信息',
    updated_at           DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_module_contract_chain (module, contract_address, chain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='链上同步游标';

-- 链上原始事件（数据真相来源）
CREATE TABLE IF NOT EXISTS chain_raw_events (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id         BIGINT          NOT NULL,
    module           VARCHAR(32)     NOT NULL COMMENT '所属模块',
    contract_address VARCHAR(42)     NOT NULL,
    event_name       VARCHAR(64)     NOT NULL,
    block_number     BIGINT          NOT NULL,
    block_time       DATETIME        NOT NULL,
    tx_hash          VARCHAR(66)     NOT NULL,
    tx_index         INT             NOT NULL DEFAULT 0,
    log_index        INT             NOT NULL,
    topic0           VARCHAR(66)     NOT NULL COMMENT '事件签名哈希',
    payload_json     JSON            NOT NULL COMMENT '解析后的事件参数',
    status           VARCHAR(16)     NOT NULL DEFAULT 'pending' COMMENT 'pending / confirmed / reverted',
    created_at       DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE INDEX uk_chain_tx_log (chain_id, tx_hash, log_index),
    INDEX idx_module_event (module, event_name, block_number),
    INDEX idx_contract_block (contract_address, block_number),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='链上原始事件（数据真相来源）';
