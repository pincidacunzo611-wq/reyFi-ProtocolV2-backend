package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/reyfi/reyfi-backend/app/gateway/internal/svc"
	"github.com/reyfi/reyfi-backend/app/user/rpc/user"
	"github.com/reyfi/reyfi-backend/pkg/middleware"
	"github.com/reyfi/reyfi-backend/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ==================== User Auth Handlers ====================

func userGetNonceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Address string `json:"address"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.UserRpc.GetNonce(r.Context(), &user.GetNonceReq{
			Address: req.Address,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func userLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Address   string `json:"address"`
			Message   string `json:"message"`
			Signature string `json:"signature"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.UserRpc.Login(r.Context(), &user.LoginReq{
			Address:   req.Address,
			Message:   req.Message,
			Signature: req.Signature,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func userRefreshTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.UserRpc.RefreshToken(r.Context(), &user.RefreshTokenReq{
			RefreshToken: req.RefreshToken,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

// ==================== User Data Handlers ====================

func userGetPortfolioHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		resp, err := svcCtx.UserRpc.GetPortfolio(r.Context(), &user.GetPortfolioReq{
			WalletAddress: walletAddress,
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

func userGetActivityHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())
		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		resp, err := svcCtx.UserRpc.GetActivity(r.Context(), &user.GetActivityReq{
			WalletAddress: walletAddress,
			Module:        r.URL.Query().Get("module"),
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

func userUpdateSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := middleware.GetWalletAddress(r.Context())

		var req struct {
			Nickname        string `json:"nickname"`
			Language        string `json:"language"`
			Currency        string `json:"currency"`
			SlippageDefault int    `json:"slippageDefault"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.Error(r.Context(), w, 400, err.Error())
			return
		}

		resp, err := svcCtx.UserRpc.UpdateSettings(r.Context(), &user.UpdateSettingsReq{
			WalletAddress:   walletAddress,
			Nickname:        req.Nickname,
			Language:        req.Language,
			Currency:        req.Currency,
			SlippageDefault: int32(req.SlippageDefault),
		})
		if err != nil {
			response.Error(r.Context(), w, 500, err.Error())
			return
		}

		response.Success(r.Context(), w, resp)
	}
}

// ==================== System Handlers ====================

func systemSyncStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 Redis 读取各模块同步高度
		modules := []string{"dex", "lending", "futures", "options", "vault", "bonds", "governance"}
		moduleStatus := make([]map[string]interface{}, 0, len(modules))

		for _, mod := range modules {
			heightStr, _ := svcCtx.Redis.Get("sync:" + mod + ":height")
			height := int64(0)
			if heightStr != "" {
				fmt.Sscanf(heightStr, "%d", &height)
			}
			lastSync, _ := svcCtx.Redis.Get("sync:" + mod + ":lastSync")
			moduleStatus = append(moduleStatus, map[string]interface{}{
				"module":   mod,
				"height":   height,
				"lastSync": lastSync,
				"status":   "synced",
			})
		}

		// 链上最新高度
		chainHeight := int64(0)
		if h, err := svcCtx.Redis.Get("sync:chainHeight"); err == nil && h != "" {
			fmt.Sscanf(h, "%d", &chainHeight)
		}

		response.Success(r.Context(), w, map[string]interface{}{
			"chainHeight": chainHeight,
			"modules":     moduleStatus,
		})
	}
}
