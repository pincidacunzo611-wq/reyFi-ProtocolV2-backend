// Package model DEX 模块数据库 Model 层
package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ==================== DexPairModel ====================

type DexPair struct {
	Id             int64     `db:"id"`
	ChainId        int64     `db:"chain_id"`
	PairAddress    string    `db:"pair_address"`
	Token0Address  string    `db:"token0_address"`
	Token1Address  string    `db:"token1_address"`
	Token0Symbol   string    `db:"token0_symbol"`
	Token1Symbol   string    `db:"token1_symbol"`
	Token0Decimals int       `db:"token0_decimals"`
	Token1Decimals int       `db:"token1_decimals"`
	FeeBps         int       `db:"fee_bps"`
	CreatedBlock   int64     `db:"created_block"`
	IsActive       bool      `db:"is_active"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type DexPairModel struct {
	db *sql.DB
}

func NewDexPairModel(db *sql.DB) *DexPairModel {
	return &DexPairModel{db: db}
}

func (m *DexPairModel) FindByAddress(ctx context.Context, chainId int64, pairAddress string) (*DexPair, error) {
	pair := &DexPair{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, pair_address, token0_address, token1_address, 
		 token0_symbol, token1_symbol, token0_decimals, token1_decimals, 
		 fee_bps, created_block, is_active, created_at, updated_at
		 FROM dex_pairs WHERE chain_id = ? AND pair_address = ?`,
		chainId, pairAddress,
	).Scan(&pair.Id, &pair.ChainId, &pair.PairAddress, &pair.Token0Address, &pair.Token1Address,
		&pair.Token0Symbol, &pair.Token1Symbol, &pair.Token0Decimals, &pair.Token1Decimals,
		&pair.FeeBps, &pair.CreatedBlock, &pair.IsActive, &pair.CreatedAt, &pair.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return pair, nil
}

func (m *DexPairModel) ListPairs(ctx context.Context, chainId int64, keyword string, page, pageSize int64) ([]*DexPair, int64, error) {
	where := "chain_id = ? AND is_active = 1"
	args := []interface{}{chainId}

	if keyword != "" {
		where += " AND (token0_symbol LIKE ? OR token1_symbol LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}

	// Count
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM dex_pairs WHERE %s", where)
	if err := m.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List
	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf(
		`SELECT id, chain_id, pair_address, token0_address, token1_address, 
		 token0_symbol, token1_symbol, token0_decimals, token1_decimals, 
		 fee_bps, created_block, is_active, created_at, updated_at
		 FROM dex_pairs WHERE %s ORDER BY id DESC LIMIT ? OFFSET ?`, where)
	listArgs := append(args, pageSize, offset)

	rows, err := m.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pairs []*DexPair
	for rows.Next() {
		p := &DexPair{}
		if err := rows.Scan(&p.Id, &p.ChainId, &p.PairAddress, &p.Token0Address, &p.Token1Address,
			&p.Token0Symbol, &p.Token1Symbol, &p.Token0Decimals, &p.Token1Decimals,
			&p.FeeBps, &p.CreatedBlock, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		pairs = append(pairs, p)
	}

	return pairs, total, nil
}

func (m *DexPairModel) Upsert(ctx context.Context, pair *DexPair) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO dex_pairs (chain_id, pair_address, token0_address, token1_address,
		 token0_symbol, token1_symbol, token0_decimals, token1_decimals, fee_bps, created_block, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		 token0_symbol=VALUES(token0_symbol), token1_symbol=VALUES(token1_symbol),
		 updated_at=NOW()`,
		pair.ChainId, pair.PairAddress, pair.Token0Address, pair.Token1Address,
		pair.Token0Symbol, pair.Token1Symbol, pair.Token0Decimals, pair.Token1Decimals,
		pair.FeeBps, pair.CreatedBlock, pair.IsActive)
	return err
}

// ==================== DexTradeModel ====================

type DexTrade struct {
	Id             int64     `db:"id"`
	ChainId        int64     `db:"chain_id"`
	PairAddress    string    `db:"pair_address"`
	TraderAddress  string    `db:"trader_address"`
	SenderAddress  string    `db:"sender_address"`
	Direction      string    `db:"direction"`
	Amount0In      string    `db:"amount0_in"`
	Amount1In      string    `db:"amount1_in"`
	Amount0Out     string    `db:"amount0_out"`
	Amount1Out     string    `db:"amount1_out"`
	AmountUsd      string    `db:"amount_usd"`
	Price          string    `db:"price"`
	BlockNumber    int64     `db:"block_number"`
	BlockTime      time.Time `db:"block_time"`
	TxHash         string    `db:"tx_hash"`
	LogIndex       int       `db:"log_index"`
}

type DexTradeModel struct {
	db *sql.DB
}

func NewDexTradeModel(db *sql.DB) *DexTradeModel {
	return &DexTradeModel{db: db}
}

func (m *DexTradeModel) Insert(ctx context.Context, trade *DexTrade) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO dex_trades (chain_id, pair_address, trader_address, sender_address, direction,
		 amount0_in, amount1_in, amount0_out, amount1_out, amount_usd, price,
		 block_number, block_time, tx_hash, log_index)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id=id`,
		trade.ChainId, trade.PairAddress, trade.TraderAddress, trade.SenderAddress, trade.Direction,
		trade.Amount0In, trade.Amount1In, trade.Amount0Out, trade.Amount1Out, trade.AmountUsd, trade.Price,
		trade.BlockNumber, trade.BlockTime, trade.TxHash, trade.LogIndex)
	return err
}

func (m *DexTradeModel) ListByPair(ctx context.Context, chainId int64, pairAddress string, page, pageSize int64) ([]*DexTrade, int64, error) {
	where := "chain_id = ?"
	args := []interface{}{chainId}
	if pairAddress != "" {
		where += " AND pair_address = ?"
		args = append(args, pairAddress)
	}

	var total int64
	if err := m.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM dex_trades WHERE %s", where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, chain_id, pair_address, trader_address, sender_address, direction,
		 amount0_in, amount1_in, amount0_out, amount1_out, amount_usd, price,
		 block_number, block_time, tx_hash, log_index
		 FROM dex_trades WHERE %s ORDER BY block_time DESC LIMIT ? OFFSET ?`, where),
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var trades []*DexTrade
	for rows.Next() {
		t := &DexTrade{}
		if err := rows.Scan(&t.Id, &t.ChainId, &t.PairAddress, &t.TraderAddress, &t.SenderAddress, &t.Direction,
			&t.Amount0In, &t.Amount1In, &t.Amount0Out, &t.Amount1Out, &t.AmountUsd, &t.Price,
			&t.BlockNumber, &t.BlockTime, &t.TxHash, &t.LogIndex); err != nil {
			return nil, 0, err
		}
		trades = append(trades, t)
	}
	return trades, total, nil
}

func (m *DexTradeModel) ExistsByTxHashAndLogIndex(ctx context.Context, chainId int64, txHash string, logIndex int) (bool, error) {
	var count int64
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dex_trades WHERE chain_id = ? AND tx_hash = ? AND log_index = ?`,
		chainId, txHash, logIndex).Scan(&count)
	return count > 0, err
}

