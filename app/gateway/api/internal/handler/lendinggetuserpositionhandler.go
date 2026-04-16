// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/logic"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func LendingGetUserPositionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewLendingGetUserPositionLogic(r.Context(), svcCtx)
		err := l.LendingGetUserPosition()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
