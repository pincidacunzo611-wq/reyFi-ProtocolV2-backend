package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ==================== LendingMarketModel ====================

type LendingMarket struct {
	Id                    int64     `db:"id"`
	ChainId               int64     `db:"chain_id"`
	AssetAddress          string    `db:"asset_address"`
	AssetSymbol           string    `db:"asset_symbol"`
	AssetDecimals         int       `db:"asset_decimals"`
	RtokenAddress         string    `db:"rtoken_address"`
	DebtTokenAddress      string    `db:"debt_token_address"`
	CollateralFactor      string    `db:"collateral_factor"`
	LiquidationThreshold  string    `db:"liquidation_threshold"`
	LiquidationPenalty    string    `db:"liquidation_penalty"`
	ReserveFactor         string    `db:"reserve_factor"`
	IsActive              bool      `db:"is_active"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}

type LendingMarketModel struct {
	db *sql.DB
}

func NewLendingMarketModel(db *sql.DB) *LendingMarketModel {
	return &LendingMarketModel{db: db}
}

func (m *LendingMarketModel) ListMarkets(ctx context.Context, chainId int64, keyword string, page, pageSize int64) ([]*LendingMarket, int64, error) {
	where := "chain_id = ? AND is_active = 1"
	args := []interface{}{chainId}
	if keyword != "" {
		where += " AND asset_symbol LIKE ?"
		args = append(args, "%"+keyword+"%")
	}

	var total int64
	m.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM lending_markets WHERE %s", where), args...).Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, chain_id, asset_address, asset_symbol, asset_decimals,
		 rtoken_address, debt_token_address, collateral_factor, liquidation_threshold,
		 liquidation_penalty, reserve_factor, is_active, created_at, updated_at
		 FROM lending_markets WHERE %s ORDER BY id ASC LIMIT ? OFFSET ?`, where),
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var markets []*LendingMarket
	for rows.Next() {
		m := &LendingMarket{}
		if err := rows.Scan(&m.Id, &m.ChainId, &m.AssetAddress, &m.AssetSymbol, &m.AssetDecimals,
			&m.RtokenAddress, &m.DebtTokenAddress, &m.CollateralFactor, &m.LiquidationThreshold,
			&m.LiquidationPenalty, &m.ReserveFactor, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, err
		}
		markets = append(markets, m)
	}
	return markets, total, nil
}

func (m *LendingMarketModel) FindByAsset(ctx context.Context, chainId int64, assetAddress string) (*LendingMarket, error) {
	market := &LendingMarket{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, asset_address, asset_symbol, asset_decimals,
		 rtoken_address, debt_token_address, collateral_factor, liquidation_threshold,
		 liquidation_penalty, reserve_factor, is_active, created_at, updated_at
		 FROM lending_markets WHERE chain_id = ? AND asset_address = ?`,
		chainId, assetAddress,
	).Scan(&market.Id, &market.ChainId, &market.AssetAddress, &market.AssetSymbol, &market.AssetDecimals,
		&market.RtokenAddress, &market.DebtTokenAddress, &market.CollateralFactor, &market.LiquidationThreshold,
		&market.LiquidationPenalty, &market.ReserveFactor, &market.IsActive, &market.CreatedAt, &market.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return market, nil
}

func (m *LendingMarketModel) Upsert(ctx context.Context, market *LendingMarket) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO lending_markets (chain_id, asset_address, asset_symbol, asset_decimals,
		 rtoken_address, debt_token_address, collateral_factor, liquidation_threshold,
		 liquidation_penalty, reserve_factor, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		 asset_symbol=VALUES(asset_symbol), collateral_factor=VALUES(collateral_factor),
		 liquidation_threshold=VALUES(liquidation_threshold), reserve_factor=VALUES(reserve_factor),
		 updated_at=NOW()`,
		market.ChainId, market.AssetAddress, market.AssetSymbol, market.AssetDecimals,
		market.RtokenAddress, market.DebtTokenAddress, market.CollateralFactor,
		market.LiquidationThreshold, market.LiquidationPenalty, market.ReserveFactor, market.IsActive)
	return err
}

