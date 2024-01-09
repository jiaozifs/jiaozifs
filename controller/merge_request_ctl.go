package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jiaozifs/jiaozifs/utils/hash"

	"github.com/jiaozifs/jiaozifs/utils"

	"github.com/jiaozifs/jiaozifs/versionmgr"

	"github.com/jiaozifs/jiaozifs/api"
	"github.com/jiaozifs/jiaozifs/auth"
	"github.com/jiaozifs/jiaozifs/block/params"
	"github.com/jiaozifs/jiaozifs/models"
	"go.uber.org/fx"
)

type MergeRequestController struct {
	fx.In

	Repo                models.IRepo
	PublicStorageConfig params.AdapterConfig
}

func (mrCtl MergeRequestController) ListMergeRequests(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, params api.ListMergeRequestsParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := mrCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	repository, err := mrCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetOwnerID(owner.ID).SetName(repositoryName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != owner.Name {
		w.Forbidden()
		return
	}

	listParams := models.NewListMergeRequestParams().SetTargetRepoID(repository.ID)
	if params.State != nil {
		listParams.SetMergeState(models.MergeState(*params.State))
	}

	if params.After != nil {
		listParams.SetAfter(*params.After)
	}
	pageAmount := utils.IntValue(params.Amount)
	if pageAmount > utils.DefaultMaxPerPage || pageAmount <= 0 {
		listParams.SetAmount(utils.DefaultMaxPerPage)
	} else {
		listParams.SetAmount(pageAmount)
	}

	mrs, hasMore, err := mrCtl.Repo.MergeRequestRepo().List(ctx, listParams)
	if err != nil {
		w.Error(err)
		return
	}

	results := make([]api.MergeRequest, len(mrs))
	for index, mr := range mrs {
		results[index] = api.MergeRequest{
			Title:        mr.Title,
			Description:  mr.Description,
			AuthorId:     mr.AuthorID,
			MergeStatus:  int(mr.MergeState),
			SourceBranch: mr.SourceBranchID,
			SourceRepoId: mr.SourceRepoID,
			TargetBranch: mr.TargetBranchID,
			TargetRepoId: mr.TargetRepoID,
			CreatedAt:    mr.CreatedAt,
			UpdatedAt:    mr.UpdatedAt,
		}
	}
	pagMag := utils.PaginationFor(hasMore, results, "UpdatedAt")
	pagination := api.Pagination{
		HasMore:    pagMag.HasMore,
		MaxPerPage: pagMag.MaxPerPage,
		NextOffset: pagMag.NextOffset,
		Results:    pagMag.Results,
	}
	w.JSON(api.MergeRequestList{
		Pagination: pagination,
		Results:    results,
	})
}

func (mrCtl MergeRequestController) CreateMergeRequest(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, body api.CreateMergeRequestJSONRequestBody, ownerName string, repositoryName string) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := mrCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != owner.Name {
		w.Forbidden()
		return
	}

	// Get repo
	repository, err := mrCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetOwnerID(owner.ID).SetName(repositoryName))
	if err != nil {
		w.Error(err)
		return
	}

	if body.SourceBranchName == body.TargetBranchName {
		w.BadRequest(fmt.Sprintf("source branch name %s and target branch name %s can not be same", body.SourceBranchName, body.SourceBranchName))
		return
	}

	sourceBranch, err := mrCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetRepositoryID(repository.ID).SetName(body.SourceBranchName))
	if err != nil {
		w.Error(err)
		return
	}

	targetBranch, err := mrCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetRepositoryID(repository.ID).SetName(body.TargetBranchName))
	if err != nil {
		w.Error(err)
		return
	}

	params := models.NewGetMergeRequestParams().SetTargetRepo(repository.ID).SetTargetBranch(targetBranch.ID).SetSourceBranch(sourceBranch.ID).SetState(models.MergeStateInit)
	mr, err := mrCtl.Repo.MergeRequestRepo().Get(ctx, params)
	if err == nil {
		fmt.Println(mr)
		w.BadRequest(fmt.Sprintf("repo %s merge request between %s and %s already exists", repositoryName, body.SourceBranchName, body.TargetBranchName))
		return
	}

	if err != nil && !errors.Is(err, models.ErrNotFound) {
		w.Error(err)
		return
	}

	mrModel, err := mrCtl.Repo.MergeRequestRepo().Insert(ctx, &models.MergeRequest{
		TargetBranchID: targetBranch.ID,
		SourceBranchID: sourceBranch.ID,
		SourceRepoID:   repository.ID,
		TargetRepoID:   repository.ID,
		Title:          body.Title,
		MergeState:     models.MergeStateInit,
		Description:    body.Description,
		AuthorID:       operator.ID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	if err != nil {
		w.Error(err)
		return
	}

	workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, mrCtl.Repo, mrCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.CheckOut(ctx, versionmgr.InBranch, sourceBranch.Name)
	if err != nil {
		w.Error(err)
		return
	}

	changePairs, err := workRepo.GetMergeState(ctx, targetBranch.CommitHash)
	if err != nil {
		w.Error(err)
		return
	}

	resp := api.MergeRequestFullState{
		Id:           mrModel.ID,
		Sequence:     mrModel.Sequence,
		Title:        mrModel.Title,
		Description:  mrModel.Description,
		AuthorId:     mrModel.AuthorID,
		MergeStatus:  int(mrModel.MergeState),
		SourceBranch: mrModel.SourceBranchID,
		SourceRepoId: mrModel.SourceRepoID,
		TargetBranch: mrModel.TargetBranchID,
		TargetRepoId: mrModel.TargetRepoID,
		CreatedAt:    mrModel.CreatedAt,
		UpdatedAt:    mrModel.UpdatedAt,
	}

	resp.Changes, err = changePairToDTO(changePairs)
	if err != nil {
		w.Error(err)
		return
	}
	//get merge state
	w.JSON(mrModel, http.StatusCreated)
}
func (mrCtl MergeRequestController) GetMergeRequest(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, mrSeq uint64) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := mrCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != owner.Name {
		w.Forbidden()
		return
	}

	repository, err := mrCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetOwnerID(owner.ID).SetName(repositoryName))
	if err != nil {
		w.Error(err)
		return
	}

	workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, mrCtl.Repo, mrCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}

	mergeRequest, err := mrCtl.Repo.MergeRequestRepo().Get(ctx, models.NewGetMergeRequestParams().SetTargetRepo(repository.ID).SetNumber(mrSeq))
	if err != nil {
		w.Error(err)
		return
	}

	sourceBranch, err := mrCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetID(mergeRequest.SourceBranchID))
	if err != nil {
		w.Error(err)
		return
	}

	targetBranch, err := mrCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetID(mergeRequest.TargetBranchID))
	if err != nil {
		w.Error(err)
		return
	}

	err = workRepo.CheckOut(ctx, versionmgr.InBranch, sourceBranch.Name)
	if err != nil {
		w.Error(err)
		return
	}

	changePairs, err := workRepo.GetMergeState(ctx, targetBranch.CommitHash)
	if err != nil {
		w.Error(err)
		return
	}

	resp := api.MergeRequestFullState{
		Id:           mergeRequest.ID,
		Sequence:     mergeRequest.Sequence,
		Title:        mergeRequest.Title,
		Description:  mergeRequest.Description,
		AuthorId:     mergeRequest.AuthorID,
		MergeStatus:  int(mergeRequest.MergeState),
		SourceBranch: mergeRequest.SourceBranchID,
		SourceRepoId: mergeRequest.SourceRepoID,
		TargetBranch: mergeRequest.TargetBranchID,
		TargetRepoId: mergeRequest.TargetRepoID,
		CreatedAt:    mergeRequest.CreatedAt,
		UpdatedAt:    mergeRequest.UpdatedAt,
	}
	resp.Changes, err = changePairToDTO(changePairs)
	if err != nil {
		w.Error(err)
		return
	}
	w.JSON(resp)
}

