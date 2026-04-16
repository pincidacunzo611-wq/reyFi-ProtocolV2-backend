package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ==================== FuturesMarketModel ====================
type FuturesMarket struct {
	Id                     int64; ChainId int64; MarketAddress string; MarketName string
	BaseAsset, QuoteAsset  string; MaxLeverage string; MaintenanceMarginRate string
	TakerFeeRate, MakerFeeRate string; FundingInterval int; IsActive bool
	CreatedAt, UpdatedAt   time.Time
}

type FuturesMarketModel struct{ db *sql.DB }
func NewFuturesMarketModel(db *sql.DB) *FuturesMarketModel { return &FuturesMarketModel{db: db} }

func (m *FuturesMarketModel) ListMarkets(ctx context.Context, chainId int64, page, pageSize int64) ([]*FuturesMarket, int64, error) {
	var total int64
	m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM futures_markets WHERE chain_id = ? AND is_active = 1", chainId).Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, market_address, market_name, base_asset, quote_asset,
		 max_leverage, maintenance_margin_rate, taker_fee_rate, maker_fee_rate,
		 funding_interval, is_active, created_at, updated_at
		 FROM futures_markets WHERE chain_id = ? AND is_active = 1 ORDER BY id LIMIT ? OFFSET ?`,
		chainId, pageSize, offset)
	if err != nil { return nil, 0, err }
	defer rows.Close()

	var list []*FuturesMarket
	for rows.Next() {
		f := &FuturesMarket{}
		rows.Scan(&f.Id, &f.ChainId, &f.MarketAddress, &f.MarketName, &f.BaseAsset, &f.QuoteAsset,
			&f.MaxLeverage, &f.MaintenanceMarginRate, &f.TakerFeeRate, &f.MakerFeeRate,
			&f.FundingInterval, &f.IsActive, &f.CreatedAt, &f.UpdatedAt)
		list = append(list, f)
	}
	return list, total, nil
}

func (m *FuturesMarketModel) FindByAddress(ctx context.Context, chainId int64, address string) (*FuturesMarket, error) {
	f := &FuturesMarket{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, market_address, market_name, base_asset, quote_asset,
		 max_leverage, maintenance_margin_rate, taker_fee_rate, maker_fee_rate,
		 funding_interval, is_active, created_at, updated_at
		 FROM futures_markets WHERE chain_id = ? AND market_address = ?`,
		chainId, address).Scan(&f.Id, &f.ChainId, &f.MarketAddress, &f.MarketName, &f.BaseAsset, &f.QuoteAsset,
		&f.MaxLeverage, &f.MaintenanceMarginRate, &f.TakerFeeRate, &f.MakerFeeRate,
		&f.FundingInterval, &f.IsActive, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

// ==================== FuturesPositionModel ====================
type FuturesPosition struct {
	Id, ChainId, BlockNumber    int64
	PositionId, UserAddress     string
	MarketAddress, Side         string
	Size, EntryPrice, MarkPrice string
	Margin, Leverage            string
	UnrealizedPnl, RealizedPnl  string
	LiquidationPrice, Status    string
	OpenedAt                    time.Time
	ClosedAt                    sql.NullTime
	UpdatedAt                   time.Time
}

type FuturesPositionModel struct{ db *sql.DB }
func NewFuturesPositionModel(db *sql.DB) *FuturesPositionModel { return &FuturesPositionModel{db: db} }

func (m *FuturesPositionModel) ListByUser(ctx context.Context, chainId int64, userAddress string) ([]*FuturesPosition, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, position_id, user_address, market_address, side,
		 size, entry_price, mark_price, margin, leverage, unrealized_pnl, realized_pnl,
		 liquidation_price, status, opened_at, closed_at, updated_at
		 FROM futures_positions WHERE chain_id = ? AND user_address = ? AND status = 'open'`,
		chainId, userAddress)
	if err != nil { return nil, err }
	defer rows.Close()

	var list []*FuturesPosition
	for rows.Next() {
		p := &FuturesPosition{}
		rows.Scan(&p.Id, &p.ChainId, &p.PositionId, &p.UserAddress, &p.MarketAddress, &p.Side,
			&p.Size, &p.EntryPrice, &p.MarkPrice, &p.Margin, &p.Leverage,
			&p.UnrealizedPnl, &p.RealizedPnl, &p.LiquidationPrice, &p.Status,
			&p.OpenedAt, &p.ClosedAt, &p.UpdatedAt)
		list = append(list, p)
	}
	return list, nil
}

func (m *FuturesPositionModel) Upsert(ctx context.Context, pos *FuturesPosition) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO futures_positions (chain_id, position_id, user_address, market_address,
		 side, size, entry_price, mark_price, margin, leverage, unrealized_pnl, realized_pnl,
		 liquidation_price, status, opened_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		 size=VALUES(size), mark_price=VALUES(mark_price), margin=VALUES(margin),
		 unrealized_pnl=VALUES(unrealized_pnl), realized_pnl=VALUES(realized_pnl),
		 liquidation_price=VALUES(liquidation_price), status=VALUES(status), updated_at=NOW()`,
		pos.ChainId, pos.PositionId, pos.UserAddress, pos.MarketAddress,
		pos.Side, pos.Size, pos.EntryPrice, pos.MarkPrice, pos.Margin, pos.Leverage,
		pos.UnrealizedPnl, pos.RealizedPnl, pos.LiquidationPrice, pos.Status, pos.OpenedAt)
	return err
}