// ==================== LendingMarketSnapshotModel ====================

type LendingMarketSnapshot struct {
	Id              int64     `db:"id"`
	ChainId         int64     `db:"chain_id"`
	AssetAddress    string    `db:"asset_address"`
	TotalSupply     string    `db:"total_supply"`
	TotalBorrow     string    `db:"total_borrow"`
	UtilizationRate string    `db:"utilization_rate"`
	SupplyApr       string    `db:"supply_apr"`
	BorrowApr       string    `db:"borrow_apr"`
	TvlUsd          string    `db:"tvl_usd"`
	SnapshotTime    time.Time `db:"snapshot_time"`
}

type LendingMarketSnapshotModel struct {
	db *sql.DB
}

func NewLendingMarketSnapshotModel(db *sql.DB) *LendingMarketSnapshotModel {
	return &LendingMarketSnapshotModel{db: db}
}

func (m *LendingMarketSnapshotModel) Insert(ctx context.Context, snap *LendingMarketSnapshot) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO lending_market_snapshots (chain_id, asset_address, total_supply, total_borrow,
		 utilization_rate, supply_apr, borrow_apr, tvl_usd, snapshot_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.ChainId, snap.AssetAddress, snap.TotalSupply, snap.TotalBorrow,
		snap.UtilizationRate, snap.SupplyApr, snap.BorrowApr, snap.TvlUsd, snap.SnapshotTime)
	return err
}

