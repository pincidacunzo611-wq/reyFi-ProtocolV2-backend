package handler

import (
	"net/http"
	"strconv"

	"github.com/reyfi/reyfi-backend/app/gateway/internal/svc"
	"github.com/reyfi/reyfi-backend/app/governance/rpc/governance"
	"github.com/reyfi/reyfi-backend/pkg/middleware"
	"github.com/reyfi/reyfi-backend/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ==================== Governance Public Handlers ====================

func govGetProposalsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		resp, err := svcCtx.GovernanceRpc.GetProposals(r.Context(), &governance.GetProposalsReq{
			Page:     page,
			PageSize: pageSize,
			Status:   r.URL.Query().Get("status"),
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.SuccessWithPage(r.Context(), w, resp.List, page, pageSize, resp.Total)
	}
}

func govGetProposalDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ProposalId string `path:"proposalId"`
		}
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.GovernanceRpc.GetProposalDetail(r.Context(), &governance.GetProposalDetailReq{
			ProposalId: req.ProposalId,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func govGetVotesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pathReq struct {
			ProposalId string `path:"proposalId"`
		}
		if err := httpx.Parse(r, &pathReq); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		resp, err := svcCtx.GovernanceRpc.GetVotes(r.Context(), &governance.GetVotesReq{
			ProposalId: pathReq.ProposalId,
			Page:       page,
			PageSize:   pageSize,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.SuccessWithPage(r.Context(), w, resp.List, page, pageSize, resp.Total)
	}
}

// ==================== Governance Auth Handlers ====================

func govGetVeLockHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		resp, err := svcCtx.GovernanceRpc.GetVeLock(r.Context(), &governance.GetVeLockReq{
			UserAddress: walletAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func govBuildVoteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			ProposalId string `json:"proposalId"`
			Support    int    `json:"support"`
			Reason     string `json:"reason"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.GovernanceRpc.BuildVote(r.Context(), &governance.BuildVoteReq{
			UserAddress: walletAddress,
			ProposalId:  req.ProposalId,
			Support:     int32(req.Support),
			Reason:      req.Reason,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func govBuildLockHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			Amount   string `json:"amount"`
			Duration int64  `json:"duration"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.GovernanceRpc.BuildLock(r.Context(), &governance.BuildLockReq{
			UserAddress: walletAddress,
			Amount:      req.Amount,
			Duration:    req.Duration,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}