// ==================== DexPairSnapshotModel ====================

type DexPairSnapshot struct {
	Id           int64     `db:"id"`
	ChainId      int64     `db:"chain_id"`
	PairAddress  string    `db:"pair_address"`
	Reserve0     string    `db:"reserve0"`
	Reserve1     string    `db:"reserve1"`
	Price0       string    `db:"price0"`
	Price1       string    `db:"price1"`
	TotalSupply  string    `db:"total_supply"`
	TvlUsd       string    `db:"tvl_usd"`
	SnapshotTime time.Time `db:"snapshot_time"`
}

type DexPairSnapshotModel struct {
	db *sql.DB
}

func NewDexPairSnapshotModel(db *sql.DB) *DexPairSnapshotModel {
	return &DexPairSnapshotModel{db: db}
}

func (m *DexPairSnapshotModel) Insert(ctx context.Context, snap *DexPairSnapshot) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO dex_pair_snapshots (chain_id, pair_address, reserve0, reserve1, price0, price1, total_supply, tvl_usd, snapshot_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.ChainId, snap.PairAddress, snap.Reserve0, snap.Reserve1,
		snap.Price0, snap.Price1, snap.TotalSupply, snap.TvlUsd, snap.SnapshotTime)
	return err
}

func (m *DexPairSnapshotModel) GetLatest(ctx context.Context, chainId int64, pairAddress string) (*DexPairSnapshot, error) {
	snap := &DexPairSnapshot{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, pair_address, reserve0, reserve1, price0, price1, total_supply, tvl_usd, snapshot_time
		 FROM dex_pair_snapshots WHERE chain_id = ? AND pair_address = ? ORDER BY snapshot_time DESC LIMIT 1`,
		chainId, pairAddress,
	).Scan(&snap.Id, &snap.ChainId, &snap.PairAddress, &snap.Reserve0, &snap.Reserve1,
		&snap.Price0, &snap.Price1, &snap.TotalSupply, &snap.TvlUsd, &snap.SnapshotTime)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// ==================== Model 层扩展（供路由器使用） ====================

// ListAllActivePairs 获取指定链上所有活跃交易对（供路由器构建代币图）
func (m *DexPairModel) ListAllActivePairs(ctx context.Context, chainId int64) ([]*DexPair, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, pair_address, token0_address, token1_address,
		 token0_symbol, token1_symbol, token0_decimals, token1_decimals,
		 fee_bps, created_block, is_active, created_at, updated_at
		 FROM dex_pairs WHERE chain_id = ? AND is_active = 1`, chainId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []*DexPair
	for rows.Next() {
		p := &DexPair{}
		if err := rows.Scan(&p.Id, &p.ChainId, &p.PairAddress, &p.Token0Address, &p.Token1Address,
			&p.Token0Symbol, &p.Token1Symbol, &p.Token0Decimals, &p.Token1Decimals,
			&p.FeeBps, &p.CreatedBlock, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pairs = append(pairs, p)
	}
	return pairs, nil
}

// GetLatestSnapshots 批量获取多个交易对的最新快照（用于拿到最新 reserve）
// 返回 map[pairAddress]*DexPairSnapshot
func (m *DexPairSnapshotModel) GetLatestSnapshots(ctx context.Context, chainId int64, pairAddresses []string) (map[string]*DexPairSnapshot, error) {
	if len(pairAddresses) == 0 {
		return make(map[string]*DexPairSnapshot), nil
	}

	// 构建 IN 子句
	placeholders := make([]string, len(pairAddresses))
	args := make([]interface{}, 0, len(pairAddresses)+1)
	args = append(args, chainId)
	for i, addr := range pairAddresses {
		placeholders[i] = "?"
		args = append(args, addr)
	}
	inClause := strings.Join(placeholders, ",")

	// 使用子查询获取每个 pair 的最新快照
	query := fmt.Sprintf(`
		SELECT s.id, s.chain_id, s.pair_address, s.reserve0, s.reserve1,
		       s.price0, s.price1, s.total_supply, s.tvl_usd, s.snapshot_time
		FROM dex_pair_snapshots s
		INNER JOIN (
			SELECT pair_address, MAX(snapshot_time) AS max_time
			FROM dex_pair_snapshots
			WHERE chain_id = ? AND pair_address IN (%s)
			GROUP BY pair_address
		) latest ON s.pair_address = latest.pair_address AND s.snapshot_time = latest.max_time
		WHERE s.chain_id = ?`, inClause)
	args = append(args, chainId)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*DexPairSnapshot)
	for rows.Next() {
		snap := &DexPairSnapshot{}
		if err := rows.Scan(&snap.Id, &snap.ChainId, &snap.PairAddress, &snap.Reserve0, &snap.Reserve1,
			&snap.Price0, &snap.Price1, &snap.TotalSupply, &snap.TvlUsd, &snap.SnapshotTime); err != nil {
			return nil, err
		}
		result[snap.PairAddress] = snap
	}
	return result, nil
}

// ==================== DexLiquidityEvent Model ====================

type DexLiquidityEvent struct {
	Id          int64     `json:"id"`
	ChainId     int64     `json:"chainId"`
	PairAddress string    `json:"pairAddress"`
	UserAddress string    `json:"userAddress"`
	EventType   string    `json:"eventType"` // mint / burn
	Amount0     string    `json:"amount0"`
	Amount1     string    `json:"amount1"`
	LpAmount    string    `json:"lpAmount"`
	TxHash      string    `json:"txHash"`
	BlockTime   time.Time `json:"blockTime"`
	LogIndex    int       `json:"logIndex"`
}

type DexLiquidityEventModel struct{ db *sql.DB }

func NewDexLiquidityEventModel(db *sql.DB) *DexLiquidityEventModel {
	return &DexLiquidityEventModel{db: db}
}

func (m *DexLiquidityEventModel) Insert(ctx context.Context, event *DexLiquidityEvent) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO dex_liquidity_events (chain_id, pair_address, user_address, event_type, amount0, amount1, lp_amount, tx_hash, block_time, log_index)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id = id`,
		event.ChainId, event.PairAddress, event.UserAddress, event.EventType,
		event.Amount0, event.Amount1, event.LpAmount, event.TxHash, event.BlockTime, event.LogIndex)
	return err
}

func (m *DexLiquidityEventModel) ListByUser(ctx context.Context, chainId int64, userAddress string, page, pageSize int64) ([]*DexLiquidityEvent, int64, error) {
	offset := (page - 1) * pageSize
	var total int64
	m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dex_liquidity_events WHERE chain_id = ? AND user_address = ?`,
		chainId, userAddress).Scan(&total)

	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, pair_address, user_address, event_type, amount0, amount1, lp_amount, tx_hash, block_time, log_index
		 FROM dex_liquidity_events WHERE chain_id = ? AND user_address = ?
		 ORDER BY block_time DESC LIMIT ? OFFSET ?`,
		chainId, userAddress, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*DexLiquidityEvent
	for rows.Next() {
		e := &DexLiquidityEvent{}
		rows.Scan(&e.Id, &e.ChainId, &e.PairAddress, &e.UserAddress, &e.EventType,
			&e.Amount0, &e.Amount1, &e.LpAmount, &e.TxHash, &e.BlockTime, &e.LogIndex)
		list = append(list, e)
	}
	return list, total, nil
}

// ==================== DexLiquidityPosition Model ====================

type DexLiquidityPosition struct {
	Id          int64  `json:"id"`
	ChainId     int64  `json:"chainId"`
	PairAddress string `json:"pairAddress"`
	UserAddress string `json:"userAddress"`
	LpBalance   string `json:"lpBalance"`
	ShareRatio  string `json:"shareRatio"`
}

type DexLiquidityPositionModel struct{ db *sql.DB }

func NewDexLiquidityPositionModel(db *sql.DB) *DexLiquidityPositionModel {
	return &DexLiquidityPositionModel{db: db}
}

func (m *DexLiquidityPositionModel) GetByUser(ctx context.Context, chainId int64, userAddress string) ([]*DexLiquidityPosition, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, pair_address, user_address, lp_balance, share_ratio
		 FROM dex_liquidity_positions WHERE chain_id = ? AND user_address = ? AND CAST(lp_balance AS DECIMAL(65,18)) > 0`,
		chainId, userAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DexLiquidityPosition
	for rows.Next() {
		p := &DexLiquidityPosition{}
		rows.Scan(&p.Id, &p.ChainId, &p.PairAddress, &p.UserAddress, &p.LpBalance, &p.ShareRatio)
		list = append(list, p)
	}
	return list, nil
}

