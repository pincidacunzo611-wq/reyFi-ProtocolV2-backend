package handler

import (
	"net/http"
	"strconv"

	"github.com/reyfi/reyfi-backend/app/futures/rpc/futures"
	"github.com/reyfi/reyfi-backend/app/gateway/internal/svc"
	"github.com/reyfi/reyfi-backend/pkg/middleware"
	"github.com/reyfi/reyfi-backend/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ==================== Futures Public Handlers ====================

func futuresGetMarketsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		resp, err := svcCtx.FuturesRpc.GetMarkets(r.Context(), &futures.GetMarketsReq{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.SuccessWithPage(r.Context(), w, resp.List, page, pageSize, resp.Total)
	}
}

func futuresGetMarketDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			MarketAddress string `path:"marketAddress"`
		}
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.FuturesRpc.GetMarketDetail(r.Context(), &futures.GetMarketDetailReq{
			MarketAddress: req.MarketAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func futuresGetFundingHistoryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pathReq struct {
			MarketAddress string `path:"marketAddress"`
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

		resp, err := svcCtx.FuturesRpc.GetFundingHistory(r.Context(), &futures.GetFundingHistoryReq{
			MarketAddress: pathReq.MarketAddress,
			Page:          page,
			PageSize:      pageSize,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.SuccessWithPage(r.Context(), w, resp.List, page, pageSize, resp.Total)
	}
}

// ==================== Futures Auth Handlers ====================

func futuresGetPositionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		resp, err := svcCtx.FuturesRpc.GetPositions(r.Context(), &futures.GetPositionsReq{
			UserAddress: walletAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func futuresBuildOpenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			MarketAddress string `json:"marketAddress"`
			Side          string `json:"side"`
			Margin        string `json:"margin"`
			Leverage      string `json:"leverage"`
			LimitPrice    string `json:"limitPrice"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.FuturesRpc.BuildOpenPosition(r.Context(), &futures.BuildOpenPositionReq{
			UserAddress:   walletAddress,
			MarketAddress: req.MarketAddress,
			Side:          req.Side,
			Margin:        req.Margin,
			Leverage:      req.Leverage,
			LimitPrice:    req.LimitPrice,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func futuresBuildCloseHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			PositionId string `json:"positionId"`
			Amount     string `json:"amount"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.FuturesRpc.BuildClosePosition(r.Context(), &futures.BuildClosePositionReq{
			UserAddress: walletAddress,
			PositionId:  req.PositionId,
			Amount:      req.Amount,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func futuresBuildAdjustMarginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			PositionId string `json:"positionId"`
			Amount     string `json:"amount"`
			IsAdd      bool   `json:"isAdd"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.FuturesRpc.BuildAdjustMargin(r.Context(), &futures.BuildAdjustMarginReq{
			UserAddress: walletAddress,
			PositionId:  req.PositionId,
			Amount:      req.Amount,
			IsAdd:       req.IsAdd,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}
