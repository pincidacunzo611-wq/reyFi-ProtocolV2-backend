// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/logic"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func VaultGetListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.VaultListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewVaultGetListLogic(r.Context(), svcCtx)
		err := l.VaultGetList(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
