package controller

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/jiaozifs/jiaozifs/api"
	"github.com/jiaozifs/jiaozifs/auth"
	"github.com/jiaozifs/jiaozifs/block/params"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/utils"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/jiaozifs/jiaozifs/versionmgr"
	"go.uber.org/fx"
)

type CommitController struct {
	fx.In

	Repo                models.IRepo
	PublicStorageConfig params.AdapterConfig
}

func (commitCtl CommitController) GetEntriesInRef(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, params api.GetEntriesInRefParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := commitCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := commitCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != ownerName { //todo check permission
		w.Forbidden()
		return
	}

	treeHash := hash.Empty
	if params.Type == api.RefTypeWip {
		refName := repository.HEAD
		if params.Ref != nil {
			refName = *params.Ref
		}

		//todo maybe from tag reference
		ref, err := commitCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetRepositoryID(repository.ID).SetName(refName))
		if err != nil {
			w.Error(err)
			return
		}
		wip, err := commitCtl.Repo.WipRepo().Get(ctx, models.NewGetWipParams().SetCreatorID(operator.ID).SetRepositoryID(repository.ID).SetRefID(ref.ID))
		if err != nil {
			w.Error(err)
			return
		}
		treeHash = wip.CurrentTree
	} else if params.Type == api.RefTypeBranch {
		refName := repository.HEAD
		if params.Ref != nil {
			refName = *params.Ref
		}

		ref, err := commitCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetRepositoryID(repository.ID).SetName(refName))
		if err != nil {
			w.Error(err)
			return
		}
		if !ref.CommitHash.IsEmpty() {
			commit, err := commitCtl.Repo.CommitRepo(repository.ID).Commit(ctx, ref.CommitHash)
			if err != nil {
				w.Error(err)
				return
			}
			treeHash = commit.TreeHash
		}
	} else if params.Type == api.RefTypeCommit {
		commitHash, err := hash.FromHex(utils.StringValue(params.Ref))
		if err != nil {
			w.BadRequest(err.Error())
			return
		}

		if !commitHash.IsEmpty() {
			commit, err := commitCtl.Repo.CommitRepo(repository.ID).Commit(ctx, commitHash)
			if err != nil {
				w.Error(err)
				return
			}
			treeHash = commit.TreeHash
		}
	} else {
		//check in validate middleware, test cant cover here, keep this check
		w.BadRequest("not support")
		return
	}

	workTree, err := versionmgr.NewWorkTree(ctx, commitCtl.Repo.FileTreeRepo(repository.ID), models.NewRootTreeEntry(treeHash))
	if err != nil {
		w.Error(err)
		return
	}

	path := versionmgr.CleanPath(utils.StringValue(params.Path))
	treeEntry, err := workTree.Ls(ctx, path)
	if err != nil {
		if errors.Is(err, versionmgr.ErrPathNotFound) {
			w.NotFound()
			return
		}
		w.Error(err)
		return
	}
	w.JSON(treeEntry)
}

func (commitCtl CommitController) CompareCommit(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, basehead string, params api.CompareCommitParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := commitCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := commitCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.ID != owner.ID { //todo check permission
		w.Forbidden()
		return
	}

	baseHead := strings.Split(basehead, "...")
	if len(baseHead) != 2 {
		w.BadRequest("invalid basehead must be base...head")
		return
	}

	toCommitHash, err := hex.DecodeString(baseHead[1])
	if err != nil {
		w.Error(err)
		return
	}

	workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, commitCtl.Repo, commitCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.CheckOut(ctx, versionmgr.InCommit, baseHead[0])
	if err != nil {
		w.Error(err)
		return
	}

	changes, err := workRepo.DiffCommit(ctx, toCommitHash, utils.StringValue(params.Path))
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

func (commitCtl CommitController) GetCommitChanges(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, commitID string, params api.GetCommitChangesParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := commitCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := commitCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.ID != owner.ID { //todo check permission
		w.Forbidden()
		return
	}

	workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, commitCtl.Repo, commitCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.CheckOut(ctx, versionmgr.InCommit, commitID)
	if err != nil {
		w.Error(err)
		return
	}

	changes, err := workRepo.GetCommitChanges(ctx, utils.StringValue(params.Path))
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
