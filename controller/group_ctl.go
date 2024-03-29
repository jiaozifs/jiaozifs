package controller

import (
	"context"
	"net/http"

	"github.com/GitDataAI/jiaozifs/models/rbacmodel"

	"github.com/GitDataAI/jiaozifs/utils"

	"github.com/GitDataAI/jiaozifs/auth"

	"github.com/GitDataAI/jiaozifs/auth/rbac"

	"github.com/GitDataAI/jiaozifs/api"
	"github.com/GitDataAI/jiaozifs/models"
	"go.uber.org/fx"
)

type GroupController struct {
	fx.In
	BaseController

	Repo models.IRepo
}

func (gCtl GroupController) ListRepoGroup(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request) {
	_, err := auth.GetOperator(ctx)
	if err != nil {
		w.Forbidden()
		return
	}

	groups, err := gCtl.Repo.GroupRepo().List(ctx, rbacmodel.NewListGroupParams().SetNames(rbac.RepoAdmin, rbac.RepoWrite, rbac.RepoRead))
	if err != nil {
		w.Error(err)
		return
	}
	w.JSON(utils.Silent(utils.ArrMap[*rbacmodel.Group, *api.Group](groups, groupToDto)))
}

func groupToDto(group *rbacmodel.Group) (*api.Group, error) {
	return &api.Group{
		Id:        group.ID,
		Name:      group.Name,
		Policies:  group.Policies,
		CreatedAt: group.CreatedAt.UnixMilli(),
		UpdatedAt: group.UpdatedAt.UnixMilli(),
	}, nil
}
