package controller

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/GitDataAI/jiaozifs/api"
	"github.com/GitDataAI/jiaozifs/auth"
	"github.com/GitDataAI/jiaozifs/auth/rbac"
	"github.com/GitDataAI/jiaozifs/block/params"
	"github.com/GitDataAI/jiaozifs/models"
	"github.com/GitDataAI/jiaozifs/models/rbacmodel"
	"github.com/GitDataAI/jiaozifs/utils"
	"github.com/GitDataAI/jiaozifs/utils/hash"
	"github.com/GitDataAI/jiaozifs/versionmgr"
	"go.uber.org/fx"
)

type WipController struct {
	fx.In
	BaseController

	Repo                models.IRepo
	PublicStorageConfig params.AdapterConfig
}

// GetWip get wip of specific repository, operator only get himself wip
func (wipCtl WipController) GetWip(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, params api.GetWipParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := wipCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := wipCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetOwnerID(owner.ID).SetName(repositoryName))
	if err != nil {
		w.Error(err)
		return
	}

	if !wipCtl.authorizeMember(ctx, w, repository.ID, rbac.Node{
		Type: rbac.NodeTypeAnd,
		Nodes: []rbac.Node{
			{
				Permission: rbac.Permission{
					Action:   rbacmodel.ReadWipAction,
					Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
				},
			},
			{
				Permission: rbac.Permission{
					Action:   rbacmodel.CreateWipAction,
					Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
				},
			},
		},
	}) {
		return
	}

	workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, wipCtl.Repo, wipCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.CheckOut(ctx, versionmgr.InBranch, params.RefName)
	if err != nil {
		w.Error(err)
		return
	}

	wip, isNew, err := workRepo.GetOrCreateWip(ctx)
	if err != nil {
		w.Error(err)
		return
	}
	if isNew {
		w.JSON(wipToDto(wip), http.StatusCreated)
		return
	}
	w.JSON(wipToDto(wip))
}

// ListWip return wips of branches, operator only see himself wips in specific repository
func (wipCtl WipController) ListWip(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := wipCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := wipCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if !wipCtl.authorizeMember(ctx, w, repository.ID, rbac.Node{
		Permission: rbac.Permission{
			Action:   rbacmodel.ListWipAction,
			Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
		},
	}) {
		return
	}

	wips, err := wipCtl.Repo.WipRepo().List(ctx, models.NewListWipParams().SetCreatorID(operator.ID).SetRepositoryID(repository.ID))
	if err != nil {
		w.Error(err)
		return
	}

	apiWips := make([]*api.Wip, len(wips))
	for index, wip := range wips {
		apiWips[index] = wipToDto(wip)
	}
	w.JSON(apiWips)
}

// CommitWip commit wip to branch, operator only could operator himself wip
func (wipCtl WipController) CommitWip(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName, repositoryName string, params api.CommitWipParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := wipCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := wipCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if !wipCtl.authorizeMember(ctx, w, repository.ID, rbac.Node{
		Type: rbac.NodeTypeAnd,
		Nodes: []rbac.Node{
			{
				Permission: rbac.Permission{
					Action:   rbacmodel.ReadWipAction,
					Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
				},
			},
			{
				Permission: rbac.Permission{
					Action:   rbacmodel.WriteBranchAction,
					Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
				},
			},
			{
				Permission: rbac.Permission{
					Action:   rbacmodel.ReadCommitAction,
					Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
				},
			},
		},
	}) {
		return
	}

	workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, wipCtl.Repo, wipCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.CheckOut(ctx, versionmgr.InWip, params.RefName)
	if err != nil {
		w.Error(err)
		return
	}

	_, err = workRepo.CommitChanges(ctx, params.Msg)
	if err != nil {
		w.Error(err)
		return
	}

	w.JSON(wipToDto(workRepo.CurWip()), http.StatusCreated)
}

func (wipCtl WipController) UpdateWip(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, body api.UpdateWipJSONRequestBody, ownerName string, repositoryName string, params api.UpdateWipParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := wipCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := wipCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if !wipCtl.authorizeMember(ctx, w, repository.ID, rbac.Node{
		Permission: rbac.Permission{
			Action:   rbacmodel.WriteWipAction,
			Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
		},
	}) {
		return
	}

	ref, err := wipCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetRepositoryID(repository.ID).SetName(params.RefName))
	if err != nil {
		w.Error(err)
		return
	}

	wip, err := wipCtl.Repo.WipRepo().Get(ctx, models.NewGetWipParams().SetCreatorID(operator.ID).SetRepositoryID(repository.ID).SetRefID(ref.ID))
	if err != nil {
		w.Error(err)
		return
	}

	updateParams := models.NewUpdateWipParams(wip.ID)
	if body.BaseCommit != nil {
		baseCommitHash, err := hash.FromHex(utils.StringValue(body.BaseCommit))
		if err != nil {
			w.Error(err)
			return
		}

		if !baseCommitHash.IsEmpty() {
			_, err = wipCtl.Repo.CommitRepo(repository.ID).Commit(ctx, baseCommitHash)
			if err != nil {
				w.Error(fmt.Errorf("unable to get commit hash %s: %w", baseCommitHash, err))
				return
			}
		}
		updateParams.SetBaseCommit(baseCommitHash)
	}
	if body.CurrentTree != nil {
		currentTreeHash, err := hash.FromHex(utils.StringValue(body.CurrentTree))
		if err != nil {
			w.Error(err)
			return
		}

		if !currentTreeHash.IsEmpty() {
			_, err = wipCtl.Repo.FileTreeRepo(repository.ID).TreeNode(ctx, currentTreeHash)
			if err != nil {
				w.Error(fmt.Errorf("unable to get tree root %s: %w", currentTreeHash, err))
				return
			}
		}
		updateParams.SetCurrentTree(currentTreeHash)
	}

	err = wipCtl.Repo.WipRepo().UpdateByID(ctx, updateParams)
	if err != nil {
		w.Error(err)
		return
	}
	w.OK()
}

