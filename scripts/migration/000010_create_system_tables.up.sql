-- 000010_create_system_tables.up.sql

CREATE TABLE IF NOT EXISTS job_runs (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    job_name     VARCHAR(64)     NOT NULL COMMENT '任务名称',
    job_type     VARCHAR(32)     NOT NULL COMMENT 'cron / event / manual',
    status       VARCHAR(16)     NOT NULL DEFAULT 'running' COMMENT 'running / success / failed',
    started_at   DATETIME        NOT NULL,
    finished_at  DATETIME        DEFAULT NULL,
    duration_ms  BIGINT          DEFAULT NULL,
    result_json  JSON            DEFAULT NULL COMMENT '执行结果摘要',
    error        TEXT            DEFAULT NULL,
    PRIMARY KEY (id),
    INDEX idx_name_time (job_name, started_at DESC),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务执行记录';

CREATE TABLE IF NOT EXISTS alert_records (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    alert_type   VARCHAR(32)     NOT NULL COMMENT 'indexer_lag / service_down / high_gas 等',
    severity     VARCHAR(16)     NOT NULL COMMENT 'info / warning / critical',
    title        VARCHAR(255)    NOT NULL,
    message      TEXT            DEFAULT NULL,
    is_resolved  TINYINT(1)      NOT NULL DEFAULT 0,
    resolved_at  DATETIME        DEFAULT NULL,
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_type_resolved (alert_type, is_resolved),
    INDEX idx_severity_time (severity, created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统告警记录';

CREATE TABLE IF NOT EXISTS liquidation_candidates (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    chain_id         BIGINT          NOT NULL,
    module           VARCHAR(16)     NOT NULL COMMENT 'lending / futures',
    user_address     VARCHAR(42)     NOT NULL,
    position_id      VARCHAR(66)     DEFAULT NULL,
    health_factor    DECIMAL(20,8)   NOT NULL DEFAULT 0,
    shortfall_usd    DECIMAL(65,18)  NOT NULL DEFAULT 0 COMMENT '不足金额',
    status           VARCHAR(16)     NOT NULL DEFAULT 'pending' COMMENT 'pending / executing / done / skipped',
    detected_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    executed_at      DATETIME        DEFAULT NULL,
    execute_tx_hash  VARCHAR(66)     DEFAULT NULL,
    updated_at       DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_module_status (module, status),
    INDEX idx_health (health_factor)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='可清算候选列表';
