package model

import (
	"context"; "database/sql"; "time"
)

// ==================== ProposalModel ====================
type Proposal struct {
	Id, ChainId, BlockNumber int64; ProposalId, ProposerAddress, Title, Description string
	Status string; ForVotes, AgainstVotes, AbstainVotes, QuorumRequired string
	StartTime, EndTime sql.NullTime
	ExecutedAt sql.NullTime; CreatedAt, UpdatedAt time.Time
}
type ProposalModel struct{ db *sql.DB }
func NewProposalModel(db *sql.DB) *ProposalModel { return &ProposalModel{db: db} }

func (m *ProposalModel) ListProposals(ctx context.Context, chainId int64, status string, page, pageSize int64) ([]*Proposal, int64, error) {
	where := "chain_id = ?"
	args := []interface{}{chainId}
	if status != "" { where += " AND status = ?"; args = append(args, status) }

	var total int64
	m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM gov_proposals WHERE "+where, args...).Scan(&total)

	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, proposal_id, proposer_address, title, description, status,
		 for_votes, against_votes, abstain_votes, quorum_required,
		 start_time, end_time, executed_at, created_at, updated_at
		 FROM gov_proposals WHERE `+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		append(args, pageSize, (page-1)*pageSize)...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var list []*Proposal
	for rows.Next() {
		p := &Proposal{}
		rows.Scan(&p.Id, &p.ChainId, &p.ProposalId, &p.ProposerAddress, &p.Title, &p.Description,
			&p.Status, &p.ForVotes, &p.AgainstVotes, &p.AbstainVotes, &p.QuorumRequired,
			&p.StartTime, &p.EndTime, &p.ExecutedAt, &p.CreatedAt, &p.UpdatedAt)
		list = append(list, p)
	}
	return list, total, nil
}

func (m *ProposalModel) Upsert(ctx context.Context, p *Proposal) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO gov_proposals (chain_id, proposal_id, proposer_address, title, description,
		 status, for_votes, against_votes, abstain_votes, quorum_required, start_time, end_time, block_number)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE status=VALUES(status), for_votes=VALUES(for_votes),
		 against_votes=VALUES(against_votes), abstain_votes=VALUES(abstain_votes), updated_at=NOW()`,
		p.ChainId, p.ProposalId, p.ProposerAddress, p.Title, p.Description,
		p.Status, p.ForVotes, p.AgainstVotes, p.AbstainVotes, p.QuorumRequired,
		p.StartTime, p.EndTime, p.BlockNumber)
	return err
}

func (m *ProposalModel) UpdateStatus(ctx context.Context, chainId int64, proposalId, status string) error {
	_, err := m.db.ExecContext(ctx,
		`UPDATE gov_proposals SET status = ?, updated_at = NOW() WHERE chain_id = ? AND proposal_id = ?`,
		status, chainId, proposalId)
	return err
}

// ==================== VoteModel ====================
type Vote struct {
	Id, ChainId, BlockNumber int64; ProposalId, VoterAddress string
	VoteType int; VotingPower, Reason string
	BlockTime time.Time; TxHash string
}
type VoteModel struct{ db *sql.DB }
func NewVoteModel(db *sql.DB) *VoteModel { return &VoteModel{db: db} }

func (m *VoteModel) Insert(ctx context.Context, v *Vote) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO gov_votes (chain_id, proposal_id, voter_address, vote_type, voting_power,
		 reason, block_number, block_time, tx_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE id=id`,
		v.ChainId, v.ProposalId, v.VoterAddress, v.VoteType, v.VotingPower,
		v.Reason, v.BlockNumber, v.BlockTime, v.TxHash)
	return err
}

func (m *VoteModel) ListByProposal(ctx context.Context, chainId int64, proposalId string, page, pageSize int64) ([]*Vote, int64, error) {
	var total int64
	m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM gov_votes WHERE chain_id = ? AND proposal_id = ?", chainId, proposalId).Scan(&total)

	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, proposal_id, voter_address, vote_type, voting_power, reason,
		 block_number, block_time, tx_hash
		 FROM gov_votes WHERE chain_id = ? AND proposal_id = ? ORDER BY block_time DESC LIMIT ? OFFSET ?`,
		chainId, proposalId, pageSize, (page-1)*pageSize)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var list []*Vote
	for rows.Next() {
		v := &Vote{}
		rows.Scan(&v.Id, &v.ChainId, &v.ProposalId, &v.VoterAddress, &v.VoteType, &v.VotingPower,
			&v.Reason, &v.BlockNumber, &v.BlockTime, &v.TxHash)
		list = append(list, v)
	}
	return list, total, nil
}

// ==================== VeLockModel ====================
type VeLock struct {
	Id, ChainId int64; UserAddress, LockAmount, VotingPower string
	UnlockTime time.Time; CreatedAt, UpdatedAt time.Time
}
type VeLockModel struct{ db *sql.DB }
func NewVeLockModel(db *sql.DB) *VeLockModel { return &VeLockModel{db: db} }

func (m *VeLockModel) Upsert(ctx context.Context, l *VeLock) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO gov_ve_locks (chain_id, user_address, lock_amount, voting_power, unlock_time)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE lock_amount=VALUES(lock_amount), voting_power=VALUES(voting_power),
		 unlock_time=VALUES(unlock_time), updated_at=NOW()`,
		l.ChainId, l.UserAddress, l.LockAmount, l.VotingPower, l.UnlockTime)
	return err
}

func (m *VeLockModel) GetByUser(ctx context.Context, chainId int64, user string) (*VeLock, error) {
	l := &VeLock{}
	err := m.db.QueryRowContext(ctx,
		`SELECT id, chain_id, user_address, lock_amount, voting_power, unlock_time, created_at, updated_at
		 FROM gov_ve_locks WHERE chain_id = ? AND user_address = ?`, chainId, user).Scan(
		&l.Id, &l.ChainId, &l.UserAddress, &l.LockAmount, &l.VotingPower, &l.UnlockTime, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

// ==================== GaugeVoteModel ====================
type GaugeVote struct {
	Id, ChainId, Epoch int64; VoterAddress, GaugeAddress, Weight string
	CreatedAt time.Time
}
type GaugeVoteModel struct{ db *sql.DB }
func NewGaugeVoteModel(db *sql.DB) *GaugeVoteModel { return &GaugeVoteModel{db: db} }

func (m *GaugeVoteModel) ListByEpoch(ctx context.Context, chainId, epoch int64) ([]*GaugeVote, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, chain_id, epoch, voter_address, gauge_address, weight, created_at
		 FROM gov_gauge_votes WHERE chain_id = ? AND epoch = ? ORDER BY weight DESC`, chainId, epoch)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*GaugeVote
	for rows.Next() {
		g := &GaugeVote{}
		rows.Scan(&g.Id, &g.ChainId, &g.Epoch, &g.VoterAddress, &g.GaugeAddress, &g.Weight, &g.CreatedAt)
		list = append(list, g)
	}
	return list, nil
}
