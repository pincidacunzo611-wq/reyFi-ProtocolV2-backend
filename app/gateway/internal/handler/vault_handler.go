package handler

import (
	"net/http"
	"strconv"

	"github.com/reyfi/reyfi-backend/app/gateway/internal/svc"
	"github.com/reyfi/reyfi-backend/app/vault/rpc/vault"
	"github.com/reyfi/reyfi-backend/pkg/middleware"
	"github.com/reyfi/reyfi-backend/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ==================== Vault Public Handlers ====================

func vaultGetListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		resp, err := svcCtx.VaultRpc.GetVaults(r.Context(), &vault.GetVaultsReq{
			Page:     page,
			PageSize: pageSize,
			Keyword:  r.URL.Query().Get("keyword"),
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.SuccessWithPage(r.Context(), w, resp.List, page, pageSize, resp.Total)
	}
}

func vaultGetDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			VaultAddress string `path:"vaultAddress"`
		}
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.VaultRpc.GetVaultDetail(r.Context(), &vault.GetVaultDetailReq{
			VaultAddress: req.VaultAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

// ==================== Vault Auth Handlers ====================

func vaultGetPositionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		resp, err := svcCtx.VaultRpc.GetPositions(r.Context(), &vault.GetPositionsReq{
			UserAddress: walletAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func vaultBuildDepositHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			VaultAddress string `json:"vaultAddress"`
			Amount       string `json:"amount"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.VaultRpc.BuildDeposit(r.Context(), &vault.BuildDepositReq{
			UserAddress:  walletAddress,
			VaultAddress: req.VaultAddress,
			Amount:       req.Amount,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func vaultBuildWithdrawHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			VaultAddress string `json:"vaultAddress"`
			Shares       string `json:"shares"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.VaultRpc.BuildWithdraw(r.Context(), &vault.BuildWithdrawReq{
			UserAddress:  walletAddress,
			VaultAddress: req.VaultAddress,
			Shares:       req.Shares,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}
