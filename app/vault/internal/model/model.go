package model

import (
	"context"; "database/sql"; "time"
)

type Vault struct {
	Id int64; ChainId int64; VaultAddress, AssetAddress, AssetSymbol, StrategyType, Name, Symbol string
	IsActive bool; CreatedAt, UpdatedAt time.Time
}
type VaultModel struct{ db *sql.DB }
func NewVaultModel(db *sql.DB) *VaultModel { return &VaultModel{db: db} }

func (m *VaultModel) ListVaults(ctx context.Context, chainId int64, page, pageSize int64) ([]*Vault, int64, error) {
	var total int64
	m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vaults WHERE chain_id = ? AND is_active = 1", chainId).Scan(&total)
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, vault_address, asset_address, asset_symbol, strategy_type, name, symbol, is_active, created_at, updated_at
		 FROM vaults WHERE chain_id = ? AND is_active = 1 ORDER BY id LIMIT ? OFFSET ?`, chainId, pageSize, (page-1)*pageSize)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var list []*Vault
	for rows.Next() {
		v := &Vault{}
		rows.Scan(&v.Id, &v.ChainId, &v.VaultAddress, &v.AssetAddress, &v.AssetSymbol, &v.StrategyType, &v.Name, &v.Symbol, &v.IsActive, &v.CreatedAt, &v.UpdatedAt)
		list = append(list, v)
	}
	return list, total, nil
}

func (m *VaultModel) FindByAddress(ctx context.Context, chainId int64, addr string) (*Vault, error) {
	v := &Vault{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, vault_address, asset_address, asset_symbol, strategy_type, name, symbol, is_active, created_at, updated_at
		 FROM vaults WHERE chain_id = ? AND vault_address = ?`, chainId, addr).Scan(
		&v.Id, &v.ChainId, &v.VaultAddress, &v.AssetAddress, &v.AssetSymbol, &v.StrategyType, &v.Name, &v.Symbol, &v.IsActive, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

type VaultSnapshot struct {
	Id, ChainId int64; VaultAddress string; TotalAssets, TotalShares, Nav, TvlUsd string
	Apr7d, Apr30d, MaxDrawdown sql.NullString; SnapshotTime time.Time
}
type VaultSnapshotModel struct{ db *sql.DB }
func NewVaultSnapshotModel(db *sql.DB) *VaultSnapshotModel { return &VaultSnapshotModel{db: db} }

func (m *VaultSnapshotModel) GetLatest(ctx context.Context, chainId int64, vaultAddr string) (*VaultSnapshot, error) {
	s := &VaultSnapshot{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, vault_address, total_assets, total_shares, nav, tvl_usd, apr_7d, apr_30d, max_drawdown, snapshot_time
		 FROM vault_snapshots WHERE chain_id = ? AND vault_address = ? ORDER BY snapshot_time DESC LIMIT 1`, chainId, vaultAddr).Scan(
		&s.Id, &s.ChainId, &s.VaultAddress, &s.TotalAssets, &s.TotalShares, &s.Nav, &s.TvlUsd, &s.Apr7d, &s.Apr30d, &s.MaxDrawdown, &s.SnapshotTime)
	return s, err
}

func (m *VaultSnapshotModel) Insert(ctx context.Context, s *VaultSnapshot) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO vault_snapshots (chain_id, vault_address, total_assets, total_shares, nav, tvl_usd, apr_7d, apr_30d, max_drawdown, snapshot_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ChainId, s.VaultAddress, s.TotalAssets, s.TotalShares, s.Nav, s.TvlUsd, s.Apr7d, s.Apr30d, s.MaxDrawdown, s.SnapshotTime)
	return err
}

type VaultUserPosition struct {
	Id, ChainId int64; UserAddress, VaultAddress, Shares, CostBasis, DepositedTotal, WithdrawnTotal string; UpdatedAt time.Time
}
type VaultUserPositionModel struct{ db *sql.DB }
func NewVaultUserPositionModel(db *sql.DB) *VaultUserPositionModel { return &VaultUserPositionModel{db: db} }

func (m *VaultUserPositionModel) ListByUser(ctx context.Context, chainId int64, user string) ([]*VaultUserPosition, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, user_address, vault_address, shares, cost_basis, deposited_total, withdrawn_total, updated_at
		 FROM vault_user_positions WHERE chain_id = ? AND user_address = ?`, chainId, user)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*VaultUserPosition
	for rows.Next() {
		p := &VaultUserPosition{}
		rows.Scan(&p.Id, &p.ChainId, &p.UserAddress, &p.VaultAddress, &p.Shares, &p.CostBasis, &p.DepositedTotal, &p.WithdrawnTotal, &p.UpdatedAt)
		list = append(list, p)
	}
	return list, nil
}

type VaultHarvest struct {
	Id, ChainId, BlockNumber int64; VaultAddress, Strategy, Profit, ProfitUsd, NavBefore, NavAfter string; BlockTime time.Time; TxHash string
}
type VaultHarvestModel struct{ db *sql.DB }
func NewVaultHarvestModel(db *sql.DB) *VaultHarvestModel { return &VaultHarvestModel{db: db} }

func (m *VaultHarvestModel) Insert(ctx context.Context, h *VaultHarvest) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO vault_harvest_records (chain_id, vault_address, strategy, profit, profit_usd, nav_before, nav_after, block_number, block_time, tx_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		h.ChainId, h.VaultAddress, h.Strategy, h.Profit, h.ProfitUsd, h.NavBefore, h.NavAfter, h.BlockNumber, h.BlockTime, h.TxHash)
	return err
}
