package apiimpl

import (
	"github.com/jiaozifs/jiaozifs/api"
	"github.com/jiaozifs/jiaozifs/controller"
	"go.uber.org/fx"
)

var _ api.ServerInterface = (*APIController)(nil)

type APIController struct {
	fx.In

	controller.CommonController
	controller.ObjectController
	controller.UserController
	controller.WipController
	controller.CommitController
	controller.RepositoryController
	controller.BranchController
	controller.MergeRequestController
}
