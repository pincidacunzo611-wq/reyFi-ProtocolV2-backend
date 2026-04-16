package consumer

import (
	"context"; "database/sql"; "encoding/json"; "time"
	"github.com/segmentio/kafka-go"
	"github.com/reyfi/reyfi-backend/app/governance/internal/model"
	"github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type GovernanceEventConsumer struct {
	db *sql.DB; chainId int64; redis *redis.Redis
	proposalModel *model.ProposalModel; voteModel *model.VoteModel; veLockModel *model.VeLockModel
}

func NewGovernanceEventConsumer(db *sql.DB, chainId int64, rds *redis.Redis) *GovernanceEventConsumer {
	return &GovernanceEventConsumer{db: db, chainId: chainId, redis: rds,
		proposalModel: model.NewProposalModel(db), voteModel: model.NewVoteModel(db),
		veLockModel: model.NewVeLockModel(db)}
}

func (c *GovernanceEventConsumer) Start(ctx context.Context, brokers []string, groupId, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers, GroupID: groupId, Topic: topic,
		MinBytes: 1, MaxBytes: 10e6, CommitInterval: time.Second, StartOffset: kafka.LastOffset,
	})
	defer reader.Close()
	logx.Infof("governance consumer started: topic=%s", topic)
	for {
		select { case <-ctx.Done(): return nil; default: }
		msg, err := reader.ReadMessage(ctx)
		if err != nil { logx.Errorf("read: %v", err); continue }
		if err := c.handle(ctx, msg.Value); err != nil { logx.Errorf("handle: %v", err) }
	}
}

func (c *GovernanceEventConsumer) handle(ctx context.Context, value []byte) error {
	event, err := chains.UnmarshalChainEvent(value)
	if err != nil { return err }
	switch event.EventName {
	case chains.EventProposalCreated:
		return c.handleProposalCreated(ctx, event)
	case chains.EventVoteCast:
		return c.handleVoteCast(ctx, event)
	case chains.EventProposalExecuted:
		return c.handleProposalExecuted(ctx, event)
	case chains.EventLockCreated:
		return c.handleLockCreated(ctx, event)
	default:
		return nil
	}
}

func (c *GovernanceEventConsumer) handleProposalCreated(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.ProposalCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil { return err }

	proposal := &model.Proposal{
		ChainId: event.ChainId, ProposalId: payload.ProposalId,
		ProposerAddress: payload.Proposer, Title: payload.Description,
		Description: payload.Description, Status: "active",
		ForVotes: "0", AgainstVotes: "0", AbstainVotes: "0", QuorumRequired: "0",
		BlockNumber: event.BlockNumber,
	}
	if err := c.proposalModel.Upsert(ctx, proposal); err != nil { return err }

	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'governance', 'propose', ?, ?, ?)`,
		payload.Proposer, "创建治理提案", event.TxHash, event.GetBlockTime())

	logx.Infof("proposal created: id=%s, proposer=%s", payload.ProposalId, payload.Proposer)
	return nil
}

func (c *GovernanceEventConsumer) handleVoteCast(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.VoteCastPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil { return err }

	vote := &model.Vote{
		ChainId: event.ChainId, ProposalId: payload.ProposalId,
		VoterAddress: payload.Voter, VoteType: payload.Support,
		VotingPower: payload.Weight, Reason: payload.Reason,
		BlockNumber: event.BlockNumber, BlockTime: event.GetBlockTime(), TxHash: event.TxHash,
	}
	if err := c.voteModel.Insert(ctx, vote); err != nil { return err }

	// 更新提案投票统计
	voteTypeLabel := "赞成"
	switch payload.Support {
	case 0: voteTypeLabel = "反对"
	case 2: voteTypeLabel = "弃权"
	}

	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'governance', 'vote', ?, ?, ?)`,
		payload.Voter, voteTypeLabel+"投票 "+payload.Weight, event.TxHash, event.GetBlockTime())

	logx.Infof("vote cast: voter=%s, proposal=%s, support=%d", payload.Voter, payload.ProposalId, payload.Support)
	return nil
}

func (c *GovernanceEventConsumer) handleProposalExecuted(ctx context.Context, event *chains.ChainEvent) error {
	var payload map[string]interface{}
	json.Unmarshal(event.Payload, &payload)
	logx.Infof("proposal executed: block=%d", event.BlockNumber)
	return nil
}

func (c *GovernanceEventConsumer) handleLockCreated(ctx context.Context, event *chains.ChainEvent) error {
	var payload chains.LockCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil { return err }

	unlockTime := time.Unix(payload.UnlockTime, 0).UTC()
	lock := &model.VeLock{
		ChainId: event.ChainId, UserAddress: payload.User,
		LockAmount: payload.Amount, VotingPower: payload.Amount, // 简化，实际按时间衰减
		UnlockTime: unlockTime,
	}
	if err := c.veLockModel.Upsert(ctx, lock); err != nil { return err }

	c.db.ExecContext(ctx,
		`INSERT INTO user_activity_stream (wallet_address, module, action, summary, tx_hash, block_time)
		 VALUES (?, 'governance', 'lock', ?, ?, ?)`,
		payload.User, "锁仓 "+payload.Amount+" 获取投票权", event.TxHash, event.GetBlockTime())

	logx.Infof("ve lock created: user=%s, amount=%s", payload.User, payload.Amount)
	return nil
}
