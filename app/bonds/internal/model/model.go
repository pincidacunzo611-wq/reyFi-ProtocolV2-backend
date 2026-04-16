package model

import (
	"context"; "database/sql"; "time"
)

type BondMarket struct {
	Id, ChainId int64; MarketAddress, PaymentToken, PayoutToken string
	VestingTerm int64; DiscountRate, MinPrice, MaxDebt, CurrentDebt string
	IsActive bool; CreatedAt, UpdatedAt time.Time
}
type BondMarketModel struct{ db *sql.DB }
func NewBondMarketModel(db *sql.DB) *BondMarketModel { return &BondMarketModel{db: db} }

func (m *BondMarketModel) ListMarkets(ctx context.Context, chainId int64, page, pageSize int64) ([]*BondMarket, int64, error) {
	var total int64
	m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bond_markets WHERE chain_id = ? AND is_active = 1", chainId).Scan(&total)
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, market_address, payment_token, payout_token,
		 vesting_term, discount_rate, min_price, max_debt, current_debt, is_active, created_at, updated_at
		 FROM bond_markets WHERE chain_id = ? AND is_active = 1 ORDER BY id LIMIT ? OFFSET ?`,
		chainId, pageSize, (page-1)*pageSize)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var list []*BondMarket
	for rows.Next() {
		b := &BondMarket{}
		rows.Scan(&b.Id, &b.ChainId, &b.MarketAddress, &b.PaymentToken, &b.PayoutToken,
			&b.VestingTerm, &b.DiscountRate, &b.MinPrice, &b.MaxDebt, &b.CurrentDebt,
			&b.IsActive, &b.CreatedAt, &b.UpdatedAt)
		list = append(list, b)
	}
	return list, total, nil
}

func (m *BondMarketModel) FindByAddress(ctx context.Context, chainId int64, marketAddress string) (*BondMarket, error) {
	b := &BondMarket{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, market_address, payment_token, payout_token,
		 vesting_term, discount_rate, min_price, max_debt, current_debt, is_active, created_at, updated_at
		 FROM bond_markets WHERE chain_id = ? AND market_address = ?`,
		chainId, marketAddress).Scan(&b.Id, &b.ChainId, &b.MarketAddress, &b.PaymentToken, &b.PayoutToken,
		&b.VestingTerm, &b.DiscountRate, &b.MinPrice, &b.MaxDebt, &b.CurrentDebt,
		&b.IsActive, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (m *BondMarketModel) Upsert(ctx context.Context, b *BondMarket) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO bond_markets (chain_id, market_address, payment_token, payout_token,
		 vesting_term, discount_rate, min_price, max_debt, current_debt, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE discount_rate=VALUES(discount_rate), current_debt=VALUES(current_debt), updated_at=NOW()`,
		b.ChainId, b.MarketAddress, b.PaymentToken, b.PayoutToken, b.VestingTerm,
		b.DiscountRate, b.MinPrice, b.MaxDebt, b.CurrentDebt, b.IsActive)
	return err
}

type BondPosition struct {
	Id, ChainId, BlockNumber int64; BondId, UserAddress, MarketAddress string
	PaymentAmount, PayoutAmount, VestedAmount, ClaimableAmount string
	VestingStart, VestingEnd time.Time; Status string; CreatedAt, UpdatedAt time.Time
}
type BondPositionModel struct{ db *sql.DB }
func NewBondPositionModel(db *sql.DB) *BondPositionModel { return &BondPositionModel{db: db} }

func (m *BondPositionModel) ListByUser(ctx context.Context, chainId int64, user string) ([]*BondPosition, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, bond_id, user_address, market_address, payment_amount, payout_amount,
		 vested_amount, claimable_amount, vesting_start, vesting_end, status, created_at, updated_at
		 FROM bond_positions WHERE chain_id = ? AND user_address = ? ORDER BY created_at DESC`, chainId, user)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*BondPosition
	for rows.Next() {
		p := &BondPosition{}
		rows.Scan(&p.Id, &p.ChainId, &p.BondId, &p.UserAddress, &p.MarketAddress,
			&p.PaymentAmount, &p.PayoutAmount, &p.VestedAmount, &p.ClaimableAmount,
			&p.VestingStart, &p.VestingEnd, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		list = append(list, p)
	}
	return list, nil
}

func (m *BondPositionModel) Upsert(ctx context.Context, p *BondPosition) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO bond_positions (chain_id, bond_id, user_address, market_address,
		 payment_amount, payout_amount, vested_amount, claimable_amount,
		 vesting_start, vesting_end, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE vested_amount=VALUES(vested_amount),
		 claimable_amount=VALUES(claimable_amount), status=VALUES(status), updated_at=NOW()`,
		p.ChainId, p.BondId, p.UserAddress, p.MarketAddress,
		p.PaymentAmount, p.PayoutAmount, p.VestedAmount, p.ClaimableAmount,
		p.VestingStart, p.VestingEnd, p.Status)
	return err
}

type BondRedemption struct {
	Id, ChainId, BlockNumber int64; BondId, UserAddress string; Amount string
	BlockTime time.Time; TxHash string; LogIndex int
}
type BondRedemptionModel struct{ db *sql.DB }
func NewBondRedemptionModel(db *sql.DB) *BondRedemptionModel { return &BondRedemptionModel{db: db} }

func (m *BondRedemptionModel) Insert(ctx context.Context, r *BondRedemption) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO bond_redemptions (chain_id, bond_id, user_address, amount, block_number, block_time, tx_hash, log_index)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE id=id`,
		r.ChainId, r.BondId, r.UserAddress, r.Amount, r.BlockNumber, r.BlockTime, r.TxHash, r.LogIndex)
	return err
}