func (m *DexLiquidityPositionModel) Upsert(ctx context.Context, pos *DexLiquidityPosition) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO dex_liquidity_positions (chain_id, pair_address, user_address, lp_balance, share_ratio)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE lp_balance = VALUES(lp_balance), share_ratio = VALUES(share_ratio)`,
		pos.ChainId, pos.PairAddress, pos.UserAddress, pos.LpBalance, pos.ShareRatio)
	return err
}

// ==================== DexPairStatsDaily Model ====================

type DexPairStatsDaily struct {
	Id          int64     `json:"id"`
	ChainId     int64     `json:"chainId"`
	PairAddress string    `json:"pairAddress"`
	StatsDate   time.Time `json:"statsDate"`
	Volume      string    `json:"volume"`
	Fees        string    `json:"fees"`
	TxCount     int64     `json:"txCount"`
	HighPrice   string    `json:"highPrice"`
	LowPrice    string    `json:"lowPrice"`
}

type DexPairStatsDailyModel struct{ db *sql.DB }

func NewDexPairStatsDailyModel(db *sql.DB) *DexPairStatsDailyModel {
	return &DexPairStatsDailyModel{db: db}
}

func (m *DexPairStatsDailyModel) Upsert(ctx context.Context, stats *DexPairStatsDaily) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO dex_pair_stats_daily (chain_id, pair_address, stats_date, volume, fees, tx_count, high_price, low_price)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE volume = VALUES(volume), fees = VALUES(fees), 
		 tx_count = VALUES(tx_count), high_price = VALUES(high_price), low_price = VALUES(low_price)`,
		stats.ChainId, stats.PairAddress, stats.StatsDate, stats.Volume, stats.Fees,
		stats.TxCount, stats.HighPrice, stats.LowPrice)
	return err
}

func (m *DexPairStatsDailyModel) GetByPair(ctx context.Context, chainId int64, pairAddress string, days int) ([]*DexPairStatsDaily, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, pair_address, stats_date, volume, fees, tx_count, high_price, low_price
		 FROM dex_pair_stats_daily WHERE chain_id = ? AND pair_address = ?
		 ORDER BY stats_date DESC LIMIT ?`,
		chainId, pairAddress, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DexPairStatsDaily
	for rows.Next() {
		s := &DexPairStatsDaily{}
		rows.Scan(&s.Id, &s.ChainId, &s.PairAddress, &s.StatsDate, &s.Volume,
			&s.Fees, &s.TxCount, &s.HighPrice, &s.LowPrice)
		list = append(list, s)
	}
	return list, nil
}

// 消除未使用导入
var _ = strings.TrimSpace

