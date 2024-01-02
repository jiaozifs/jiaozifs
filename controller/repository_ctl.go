package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jiaozifs/jiaozifs/block"
	"github.com/jiaozifs/jiaozifs/block/factory"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/jiaozifs/jiaozifs/api"
	"github.com/jiaozifs/jiaozifs/auth"
	"github.com/jiaozifs/jiaozifs/block/params"
	"github.com/jiaozifs/jiaozifs/config"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/utils"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/jiaozifs/jiaozifs/versionmgr"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/fx"
)

const DefaultBranchName = "main"

var repoLog = logging.Logger("repo control")
var maxNameLength = 20
var alphanumeric = regexp.MustCompile("^[a-zA-Z0-9_]*$")

// RepoNameBlackList forbid repo name, reserve for routes
var RepoNameBlackList = []string{"repository", "repositories", "wip", "wips", "object", "objects", "commit", "commits", "ref", "refs", "repo", "repos", "user", "users"}

func CheckRepositoryName(name string) error {
	for _, blackName := range RepoNameBlackList {
		if name == blackName {
			return errors.New("repository name is black list")
		}
	}

	if !alphanumeric.MatchString(name) {
		return errors.New("repository name must be combination of number and letter")
	}
	if len(name) > maxNameLength {
		return errors.New("repository name is too long")
	}
	return nil
}

type RepositoryController struct {
	fx.In

	Repo                models.IRepo
	PublicStorageConfig params.AdapterConfig
}

func (repositoryCtl RepositoryController) ListRepositoryOfAuthenticatedUser(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, params api.ListRepositoryOfAuthenticatedUserParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	listRepoParams := models.NewListRepoParams()
	if params.Prefix != nil && len(*params.Prefix) > 0 {
		listRepoParams.SetName(*params.Prefix, models.PrefixMatch)
	}
	if params.After != nil {
		listRepoParams.SetAfter(*params.After)
	}
	pageAmount := utils.IntValue(params.Amount)
	if pageAmount > utils.DefaultMaxPerPage || pageAmount <= 0 {
		listRepoParams.SetAmount(utils.DefaultMaxPerPage)
	} else {
		listRepoParams.SetAmount(pageAmount)
	}

	repositories, hasMore, err := repositoryCtl.Repo.RepositoryRepo().List(ctx, listRepoParams.
		SetOwnerID(operator.ID))
	if err != nil {
		w.Error(err)
		return
	}
	results := make([]api.Repository, 0, len(repositories))
	for _, repo := range repositories {
		r := api.Repository{
			CreatedAt:   repo.CreatedAt,
			CreatorId:   repo.CreatorID,
			Description: repo.Description,
			Head:        repo.HEAD,
			Id:          repo.ID,
			Name:        repo.Name,
			UpdatedAt:   repo.UpdatedAt,
		}
		results = append(results, r)
	}
	pagMag := utils.PaginationFor(hasMore, results, "UpdatedAt")
	pagination := api.Pagination{
		HasMore:    pagMag.HasMore,
		MaxPerPage: pagMag.MaxPerPage,
		NextOffset: pagMag.NextOffset,
		Results:    pagMag.Results,
	}
	w.JSON(api.RepositoryList{
		Pagination: pagination,
		Results:    results,
	})
}

func (repositoryCtl RepositoryController) ListRepository(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, params api.ListRepositoryParams) {
	owner, err := repositoryCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}
	if owner.ID != operator.ID { //todo check public or private and allow  access public repos
		w.Forbidden()
		return
	}

	listRepoParams := models.NewListRepoParams().SetOwnerID(owner.ID)
	if params.Prefix != nil && len(*params.Prefix) > 0 {
		listRepoParams.SetName(*params.Prefix, models.PrefixMatch)
	}
	if params.After != nil {
		listRepoParams.SetAfter(*params.After)
	}
	pageAmount := utils.IntValue(params.Amount)
	if pageAmount > utils.DefaultMaxPerPage || pageAmount <= 0 {
		listRepoParams.SetAmount(utils.DefaultMaxPerPage)
	} else {
		listRepoParams.SetAmount(pageAmount)
	}

	repositories, hasMore, err := repositoryCtl.Repo.RepositoryRepo().List(ctx, listRepoParams)
	if err != nil {
		w.Error(err)
		return
	}
	results := make([]api.Repository, 0, len(repositories))
	for _, repo := range repositories {
		r := api.Repository{
			CreatedAt:   repo.CreatedAt,
			CreatorId:   repo.CreatorID,
			Description: repo.Description,
			Head:        repo.HEAD,
			Id:          repo.ID,
			Name:        repo.Name,
			UpdatedAt:   repo.UpdatedAt,
		}
		results = append(results, r)
	}
	pagMag := utils.PaginationFor(hasMore, results, "UpdatedAt")
	pagination := api.Pagination{
		HasMore:    pagMag.HasMore,
		MaxPerPage: pagMag.MaxPerPage,
		NextOffset: pagMag.NextOffset,
		Results:    pagMag.Results,
	}
	w.JSON(api.RepositoryList{
		Pagination: pagination,
		Results:    results,
	})
}