func (mrCtl MergeRequestController) UpdateMergeRequest(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, body api.UpdateMergeRequestJSONRequestBody, ownerName string, repositoryName string, mrSeq uint64) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := mrCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != owner.Name {
		w.Forbidden()
		return
	}
	repository, err := mrCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetOwnerID(owner.ID).SetName(repositoryName))
	if err != nil {
		w.Error(err)
		return
	}

	updateParams := models.NewUpdateMergeRequestParams(repository.ID, mrSeq)
	if body.Title != nil {
		updateParams.SetTitle(utils.StringValue(body.Title))
	}
	if body.Description != nil {
		updateParams.SetDescription(utils.StringValue(body.Description))
	}
	if body.Status != nil {
		updateParams.SetState(models.MergeState(utils.IntValue(body.Status)))
	}

	err = mrCtl.Repo.MergeRequestRepo().UpdateByID(ctx, updateParams)
	if err != nil {
		w.Error(err)
		return
	}
	w.OK()
}

func (mrCtl MergeRequestController) Merge(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, body api.MergeJSONRequestBody, ownerName string, repositoryName string, mrSeq uint64) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := mrCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != owner.Name {
		w.Forbidden()
		return
	}

	// Get repo
	repository, err := mrCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetOwnerID(owner.ID).SetName(repositoryName))
	if err != nil {
		w.Error(err)
		return
	}

	mergeRequest, err := mrCtl.Repo.MergeRequestRepo().Get(ctx, models.NewGetMergeRequestParams().SetTargetRepo(repository.ID).SetNumber(mrSeq))
	if err != nil {
		w.Error(err)
		return
	}

	var commit *models.Commit
	err = mrCtl.Repo.Transaction(ctx, func(repo models.IRepo) error {
		workRepo, err := versionmgr.NewWorkRepositoryFromConfig(ctx, operator, repository, mrCtl.Repo, mrCtl.PublicStorageConfig)
		if err != nil {
			return err
		}

		sourceBranch, err := mrCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetID(mergeRequest.SourceBranchID))
		if err != nil {
			return err
		}

		targetBranch, err := mrCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetID(mergeRequest.TargetBranchID))
		if err != nil {
			return err
		}

		err = workRepo.CheckOut(ctx, versionmgr.InBranch, sourceBranch.Name)
		if err != nil {
			return err
		}

		commit, err = workRepo.Merge(ctx, targetBranch.CommitHash, body.Msg, versionmgr.ResolveFromSelector(utils.Map(body.ConflictResolve)))
		if err != nil {
			return err
		}

		return mrCtl.Repo.MergeRequestRepo().UpdateByID(ctx, models.NewUpdateMergeRequestParams(repository.ID, mergeRequest.Sequence).SetState(models.MergeStateMerged))
	})
	if err != nil {
		w.Error(err)
		return
	}

	w.JSON(commit)
}