// DeleteWip delete active working in process operator only can delete himself wip
func (wipCtl WipController) DeleteWip(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, params api.DeleteWipParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := wipCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := wipCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if !wipCtl.authorizeMember(ctx, w, repository.ID, rbac.Node{
		Permission: rbac.Permission{
			Action:   rbacmodel.DeleteBranchAction,
			Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
		},
	}) {
		return
	}

	workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, wipCtl.Repo, wipCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.CheckOut(ctx, versionmgr.InBranch, params.RefName)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.DeleteWip(ctx)
	if err != nil {
		w.Error(err)
		return
	}
	w.OK()
}

// GetWipChanges return wip difference, operator only see himself wip
func (wipCtl WipController) GetWipChanges(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName, repositoryName string, params api.GetWipChangesParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := wipCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := wipCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if !wipCtl.authorizeMember(ctx, w, repository.ID, rbac.Node{
		Type: rbac.NodeTypeAnd,
		Nodes: []rbac.Node{
			{
				Permission: rbac.Permission{
					Action:   rbacmodel.ReadCommitAction,
					Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
				},
			},
			{
				Permission: rbac.Permission{
					Action:   rbacmodel.ReadWipAction,
					Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
				},
			},
			{
				Permission: rbac.Permission{
					Action:   rbacmodel.ReadBranchAction,
					Resource: rbacmodel.RepoURArn(owner.ID.String(), repository.ID.String()),
				},
			},
		},
	}) {
		return
	}

	ref, err := wipCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetRepositoryID(repository.ID).SetName(params.RefName))
	if err != nil {
		w.Error(err)
		return
	}

	wip, err := wipCtl.Repo.WipRepo().Get(ctx, models.NewGetWipParams().SetCreatorID(operator.ID).SetRepositoryID(repository.ID).SetRefID(ref.ID))
	if err != nil {
		w.Error(err)
		return
	}

	treeHash := hash.Empty
	if !wip.BaseCommit.IsEmpty() {
		commit, err := wipCtl.Repo.CommitRepo(repository.ID).Commit(ctx, wip.BaseCommit)
		if err != nil {
			w.Error(err)
			return
		}
		treeHash = commit.TreeHash
	}

	if bytes.Equal(treeHash, wip.CurrentTree) {
		w.JSON([]api.Change{}) //no change return nothing
		return
	}

	workTree, err := versionmgr.NewWorkTree(ctx, wipCtl.Repo.FileTreeRepo(repository.ID), models.NewRootTreeEntry(treeHash))
	if err != nil {
		w.Error(err)
		return
	}

	changes, err := workTree.Diff(ctx, wip.CurrentTree, utils.StringValue(params.Path))
	if err != nil {
		w.Error(err)
		return
	}

	changesResp, err := changesToDTO(changes)
	if err != nil {
		w.Error(err)
		return
	}
	w.JSON(changesResp)
}

// RevertWipChanges revert wip changes, if path is empty, revert all
func (wipCtl WipController) RevertWipChanges(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, params api.RevertWipChangesParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := wipCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := wipCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != owner.Name { //todo check permission to operator ownerRepo
		w.Forbidden()
		return
	}

	workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, wipCtl.Repo, wipCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.CheckOut(ctx, versionmgr.InWip, params.RefName)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.Revert(ctx, utils.StringValue(params.PathPrefix))
	if err != nil {
		w.Error(err)
		return
	}

	w.OK()
}

func wipToDto(wip *models.WorkingInProcess) *api.Wip {
	return &api.Wip{
		BaseCommit:   wip.BaseCommit.Hex(),
		CreatedAt:    wip.CreatedAt.UnixMilli(),
		CreatorId:    wip.CreatorID,
		CurrentTree:  wip.CurrentTree.Hex(),
		Id:           wip.ID,
		RefId:        wip.RefID,
		RepositoryId: wip.RepositoryID,
		State:        int(wip.State),
		UpdatedAt:    wip.UpdatedAt.UnixMilli(),
	}
}
