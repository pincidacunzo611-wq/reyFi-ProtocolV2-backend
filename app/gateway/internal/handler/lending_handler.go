package handler

import (
	"net/http"
	"strconv"

	"github.com/reyfi/reyfi-backend/app/gateway/internal/svc"
	"github.com/reyfi/reyfi-backend/app/lending/rpc/lending"
	"github.com/reyfi/reyfi-backend/pkg/middleware"
	"github.com/reyfi/reyfi-backend/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ==================== Lending Public Handlers ====================

func lendingGetMarketsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		resp, err := svcCtx.LendingRpc.GetMarkets(r.Context(), &lending.GetMarketsReq{
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

func lendingGetMarketDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AssetAddress string `path:"assetAddress"`
		}
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.LendingRpc.GetMarketDetail(r.Context(), &lending.GetMarketDetailReq{
			AssetAddress: req.AssetAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func lendingGetLiquidationsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		resp, err := svcCtx.LendingRpc.GetLiquidations(r.Context(), &lending.GetLiquidationsReq{
			AssetAddress: r.URL.Query().Get("assetAddress"),
			UserAddress:  r.URL.Query().Get("userAddress"),
			Page:         page,
			PageSize:     pageSize,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.SuccessWithPage(r.Context(), w, resp.List, page, pageSize, resp.Total)
	}
}

// ==================== Lending Auth Handlers ====================

func lendingGetUserPositionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		resp, err := svcCtx.LendingRpc.GetUserPosition(r.Context(), &lending.GetUserPositionReq{
			UserAddress: walletAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func lendingBuildSupplyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			AssetAddress string `json:"assetAddress"`
			Amount       string `json:"amount"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.LendingRpc.BuildSupply(r.Context(), &lending.BuildSupplyReq{
			UserAddress:  walletAddress,
			AssetAddress: req.AssetAddress,
			Amount:       req.Amount,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func lendingBuildBorrowHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			AssetAddress string `json:"assetAddress"`
			Amount       string `json:"amount"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.LendingRpc.BuildBorrow(r.Context(), &lending.BuildBorrowReq{
			UserAddress:  walletAddress,
			AssetAddress: req.AssetAddress,
			Amount:       req.Amount,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func lendingBuildRepayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			AssetAddress string `json:"assetAddress"`
			Amount       string `json:"amount"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.LendingRpc.BuildRepay(r.Context(), &lending.BuildRepayReq{
			UserAddress:  walletAddress,
			AssetAddress: req.AssetAddress,
			Amount:       req.Amount,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func lendingBuildWithdrawHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			AssetAddress string `json:"assetAddress"`
			Amount       string `json:"amount"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.LendingRpc.BuildWithdraw(r.Context(), &lending.BuildWithdrawReq{
			UserAddress:  walletAddress,
			AssetAddress: req.AssetAddress,
			Amount:       req.Amount,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}