func (m *FuturesPositionModel) ClosePosition(ctx context.Context, chainId int64, positionId, status string) error {
	_, err := m.db.ExecContext(ctx,
		`UPDATE futures_positions SET status = ?, closed_at = NOW(), updated_at = NOW()
		 WHERE chain_id = ? AND position_id = ?`, status, chainId, positionId)
	return err
}

// ==================== FuturesFundingModel ====================
type FuturesFundingRecord struct {
	Id, ChainId, BlockNumber int64; MarketAddress string
	FundingRate, CumulativeRate, LongPay, ShortPay string
	SettlementTime time.Time; TxHash string
}

type FuturesFundingModel struct{ db *sql.DB }
func NewFuturesFundingModel(db *sql.DB) *FuturesFundingModel { return &FuturesFundingModel{db: db} }

func (m *FuturesFundingModel) Insert(ctx context.Context, r *FuturesFundingRecord) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO futures_funding_records (chain_id, market_address, funding_rate,
		 cumulative_rate, long_pay, short_pay, settlement_time, block_number, tx_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id=id`,
		r.ChainId, r.MarketAddress, r.FundingRate, r.CumulativeRate,
		r.LongPay, r.ShortPay, r.SettlementTime, r.BlockNumber, r.TxHash)
	return err
}

func (m *FuturesFundingModel) ListByMarket(ctx context.Context, chainId int64, marketAddr string, page, pageSize int64) ([]*FuturesFundingRecord, int64, error) {
	var total int64
	m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM futures_funding_records WHERE chain_id = ? AND market_address = ?", chainId, marketAddr).Scan(&total)
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, market_address, funding_rate, cumulative_rate, long_pay, short_pay, settlement_time, block_number, tx_hash
		 FROM futures_funding_records WHERE chain_id = ? AND market_address = ?
		 ORDER BY settlement_time DESC LIMIT ? OFFSET ?`, chainId, marketAddr, pageSize, (page-1)*pageSize)
	if err != nil { return nil, 0, err }
	defer rows.Close()

	var list []*FuturesFundingRecord
	for rows.Next() {
		r := &FuturesFundingRecord{}
		rows.Scan(&r.Id, &r.ChainId, &r.MarketAddress, &r.FundingRate, &r.CumulativeRate, &r.LongPay, &r.ShortPay, &r.SettlementTime, &r.BlockNumber, &r.TxHash)
		list = append(list, r)
	}
	return list, total, nil
}

// ==================== FuturesLiquidationModel ====================
type FuturesLiquidation struct {
	Id, ChainId, BlockNumber int64; PositionId, UserAddress, MarketAddress, Liquidator string
	Side, Size, LiquidationPrice, Penalty string; BlockTime time.Time; TxHash string; LogIndex int
}

type FuturesLiquidationModel struct{ db *sql.DB }
func NewFuturesLiquidationModel(db *sql.DB) *FuturesLiquidationModel { return &FuturesLiquidationModel{db: db} }

func (m *FuturesLiquidationModel) Insert(ctx context.Context, l *FuturesLiquidation) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO futures_liquidations (chain_id, position_id, user_address, market_address,
		 liquidator, side, size, liquidation_price, penalty, block_number, block_time, tx_hash, log_index)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE id=id`,
		l.ChainId, l.PositionId, l.UserAddress, l.MarketAddress, l.Liquidator,
		l.Side, l.Size, l.LiquidationPrice, l.Penalty, l.BlockNumber, l.BlockTime, l.TxHash, l.LogIndex)
	return err
}

var _ = fmt.Sprintf // ensure import
