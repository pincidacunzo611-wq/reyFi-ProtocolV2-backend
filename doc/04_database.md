# 04 — 数据库设计

> 本文档给出 ReyFi 后端推荐的 MySQL 表设计。重点不是把所有字段一次写死，而是先建立稳定的数据分层和命名规范。

---

## 一、设计原则

数据库建议分 4 层：

```
┌─────────────────────────────────────────────────┐
│        第 4 层：系统配置层                         │
│   任务记录、告警、配置、用户偏好                     │
├─────────────────────────────────────────────────┤
│        第 3 层：聚合视图层                         │
│   24h 成交量、TVL、APR、用户总资产                  │
│   ⚠️ 这一层是性能优化层，不是数据真相来源             │
├─────────────────────────────────────────────────┤
│        第 2 层：领域明细层                         │
│   交易、头寸、投票、申赎等业务明细                    │
├─────────────────────────────────────────────────┤
│        第 1 层：链上同步层                         │
│   区块、交易、原始事件、同步游标                     │
│   ✅ 这一层是数据真相来源，可重放                    │
└─────────────────────────────────────────────────┘
```

1. **链上同步层**
   存区块、交易、原始事件、同步游标。
2. **领域明细层**
   按模块存交易、头寸、投票、申赎等业务明细。
3. **聚合视图层**
   存 24h 成交量、TVL、APR、用户总资产等结果。
4. **系统配置层**
   存任务、告警、配置、用户偏好。

---

## 二、命名规范

### 2.1 表和字段命名

- 表名统一小写下划线，例如 `dex_pairs`
- 表名前缀与模块对应：`dex_*`、`lending_*`、`futures_*`、`options_*`、`vault_*`、`bond_*`、`gov_*`
- 主键统一用 `id BIGINT UNSIGNED AUTO_INCREMENT`
- 地址字段统一用 `VARCHAR(42)` — 以太坊地址是 `0x` + 40 个十六进制字符
- 哈希字段统一用 `VARCHAR(66)` — `0x` + 64 个十六进制字符
- 大额数值统一用 `DECIMAL(65, 18)` — 覆盖 uint256 的精度需求
- 布尔值用 `TINYINT(1)` — `0` 表示 false、`1` 表示 true

### 2.2 通用字段

所有表统一有：

```sql
created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
```

如果是链上事件明细表，再加：

```sql
chain_id     BIGINT       NOT NULL,
block_number BIGINT       NOT NULL,
block_time   DATETIME     NOT NULL,
tx_hash      VARCHAR(66)  NOT NULL,
log_index    INT          NOT NULL
```

### 2.3 索引命名

| 类型 | 前缀 | 示例 |
|------|------|------|
| 主键 | `pk_` | `pk_id`（MySQL 自动处理） |
| 唯一索引 | `uk_` | `uk_chain_pair` |
| 普通索引 | `idx_` | `idx_pair_block_time` |
| 联合索引 | `idx_` | `idx_user_status` |

---

## 三、公共基础表

### 3.1 `chain_blocks`

记录已同步区块。

```sql
CREATE TABLE chain_blocks (
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
```

### 3.2 `chain_sync_cursors`

记录每个模块同步到哪里。

```sql
CREATE TABLE chain_sync_cursors (
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
```

### 3.3 `chain_raw_events`

统一存原始事件，支持重放。

```sql
CREATE TABLE chain_raw_events (
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
```

> **重要**: 这张表是整个系统的数据真相来源。所有聚合数据都可以通过重放这张表来重建。

---

## 四、DEX 模块表

### 4.1 `dex_pairs`

```sql
CREATE TABLE dex_pairs (
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
```

### 4.2 `dex_pair_snapshots`

记录某时刻池子状态，用于绘制历史曲线。

```sql
CREATE TABLE dex_pair_snapshots (
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
```

### 4.3 `dex_trades`

```sql
CREATE TABLE dex_trades (
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
```

### 4.4 `dex_liquidity_events`

记录加减流动性事件。

```sql
CREATE TABLE dex_liquidity_events (
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
```

### 4.5 `dex_liquidity_positions`

```sql
CREATE TABLE dex_liquidity_positions (
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
```

### 4.6 `dex_pair_stats_daily`

按日聚合，生成 K 线和统计数据。

```sql
CREATE TABLE dex_pair_stats_daily (
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
```

---

## 五、Lending 模块表

### 5.1 `lending_markets`

```sql
CREATE TABLE lending_markets (
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
```

### 5.2 `lending_market_snapshots`

```sql
CREATE TABLE lending_market_snapshots (
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
```

### 5.3 `lending_user_positions`

```sql
CREATE TABLE lending_user_positions (
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
```

### 5.4 `lending_liquidations`

```sql
CREATE TABLE lending_liquidations (
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
```

---

## 六、Futures / Leverage 模块表

### 6.1 `futures_markets`

```sql
CREATE TABLE futures_markets (
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
```

### 6.2 `futures_positions`

```sql
CREATE TABLE futures_positions (
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
```

