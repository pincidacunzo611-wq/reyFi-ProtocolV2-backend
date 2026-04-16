package handler

import (
	"net/http"
	"strconv"

	"github.com/reyfi/reyfi-backend/app/dex/rpc/dex"
	"github.com/reyfi/reyfi-backend/app/gateway/internal/svc"
	"github.com/reyfi/reyfi-backend/pkg/middleware"
	"github.com/reyfi/reyfi-backend/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ==================== Health ====================

func healthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Success(r.Context(), w, map[string]string{
			"status":  "healthy",
			"service": "reyfi-gateway",
		})
	}
}

// ==================== DEX Public Handlers ====================

func dexGetPairsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		resp, err := svcCtx.DexRpc.GetPairs(r.Context(), &dex.GetPairsReq{
			Page:      page,
			PageSize:  pageSize,
			Keyword:   r.URL.Query().Get("keyword"),
			SortBy:    r.URL.Query().Get("sortBy"),
			SortOrder: r.URL.Query().Get("sortOrder"),
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.SuccessWithPage(r.Context(), w, resp.List, page, pageSize, resp.Total)
	}
}

func dexGetPairDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PairAddress string `path:"pairAddress"`
		}
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.DexRpc.GetPairDetail(r.Context(), &dex.GetPairDetailReq{
			PairAddress: req.PairAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func dexGetTradesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pathReq struct {
			PairAddress string `path:"pairAddress"`
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

		resp, err := svcCtx.DexRpc.GetTrades(r.Context(), &dex.GetTradesReq{
			PairAddress: pathReq.PairAddress,
			Page:        page,
			PageSize:    pageSize,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.SuccessWithPage(r.Context(), w, resp.List, page, pageSize, resp.Total)
	}
}

func dexGetCandlesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pathReq struct {
			PairAddress string `path:"pairAddress"`
		}
		if err := httpx.Parse(r, &pathReq); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		from, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
		to, _ := strconv.ParseInt(r.URL.Query().Get("to"), 10, 64)
		interval := r.URL.Query().Get("interval")
		if interval == "" {
			interval = "1h"
		}

		resp, err := svcCtx.DexRpc.GetCandles(r.Context(), &dex.GetCandlesReq{
			PairAddress: pathReq.PairAddress,
			Interval:    interval,
			From:        from,
			To:          to,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func dexGetOverviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := svcCtx.DexRpc.GetOverview(r.Context(), &dex.GetOverviewReq{})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func dexFindRouteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxHops, _ := strconv.ParseInt(r.URL.Query().Get("maxHops"), 10, 32)
		maxResults, _ := strconv.ParseInt(r.URL.Query().Get("maxResults"), 10, 32)
		if maxHops <= 0 {
			maxHops = 3
		}
		if maxResults <= 0 {
			maxResults = 3
		}

		resp, err := svcCtx.DexRpc.FindBestRoute(r.Context(), &dex.FindBestRouteReq{
			TokenIn:    r.URL.Query().Get("tokenIn"),
			TokenOut:   r.URL.Query().Get("tokenOut"),
			AmountIn:   r.URL.Query().Get("amountIn"),
			MaxHops:    int32(maxHops),
			MaxResults: int32(maxResults),
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

// ==================== DEX Auth Handlers ====================

func dexGetPositionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		resp, err := svcCtx.DexRpc.GetPositions(r.Context(), &dex.GetPositionsReq{
			UserAddress: walletAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func dexBuildSwapHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			TokenIn     string `json:"tokenIn"`
			TokenOut    string `json:"tokenOut"`
			AmountIn    string `json:"amountIn"`
			SlippageBps int    `json:"slippageBps"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}
		if req.SlippageBps <= 0 {
			req.SlippageBps = 50
		}

		resp, err := svcCtx.DexRpc.BuildSwap(r.Context(), &dex.BuildSwapReq{
			UserAddress: walletAddress,
			TokenIn:     req.TokenIn,
			TokenOut:    req.TokenOut,
			AmountIn:    req.AmountIn,
			SlippageBps: int32(req.SlippageBps),
			Receiver:    walletAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}