func (repositoryCtl RepositoryController) CreateRepository(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, body api.CreateRepositoryJSONRequestBody) {
	err := CheckRepositoryName(body.Name)
	if err != nil {
		w.BadRequest(err.Error())
		return
	}

	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	var usePublicStorage = true
	storageConfig := utils.StringValue(body.BlockstoreConfig)
	repoID := uuid.New()
	var storageNamespace *string
	if len(storageConfig) > 0 {
		usePublicStorage = false
		var cfg = config.BlockStoreConfig{}
		err = json.Unmarshal([]byte(storageConfig), &cfg)
		if err != nil {
			w.BadRequest("storage config not json format")
			return
		}

		if cfg.BlockstoreType() == "local" {
			repoLog.Infof("custom storage cnofig can not be local")
			w.Forbidden()
			return
		}
		storageNamespace = cfg.DefaultNamespacePrefix
	} else {
		storageNamespace = utils.String(fmt.Sprintf("%s://%s", repositoryCtl.PublicStorageConfig.BlockstoreType(), repoID.String()))
	}

	defaultRef := &models.Branch{
		RepositoryID: repoID,
		CommitHash:   hash.Hash{},
		Name:         DefaultBranchName,
		CreatorID:    operator.ID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repository := &models.Repository{
		ID:                   repoID,
		Name:                 body.Name,
		UsePublicStorage:     usePublicStorage,
		StorageAdapterParams: &storageConfig,
		StorageNamespace:     storageNamespace,
		Description:          body.Description,
		HEAD:                 DefaultBranchName,
		OwnerID:              operator.ID, // this api only create repo for operator
		CreatorID:            operator.ID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	//create default ref
	var createdRepo *models.Repository
	err = repositoryCtl.Repo.Transaction(ctx, func(repo models.IRepo) error {
		_, err := repositoryCtl.Repo.BranchRepo().Insert(ctx, defaultRef)
		if err != nil {
			return err
		}
		createdRepo, err = repositoryCtl.Repo.RepositoryRepo().Insert(ctx, repository)
		return err
	})
	if err != nil {
		w.Error(err)
		return
	}

	w.JSON(api.Repository{
		CreatedAt:   createdRepo.CreatedAt,
		CreatorId:   createdRepo.CreatorID,
		Description: createdRepo.Description,
		Head:        createdRepo.HEAD,
		Id:          createdRepo.ID,
		Name:        createdRepo.Name,
		UpdatedAt:   createdRepo.UpdatedAt,
	})
}

func (repositoryCtl RepositoryController) DeleteRepository(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := repositoryCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != owner.Name {
		w.Forbidden()
		return
	}

	repo, err := repositoryCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	affectRows, err := repositoryCtl.Repo.RepositoryRepo().Delete(ctx, models.NewDeleteRepoParams().SetID(repo.ID))
	if err != nil {
		w.Error(err)
		return
	}

	if affectRows == 0 {
		w.NotFound()
		return
	}

	var adapter block.Adapter
	if !repo.UsePublicStorage {
		w.String("The repo doesn't use public storage and cannot be cleared.", 200)
		return
	}
	adapter, err = factory.BuildBlockAdapter(ctx, repositoryCtl.PublicStorageConfig)
	if err != nil {
		w.Error(err)
		return
	}
	err = adapter.Clean(ctx, config.DefaultLocalBSPath, *repo.StorageNamespace)
	if err != nil {
		w.Error(err)
		return
	}
	w.OK()
}

func (repositoryCtl RepositoryController) GetRepository(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := repositoryCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != owner.Name { //todo check public or private / and permission
		w.Forbidden()
		return
	}

	repo, err := repositoryCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	w.JSON(repo)
}

func (repositoryCtl RepositoryController) UpdateRepository(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, body api.UpdateRepositoryJSONRequestBody, ownerName string, repositoryName string) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := repositoryCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != ownerName { //todo check permission to modify owner repo
		w.Forbidden()
		return
	}

	repo, err := repositoryCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetName(repositoryName).SetOwnerID(owner.ID))
	if err != nil {
		w.Error(err)
		return
	}

	err = repositoryCtl.Repo.RepositoryRepo().UpdateByID(ctx, models.NewUpdateRepoParams(repo.ID).SetDescription(utils.StringValue(body.Description)))
	if err != nil {
		w.Error(err)
		return
	}
	w.OK()
}