### 6.3 `futures_funding_records`

```sql
CREATE TABLE futures_funding_records (
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
```

### 6.4 `futures_liquidations`

```sql
CREATE TABLE futures_liquidations (
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
```

---

## 七、Options 模块表

### 7.1 `options_markets`

```sql
CREATE TABLE options_markets (
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
```

### 7.2 `options_positions`

```sql
CREATE TABLE options_positions (
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
```

### 7.3 `options_settlements`

```sql
CREATE TABLE options_settlements (
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
```

### 7.4 `options_vol_surfaces`

如果后面需要独立做报价引擎，可存隐含波动率面或历史波动率快照。

```sql
CREATE TABLE options_vol_surfaces (
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
```

---

## 八、Vault 模块表

### 8.1 `vaults`

```sql
CREATE TABLE vaults (
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
```

### 8.2 `vault_snapshots`

```sql
CREATE TABLE vault_snapshots (
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
```

### 8.3 `vault_user_positions`

```sql
CREATE TABLE vault_user_positions (
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
```

### 8.4 `vault_harvest_records`

```sql
CREATE TABLE vault_harvest_records (
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
```

---

## 九、Bonds 模块表

### 9.1 `bond_markets`

```sql
CREATE TABLE bond_markets (
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
```

### 9.2 `bond_positions`

```sql
CREATE TABLE bond_positions (
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
```

### 9.3 `bond_redemptions`

```sql
CREATE TABLE bond_redemptions (
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
```

---

## 十、Governance 模块表

### 10.1 `gov_proposals`

```sql
CREATE TABLE gov_proposals (
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
```

### 10.2 `gov_votes`

```sql
CREATE TABLE gov_votes (
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
```

### 10.3 `gov_ve_locks`

```sql
CREATE TABLE gov_ve_locks (
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
```

### 10.4 `gov_gauge_votes`

```sql
CREATE TABLE gov_gauge_votes (
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
```

### 10.5 `gov_bribe_rewards`

```sql
CREATE TABLE gov_bribe_rewards (
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
```

---

## 十一、User / Account 模块表

### 11.1 `users`

```sql
CREATE TABLE users (
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
```

### 11.2 `user_nonces`

用于钱包签名登录的一次性随机数。

```sql
CREATE TABLE user_nonces (
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
```

### 11.3 `user_sessions`

```sql
CREATE TABLE user_sessions (
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
```

### 11.4 `user_asset_snapshots`

```sql
CREATE TABLE user_asset_snapshots (
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
```

### 11.5 `user_activity_stream`

```sql
CREATE TABLE user_activity_stream (
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
```

---

## 十二、Bot / System 模块表

### 12.1 `job_runs`

```sql
CREATE TABLE job_runs (
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
```

### 12.2 `alert_records`

```sql
CREATE TABLE alert_records (
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
```

### 12.3 `liquidation_candidates`

```sql
CREATE TABLE liquidation_candidates (
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
```

---

## 十三、索引建议

高频索引重点放在：

| 索引场景 | 建议索引 |
|---------|---------|
| 交易对成交查询 | `dex_trades(pair_address, block_time DESC)` |
| 用户交易历史 | `dex_trades(trader_address, block_time DESC)` |
| 用户借贷头寸 | `lending_user_positions(user_address, asset_address)` |
| 用户期货仓位 | `futures_positions(user_address, status)` |
| 投票查询 | `gov_votes(proposal_id, voter_address)` |
| 事件去重 | `chain_raw_events(chain_id, tx_hash, log_index)` — UNIQUE |
| 同步游标 | `chain_sync_cursors(module, contract_address, chain_id)` — UNIQUE |
| 活动流 | `user_activity_stream(wallet_address, block_time DESC)` |

> **注意**: 不要过度索引。每个索引都会增加写入开销。先上线，再根据慢查询日志加索引。

---

## 十四、分表策略

当数据量上涨后，优先考虑按时间分表的表：

| 表名 | 分表策略 | 预计何时需要 |
|------|---------|------------|
| `chain_raw_events` | 按月分表 | 数据量 > 1000 万 |
| `dex_trades` | 按月分表 | 数据量 > 500 万 |
| `futures_funding_records` | 按月分表 | 取决于市场数量 |
| `user_activity_stream` | 按月分表 | 数据量 > 1000 万 |

分表命名示例 ：

```
dex_trades_2026_04
dex_trades_2026_05
chain_raw_events_2026_04
```

> **Phase 1 不需要做分表**。先用单表 + 合理索引，等数据量真正上来再考虑。过早优化是万恶之源。

---

## 十五、最重要的结论

1. **原始事件表（`chain_raw_events`）一定要保留** — 它是重建一切数据的基础
2. **用户视角表和模块明细表要分开** — 不要让一张表承担两种查询模式
3. **聚合表是性能层，不是真相来源** — 聚合表可以从原始表重建
4. **所有链上明细表都要能靠 `chain_id + tx_hash + log_index` 回溯**
5. **先单表 + 索引，后分表** — 避免过早优化