func changePairToDTO(pairs []*versionmgr.ChangePair) ([]api.ChangePair, error) {

	var changes = make([]api.ChangePair, len(pairs))
	for index, ch := range pairs {
		var path string
		if ch.Left != nil {
			path = ch.Left.Path()
		} else {
			path = ch.Right.Path()
		}
		pair := api.ChangePair{
			Path:       path,
			IsConflict: ch.IsConflict,
		}

		if ch.Left != nil {
			leftAction, err := ch.Left.Action()
			if err != nil {
				return nil, err
			}
			pair.Left = &api.Change{
				Action: api.ChangeAction(leftAction),
				Path:   path,
			}
			if ch.Left.From() != nil {
				pair.Left.BaseHash = utils.String(hash.Hash(ch.Left.From().Hash()).Hex())
			}
			if ch.Left.To() != nil {
				pair.Left.ToHash = utils.String(hash.Hash(ch.Left.To().Hash()).Hex())
			}
		}

		if ch.Right != nil {
			rightAction, err := ch.Right.Action()
			if err != nil {
				return nil, err
			}
			pair.Right = &api.Change{
				Action: api.ChangeAction(rightAction),
				Path:   path,
			}
			if ch.Right.From() != nil {
				pair.Right.BaseHash = utils.String(hash.Hash(ch.Right.From().Hash()).Hex())
			}
			if ch.Right.To() != nil {
				pair.Right.ToHash = utils.String(hash.Hash(ch.Right.To().Hash()).Hex())
			}
		}
		changes[index] = pair
	}
	return changes, nil
}
