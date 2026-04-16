package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reyfi/reyfi-backend/app/governance/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

type GovernanceServiceServer struct{ svcCtx *svc.ServiceContext }

func NewGovernanceServiceServer(svcCtx *svc.ServiceContext) *GovernanceServiceServer {
	return &GovernanceServiceServer{svcCtx: svcCtx}
}
func RegisterGovernanceServiceServer(s *grpc.Server, srv *GovernanceServiceServer) {
	logx.Info("governance service registered")
}

// ==================== GetProposals ====================

func (s *GovernanceServiceServer) GetProposals(ctx context.Context, status string, page, pageSize int64) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	cacheKey := fmt.Sprintf("gov:proposals:%s:%d", status, page)
	if c, err := s.svcCtx.Redis.Get(cacheKey); err == nil && c != "" {
		var r interface{}
		json.Unmarshal([]byte(c), &r)
		return r, nil
	}
	proposals, total, err := s.svcCtx.ProposalModel.ListProposals(ctx, chainId, status, page, pageSize)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]interface{}, 0, len(proposals))
	for _, p := range proposals {
		item := map[string]interface{}{
			"proposalId": p.ProposalId, "proposer": p.ProposerAddress,
			"title": p.Title, "status": p.Status,
			"forVotes": p.ForVotes, "againstVotes": p.AgainstVotes, "abstainVotes": p.AbstainVotes,
			"quorumRequired": p.QuorumRequired,
			"createdAt":      p.CreatedAt.UTC().Format(time.RFC3339),
		}
		if p.StartTime.Valid {
			item["startTime"] = p.StartTime.Time.UTC().Format(time.RFC3339)
		}
		if p.EndTime.Valid {
			item["endTime"] = p.EndTime.Time.UTC().Format(time.RFC3339)
		}
		list = append(list, item)
	}
	resp := map[string]interface{}{"list": list, "total": total}
	if d, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(d), 10)
	}
	return resp, nil
}

// ==================== GetProposalDetail ====================

func (s *GovernanceServiceServer) GetProposalDetail(ctx context.Context, proposalId string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	proposal, err := s.svcCtx.ProposalModel.FindById(ctx, chainId, proposalId)
	if err != nil {
		return nil, fmt.Errorf("proposal not found: %w", err)
	}

	item := map[string]interface{}{
		"proposalId": proposal.ProposalId, "proposer": proposal.ProposerAddress,
		"title": proposal.Title, "description": proposal.Description,
		"status": proposal.Status,
		"forVotes": proposal.ForVotes, "againstVotes": proposal.AgainstVotes,
		"abstainVotes": proposal.AbstainVotes, "quorumRequired": proposal.QuorumRequired,
		"createdAt": proposal.CreatedAt.UTC().Format(time.RFC3339),
	}
	if proposal.StartTime.Valid {
		item["startTime"] = proposal.StartTime.Time.UTC().Format(time.RFC3339)
	}
	if proposal.EndTime.Valid {
		item["endTime"] = proposal.EndTime.Time.UTC().Format(time.RFC3339)
	}

	// 查询关联的 targets 和 calldatas（如果存在）
	var snapshotBlock string
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(snapshot_block, '0') FROM governance_proposals
		 WHERE chain_id = ? AND proposal_id = ?`,
		chainId, proposalId).Scan(&snapshotBlock)

	return map[string]interface{}{
		"proposal":      item,
		"targets":       []string{},
		"calldatas":     []string{},
		"snapshotBlock": snapshotBlock,
	}, nil
}

// ==================== GetVotes ====================

func (s *GovernanceServiceServer) GetVotes(ctx context.Context, proposalId string, page, pageSize int64) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	votes, total, err := s.svcCtx.VoteModel.ListByProposal(ctx, chainId, proposalId, page, pageSize)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]interface{}, 0, len(votes))
	for _, v := range votes {
		voteType := "for"
		if v.VoteType == 0 {
			voteType = "against"
		} else if v.VoteType == 2 {
			voteType = "abstain"
		}
		list = append(list, map[string]interface{}{
			"voter": v.VoterAddress, "voteType": voteType,
			"votingPower": v.VotingPower, "reason": v.Reason,
			"blockTime": v.BlockTime.UTC().Format(time.RFC3339), "txHash": v.TxHash,
		})
	}
	return map[string]interface{}{"list": list, "total": total}, nil
}

// ==================== GetVeLock ====================

func (s *GovernanceServiceServer) GetVeLock(ctx context.Context, userAddress string) (interface{}, error) {
	chainId := s.svcCtx.Config.Chain.ChainId
	lock, err := s.svcCtx.VeLockModel.GetByUser(ctx, chainId, userAddress)
	if err != nil {
		return map[string]interface{}{"lockAmount": "0", "votingPower": "0", "unlockTime": ""}, nil
	}
	return map[string]interface{}{
		"lockAmount": lock.LockAmount, "votingPower": lock.VotingPower,
		"unlockTime": lock.UnlockTime.UTC().Format(time.RFC3339),
	}, nil
}

// ==================== Build* 交易构建 ====================

func (s *GovernanceServiceServer) BuildVote(ctx context.Context, userAddr, proposalId string, support int, reason string) (interface{}, error) {
	logx.Infof("build vote: user=%s, proposal=%s, support=%d", userAddr, proposalId, support)
	governor := s.svcCtx.Config.Chain.Contracts["governor"]
	if governor == "" {
		governor = s.svcCtx.Config.Chain.Contracts["Governor"]
	}
	data, _ := json.Marshal(map[string]interface{}{
		"method": "castVoteWithReason",
		"params": map[string]interface{}{
			"proposalId": proposalId, "support": support, "reason": reason,
		},
	})
	return map[string]interface{}{
		"to": governor, "value": "0", "gasLimit": "200000",
		"data": string(data),
	}, nil
}

func (s *GovernanceServiceServer) BuildLock(ctx context.Context, userAddr, amount string, duration int64) (interface{}, error) {
	logx.Infof("build lock: user=%s, amount=%s, duration=%d", userAddr, amount, duration)
	veToken := s.svcCtx.Config.Chain.Contracts["veToken"]
	if veToken == "" {
		veToken = s.svcCtx.Config.Chain.Contracts["VeToken"]
	}
	unlockTime := time.Now().UTC().Add(time.Duration(duration) * time.Second).Unix()
	data, _ := json.Marshal(map[string]interface{}{
		"method": "createLock",
		"params": map[string]interface{}{
			"amount": amount, "unlockTime": unlockTime,
		},
	})
	return map[string]interface{}{
		"to": veToken, "value": "0", "gasLimit": "300000",
		"data": string(data),
	}, nil
}
