// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BondsBuildClaimLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBondsBuildClaimLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BondsBuildClaimLogic {
	return &BondsBuildClaimLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BondsBuildClaimLogic) BondsBuildClaim(req *types.BondsBuildClaimReq) error {
	// todo: add your logic here and delete this line

	return nil
}
