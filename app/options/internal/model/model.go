package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ==================== OptionsMarketModel ====================
type OptionsMarket struct {
	Id int64; ChainId int64; MarketAddress, UnderlyingAddress, UnderlyingSymbol, SettlementAsset string
	IsActive bool; CreatedAt, UpdatedAt time.Time
}

type OptionsMarketModel struct{ db *sql.DB }
func NewOptionsMarketModel(db *sql.DB) *OptionsMarketModel { return &OptionsMarketModel{db: db} }

func (m *OptionsMarketModel) ListMarkets(ctx context.Context, chainId int64, page, pageSize int64) ([]*OptionsMarket, int64, error) {
	var total int64
	m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM options_markets WHERE chain_id = ? AND is_active = 1", chainId).Scan(&total)
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, market_address, underlying_address, underlying_symbol,
		 settlement_asset, is_active, created_at, updated_at
		 FROM options_markets WHERE chain_id = ? AND is_active = 1 ORDER BY id LIMIT ? OFFSET ?`,
		chainId, pageSize, (page-1)*pageSize)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var list []*OptionsMarket
	for rows.Next() {
		o := &OptionsMarket{}
		rows.Scan(&o.Id, &o.ChainId, &o.MarketAddress, &o.UnderlyingAddress, &o.UnderlyingSymbol,
			&o.SettlementAsset, &o.IsActive, &o.CreatedAt, &o.UpdatedAt)
		list = append(list, o)
	}
	return list, total, nil
}

// ==================== OptionsPositionModel ====================
type OptionsPosition struct {
	Id, ChainId               int64
	OptionId, UserAddress     string
	MarketAddress             string
	UnderlyingAddress         string
	StrikePrice, Premium      string
	Size, OptionType          string
	ExpiryTime                time.Time
	SettlementPrice, Pnl      sql.NullString
	Status                    string
	CreatedAt, UpdatedAt      time.Time
}

type OptionsPositionModel struct{ db *sql.DB }
func NewOptionsPositionModel(db *sql.DB) *OptionsPositionModel { return &OptionsPositionModel{db: db} }

func (m *OptionsPositionModel) ListByUser(ctx context.Context, chainId int64, userAddress string) ([]*OptionsPosition, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, option_id, user_address, market_address, underlying_address,
		 strike_price, premium, size, option_type, expiry_time, settlement_price, pnl,
		 status, created_at, updated_at
		 FROM options_positions WHERE chain_id = ? AND user_address = ?
		 ORDER BY created_at DESC LIMIT 100`, chainId, userAddress)
	if err != nil { return nil, err }
	defer rows.Close()

	var list []*OptionsPosition
	for rows.Next() {
		p := &OptionsPosition{}
		rows.Scan(&p.Id, &p.ChainId, &p.OptionId, &p.UserAddress, &p.MarketAddress,
			&p.UnderlyingAddress, &p.StrikePrice, &p.Premium, &p.Size, &p.OptionType,
			&p.ExpiryTime, &p.SettlementPrice, &p.Pnl, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		list = append(list, p)
	}
	return list, nil
}

func (m *OptionsPositionModel) Upsert(ctx context.Context, pos *OptionsPosition) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO options_positions (chain_id, option_id, user_address, market_address,
		 underlying_address, strike_price, premium, size, option_type, expiry_time, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE status=VALUES(status), updated_at=NOW()`,
		pos.ChainId, pos.OptionId, pos.UserAddress, pos.MarketAddress,
		pos.UnderlyingAddress, pos.StrikePrice, pos.Premium, pos.Size,
		pos.OptionType, pos.ExpiryTime, pos.Status)
	return err
}

func (m *OptionsPositionModel) UpdateStatus(ctx context.Context, chainId int64, optionId, status string, settlementPrice, pnl string) error {
	_, err := m.db.ExecContext(ctx,
		`UPDATE options_positions SET status = ?, settlement_price = ?, pnl = ?, updated_at = NOW()
		 WHERE chain_id = ? AND option_id = ?`,
		status, settlementPrice, pnl, chainId, optionId)
	return err
}

// ==================== OptionsSettlementModel ====================
type OptionsSettlement struct {
	Id, ChainId, BlockNumber int64
	OptionId, UserAddress, Action string
	SettlementPrice, PayoutAmount string
	BlockTime time.Time; TxHash string; LogIndex int
}

type OptionsSettlementModel struct{ db *sql.DB }
func NewOptionsSettlementModel(db *sql.DB) *OptionsSettlementModel { return &OptionsSettlementModel{db: db} }

func (m *OptionsSettlementModel) Insert(ctx context.Context, s *OptionsSettlement) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO options_settlements (chain_id, option_id, user_address, action,
		 settlement_price, payout_amount, block_number, block_time, tx_hash, log_index)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE id=id`,
		s.ChainId, s.OptionId, s.UserAddress, s.Action, s.SettlementPrice,
		s.PayoutAmount, s.BlockNumber, s.BlockTime, s.TxHash, s.LogIndex)
	return err
}

// ==================== OptionsVolSurfaceModel ====================
type OptionsVolSurface struct {
	Id, ChainId int64; Underlying string; StrikePrice string
	ExpiryTime time.Time; ImpliedVol, HistoricalVol sql.NullString; SnapshotTime time.Time
}

type OptionsVolSurfaceModel struct{ db *sql.DB }
func NewOptionsVolSurfaceModel(db *sql.DB) *OptionsVolSurfaceModel { return &OptionsVolSurfaceModel{db: db} }

func (m *OptionsVolSurfaceModel) GetLatestByUnderlying(ctx context.Context, chainId int64, underlying string) ([]*OptionsVolSurface, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, underlying, strike_price, expiry_time, implied_vol, historical_vol, snapshot_time
		 FROM options_vol_surfaces WHERE chain_id = ? AND underlying = ?
		 ORDER BY snapshot_time DESC LIMIT 50`, chainId, underlying)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*OptionsVolSurface
	for rows.Next() {
		v := &OptionsVolSurface{}
		rows.Scan(&v.Id, &v.ChainId, &v.Underlying, &v.StrikePrice, &v.ExpiryTime,
			&v.ImpliedVol, &v.HistoricalVol, &v.SnapshotTime)
		list = append(list, v)
	}
	return list, nil
}

var _ = fmt.Sprintf
