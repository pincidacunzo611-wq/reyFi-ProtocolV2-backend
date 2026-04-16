// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DexGetPairsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDexGetPairsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DexGetPairsLogic {
	return &DexGetPairsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DexGetPairsLogic) DexGetPairs(req *types.DexPairsReq) error {
	// todo: add your logic here and delete this line

	return nil
}
