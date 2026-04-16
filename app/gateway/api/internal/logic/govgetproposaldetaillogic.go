// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/svc"
	"github.com/reyfi/reyfi-backend/app/gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GovGetProposalDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGovGetProposalDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GovGetProposalDetailLogic {
	return &GovGetProposalDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GovGetProposalDetailLogic) GovGetProposalDetail(req *types.GovProposalDetailReq) error {
	// todo: add your logic here and delete this line

	return nil
}