func (m *LendingMarketSnapshotModel) GetLatest(ctx context.Context, chainId int64, assetAddress string) (*LendingMarketSnapshot, error) {
	snap := &LendingMarketSnapshot{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, asset_address, total_supply, total_borrow,
		 utilization_rate, supply_apr, borrow_apr, tvl_usd, snapshot_time
		 FROM lending_market_snapshots
		 WHERE chain_id = ? AND asset_address = ? ORDER BY snapshot_time DESC LIMIT 1`,
		chainId, assetAddress,
	).Scan(&snap.Id, &snap.ChainId, &snap.AssetAddress, &snap.TotalSupply, &snap.TotalBorrow,
		&snap.UtilizationRate, &snap.SupplyApr, &snap.BorrowApr, &snap.TvlUsd, &snap.SnapshotTime)
	return snap, err
}

// ==================== LendingUserPositionModel ====================

type LendingUserPosition struct {
	Id                int64     `db:"id"`
	ChainId           int64     `db:"chain_id"`
	UserAddress       string    `db:"user_address"`
	AssetAddress      string    `db:"asset_address"`
	SuppliedAmount    string    `db:"supplied_amount"`
	BorrowedAmount    string    `db:"borrowed_amount"`
	CollateralEnabled bool      `db:"collateral_enabled"`
	HealthFactor      string    `db:"health_factor"`
	UpdatedAt         time.Time `db:"updated_at"`
}

type LendingUserPositionModel struct {
	db *sql.DB
}

func NewLendingUserPositionModel(db *sql.DB) *LendingUserPositionModel {
	return &LendingUserPositionModel{db: db}
}

func (m *LendingUserPositionModel) ListByUser(ctx context.Context, chainId int64, userAddress string) ([]*LendingUserPosition, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, user_address, asset_address, supplied_amount, borrowed_amount,
		 collateral_enabled, health_factor, updated_at
		 FROM lending_user_positions WHERE chain_id = ? AND user_address = ?`,
		chainId, userAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*LendingUserPosition
	for rows.Next() {
		p := &LendingUserPosition{}
		if err := rows.Scan(&p.Id, &p.ChainId, &p.UserAddress, &p.AssetAddress,
			&p.SuppliedAmount, &p.BorrowedAmount, &p.CollateralEnabled,
			&p.HealthFactor, &p.UpdatedAt); err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, nil
}

func (m *LendingUserPositionModel) Upsert(ctx context.Context, pos *LendingUserPosition) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO lending_user_positions (chain_id, user_address, asset_address,
		 supplied_amount, borrowed_amount, collateral_enabled, health_factor)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		 supplied_amount=VALUES(supplied_amount), borrowed_amount=VALUES(borrowed_amount),
		 health_factor=VALUES(health_factor), updated_at=NOW()`,
		pos.ChainId, pos.UserAddress, pos.AssetAddress,
		pos.SuppliedAmount, pos.BorrowedAmount, pos.CollateralEnabled, pos.HealthFactor)
	return err
}

// ==================== LendingLiquidationModel ====================

type LendingLiquidation struct {
	Id                  int64     `db:"id"`
	ChainId             int64     `db:"chain_id"`
	BorrowerAddress     string    `db:"borrower_address"`
	LiquidatorAddress   string    `db:"liquidator_address"`
	CollateralAsset     string    `db:"collateral_asset"`
	DebtAsset           string    `db:"debt_asset"`
	CollateralAmount    string    `db:"collateral_amount"`
	DebtAmount          string    `db:"debt_amount"`
	PenaltyAmount       string    `db:"penalty_amount"`
	HealthFactorBefore  string    `db:"health_factor_before"`
	BlockNumber         int64     `db:"block_number"`
	BlockTime           time.Time `db:"block_time"`
	TxHash              string    `db:"tx_hash"`
	LogIndex            int       `db:"log_index"`
}

type LendingLiquidationModel struct {
	db *sql.DB
}

func NewLendingLiquidationModel(db *sql.DB) *LendingLiquidationModel {
	return &LendingLiquidationModel{db: db}
}

func (m *LendingLiquidationModel) Insert(ctx context.Context, liq *LendingLiquidation) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO lending_liquidations (chain_id, borrower_address, liquidator_address,
		 collateral_asset, debt_asset, collateral_amount, debt_amount, penalty_amount,
		 health_factor_before, block_number, block_time, tx_hash, log_index)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id=id`,
		liq.ChainId, liq.BorrowerAddress, liq.LiquidatorAddress,
		liq.CollateralAsset, liq.DebtAsset, liq.CollateralAmount, liq.DebtAmount,
		liq.PenaltyAmount, liq.HealthFactorBefore, liq.BlockNumber, liq.BlockTime,
		liq.TxHash, liq.LogIndex)
	return err
}

func (m *LendingLiquidationModel) ListByBorrower(ctx context.Context, chainId int64, borrower string, page, pageSize int64) ([]*LendingLiquidation, int64, error) {
	where := "chain_id = ?"
	args := []interface{}{chainId}
	if borrower != "" {
		where += " AND borrower_address = ?"
		args = append(args, borrower)
	}

	var total int64
	m.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM lending_liquidations WHERE %s", where), args...).Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, chain_id, borrower_address, liquidator_address,
		 collateral_asset, debt_asset, collateral_amount, debt_amount, penalty_amount,
		 health_factor_before, block_number, block_time, tx_hash, log_index
		 FROM lending_liquidations WHERE %s ORDER BY block_time DESC LIMIT ? OFFSET ?`, where),
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*LendingLiquidation
	for rows.Next() {
		l := &LendingLiquidation{}
		if err := rows.Scan(&l.Id, &l.ChainId, &l.BorrowerAddress, &l.LiquidatorAddress,
			&l.CollateralAsset, &l.DebtAsset, &l.CollateralAmount, &l.DebtAmount,
			&l.PenaltyAmount, &l.HealthFactorBefore, &l.BlockNumber, &l.BlockTime,
			&l.TxHash, &l.LogIndex); err != nil {
			return nil, 0, err
		}
		list = append(list, l)
	}
	return list, total, nil
}