func (repositoryCtl RepositoryController) GetCommitsInRepository(ctx context.Context, w *api.JiaozifsResponse, _ *http.Request, ownerName string, repositoryName string, params api.GetCommitsInRepositoryParams) {
	operator, err := auth.GetOperator(ctx)
	if err != nil {
		w.Error(err)
		return
	}

	owner, err := repositoryCtl.Repo.UserRepo().Get(ctx, models.NewGetUserParams().SetName(ownerName))
	if err != nil {
		w.Error(err)
		return
	}

	if operator.Name != ownerName { //todo check public or private
		w.Forbidden()
		return
	}

	repository, err := repositoryCtl.Repo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetOwnerID(owner.ID).SetName(repositoryName))
	if err != nil {
		w.Error(err)
		return
	}

	refName := repository.HEAD
	if params.RefName != nil {
		refName = *params.RefName
	}
	ref, err := repositoryCtl.Repo.BranchRepo().Get(ctx, models.NewGetBranchParams().SetRepositoryID(repository.ID).SetName(refName))
	if err != nil {
		w.Error(err)
		return
	}

	if ref.CommitHash.IsEmpty() {
		w.JSON([]api.Commit{})
		return
	}

	commit, err := repositoryCtl.Repo.CommitRepo(repository.ID).Commit(ctx, ref.CommitHash)
	if err != nil {
		w.Error(err)
		return
	}

	var commits []api.Commit
	commitNode := versionmgr.NewWrapCommitNode(repositoryCtl.Repo.CommitRepo(repository.ID), commit)
	iter := versionmgr.NewCommitPreorderIter(ctx, commitNode, nil, nil)
	for {
		commit, err := iter.Next()
		if err == nil {
			if params.After != nil {
				parseTime, err := time.Parse(time.RFC3339Nano, *params.After)
				if err != nil {
					w.Error(err)
					return
				}
				if commit.Commit().Committer.When.Add(time.Nanosecond).After(parseTime) {
					continue
				}
			}
			if params.Amount != nil && len(commits) == *params.Amount {
				break
			}
			modelCommit := commit.Commit()
			commits = append(commits, api.Commit{
				RepositoryId: modelCommit.RepositoryID,
				Author: api.Signature{
					Email: openapi_types.Email(modelCommit.Author.Email),
					Name:  modelCommit.Author.Name,
					When:  modelCommit.Author.When,
				},

				Committer: api.Signature{
					Email: openapi_types.Email(modelCommit.Committer.Email),
					Name:  modelCommit.Committer.Name,
					When:  modelCommit.Committer.When,
				},
				CreatedAt:    modelCommit.CreatedAt,
				Hash:         modelCommit.Hash.Hex(),
				MergeTag:     modelCommit.MergeTag,
				Message:      modelCommit.Message,
				ParentHashes: hash.HexArrayOfHashes(modelCommit.ParentHashes...),
				TreeHash:     modelCommit.TreeHash.Hex(),
				UpdatedAt:    modelCommit.UpdatedAt,
			})
			continue
		}
		if err == io.EOF {
			break
		}
		w.Error(err)
		return
	}
	w.JSON(commits)
}
