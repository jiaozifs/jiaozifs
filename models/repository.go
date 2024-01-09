package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Repository struct {
	bun.BaseModel `bun:"table:repositories" json:"bun_._base_model"`
	ID            uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name          string    `bun:"name,unique:name_owner_unique,notnull" json:"name"`
	OwnerID       uuid.UUID `bun:"owner_id,unique:name_owner_unique,type:uuid,notnull" json:"owner_id"`
	HEAD          string    `bun:"head,notnull" json:"head"`

	UsePublicStorage bool    `bun:"use_public_storage,notnull" json:"use_public_storage"`
	StorageNamespace *string `bun:"storage_namespace" json:"storage_namespace,omitempty"`

	StorageAdapterParams *string `bun:"storage_adapter_params" json:"storage_adapter_params,omitempty"`

	Description *string   `bun:"description" json:"description,omitempty"`
	CreatorID   uuid.UUID `bun:"creator_id,type:uuid,notnull" json:"creator_id"`

	CreatedAt time.Time `bun:"created_at,notnull" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updated_at"`
}

type GetRepoParams struct {
	ID        uuid.UUID
	CreatorID uuid.UUID
	OwnerID   uuid.UUID
	Name      *string
}

func NewGetRepoParams() *GetRepoParams {
	return &GetRepoParams{}
}

func (gup *GetRepoParams) SetID(id uuid.UUID) *GetRepoParams {
	gup.ID = id
	return gup
}

func (gup *GetRepoParams) SetOwnerID(id uuid.UUID) *GetRepoParams {
	gup.OwnerID = id
	return gup
}

func (gup *GetRepoParams) SetCreatorID(creatorID uuid.UUID) *GetRepoParams {
	gup.CreatorID = creatorID
	return gup
}

func (gup *GetRepoParams) SetName(name string) *GetRepoParams {
	gup.Name = &name
	return gup
}

type ListRepoParams struct {
	ID        uuid.UUID
	CreatorID uuid.UUID
	OwnerID   uuid.UUID
	Name      *string
	NameMatch MatchMode
	After     *time.Time
	Amount    int
}

func NewListRepoParams() *ListRepoParams {
	return &ListRepoParams{}
}

func (lrp *ListRepoParams) SetID(id uuid.UUID) *ListRepoParams {
	lrp.ID = id
	return lrp
}
func (lrp *ListRepoParams) SetOwnerID(ownerID uuid.UUID) *ListRepoParams {
	lrp.OwnerID = ownerID
	return lrp
}

func (lrp *ListRepoParams) SetName(name string, match MatchMode) *ListRepoParams {
	lrp.Name = &name
	lrp.NameMatch = match
	return lrp
}

func (lrp *ListRepoParams) SetCreatorID(creatorID uuid.UUID) *ListRepoParams {
	lrp.CreatorID = creatorID
	return lrp
}

func (lrp *ListRepoParams) SetAfter(after time.Time) *ListRepoParams {
	lrp.After = &after
	return lrp
}

func (lrp *ListRepoParams) SetAmount(amount int) *ListRepoParams {
	lrp.Amount = amount
	return lrp
}

type DeleteRepoParams struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	Name    *string
}

func NewDeleteRepoParams() *DeleteRepoParams {
	return &DeleteRepoParams{}
}

func (drp *DeleteRepoParams) SetID(id uuid.UUID) *DeleteRepoParams {
	drp.ID = id
	return drp
}

func (drp *DeleteRepoParams) SetOwnerID(ownerID uuid.UUID) *DeleteRepoParams {
	drp.OwnerID = ownerID
	return drp
}

func (drp *DeleteRepoParams) SetName(name string) *DeleteRepoParams {
	drp.Name = &name
	return drp
}

type UpdateRepoParams struct {
	bun.BaseModel `bun:"table:repositories"`
	ID            uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	Description   *string   `bun:"description"`
	HEAD          *string   `bun:"head,notnull"`
}

func NewUpdateRepoParams(id uuid.UUID) *UpdateRepoParams {
	return &UpdateRepoParams{
		ID: id,
	}
}

func (up *UpdateRepoParams) SetDescription(description string) *UpdateRepoParams {
	up.Description = &description
	return up
}

func (up *UpdateRepoParams) SetHead(head string) *UpdateRepoParams {
	up.HEAD = &head
	return up
}

type IRepositoryRepo interface {
	Insert(ctx context.Context, repo *Repository) (*Repository, error)
	Get(ctx context.Context, params *GetRepoParams) (*Repository, error)

	List(ctx context.Context, params *ListRepoParams) ([]*Repository, bool, error)
	Delete(ctx context.Context, params *DeleteRepoParams) (int64, error)
	UpdateByID(ctx context.Context, updateModel *UpdateRepoParams) error
}

var _ IRepositoryRepo = (*RepositoryRepo)(nil)

type RepositoryRepo struct {
	db bun.IDB
}

func NewRepositoryRepo(db bun.IDB) IRepositoryRepo {
	return &RepositoryRepo{db: db}
}

func (r *RepositoryRepo) Insert(ctx context.Context, repo *Repository) (*Repository, error) {
	_, err := r.db.NewInsert().Model(repo).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *RepositoryRepo) Get(ctx context.Context, params *GetRepoParams) (*Repository, error) {
	repo := &Repository{}
	query := r.db.NewSelect().Model(repo)

	if uuid.Nil != params.ID {
		query = query.Where("id = ?", params.ID)
	}

	if uuid.Nil != params.CreatorID {
		query = query.Where("creator_id = ?", params.CreatorID)
	}

	if uuid.Nil != params.OwnerID {
		query = query.Where("owner_id = ?", params.OwnerID)
	}

	if params.Name != nil {
		query = query.Where("name = ?", *params.Name)
	}

	err := query.Limit(1).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *RepositoryRepo) List(ctx context.Context, params *ListRepoParams) ([]*Repository, bool, error) {
	repos := []*Repository{}
	query := r.db.NewSelect().Model(&repos)

	if uuid.Nil != params.CreatorID {
		query = query.Where("creator_id = ?", params.CreatorID)
	}

	if uuid.Nil != params.OwnerID {
		query = query.Where("owner_id = ?", params.OwnerID)
	}

	if params.Name != nil {
		switch params.NameMatch {
		case ExactMatch:
			query = query.Where("name = ?", *params.Name)
		case PrefixMatch:
			query = query.Where("name LIKE ?", *params.Name+"%")
		case SuffixMatch:
			query = query.Where("name LIKE ?", "%"+*params.Name)
		case LikeMatch:
			query = query.Where("name LIKE ?", "%"+*params.Name+"%")
		}
	}

	query = query.Order("updated_at DESC")
	if params.After != nil {
		query = query.Where("updated_at < ?", *params.After)
	}

	err := query.Limit(params.Amount).Scan(ctx)
	return repos, len(repos) == params.Amount, err
}

func (r *RepositoryRepo) Delete(ctx context.Context, params *DeleteRepoParams) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	defer func() {
		if err != nil {
			tx.Rollback() //nolint
		}
	}()

	affectedRowsRepos, err := r.deleteRepos(ctx, &tx, params)
	if err != nil {
		return 0, err
	}

	affectedRowsCommits, err := r.deleteCommits(ctx, &tx, params.ID)
	if err != nil {
		return 0, err
	}

	affectedRowsTags, err := r.deleteTags(ctx, &tx, params.ID)
	if err != nil {
		return 0, err
	}

	affectedRowsObejcts, err := r.deleteObjects(ctx, &tx, params.ID)
	if err != nil {
		return 0, err
	}

	affectedRowsWips, err := r.deleteWips(ctx, &tx, params.ID)
	if err != nil {
		return 0, err
	}

	affectedRowsBranches, err := r.deleteBranches(ctx, &tx, params.ID)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return affectedRowsCommits + affectedRowsTags + affectedRowsObejcts + affectedRowsWips + affectedRowsBranches + affectedRowsRepos, nil
}

func (r *RepositoryRepo) deleteCommits(ctx context.Context, tx *bun.Tx, repoID uuid.UUID) (int64, error) {
	query := tx.NewDelete().Model((*Commit)(nil))
	if uuid.Nil != repoID {
		query = query.Where("repository_id = ?", repoID)
	}

	sqlResult, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	affectedRows, _ := sqlResult.RowsAffected()
	return affectedRows, nil
}

func (r *RepositoryRepo) deleteTags(ctx context.Context, tx *bun.Tx, repoID uuid.UUID) (int64, error) {
	query := tx.NewDelete().Model((*Tag)(nil))
	if uuid.Nil != repoID {
		query = query.Where("repository_id = ?", repoID)
	}

	sqlResult, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	affectedRows, _ := sqlResult.RowsAffected()
	return affectedRows, nil
}

func (r *RepositoryRepo) deleteObjects(ctx context.Context, tx *bun.Tx, repoID uuid.UUID) (int64, error) {
	query := tx.NewDelete().Model((*TreeNode)(nil))
	if uuid.Nil != repoID {
		query = query.Where("repository_id = ?", repoID)
	}

	sqlResult, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	affectedRows, _ := sqlResult.RowsAffected()
	return affectedRows, nil
}

func (r *RepositoryRepo) deleteWips(ctx context.Context, tx *bun.Tx, repoID uuid.UUID) (int64, error) {
	query := tx.NewDelete().Model((*WorkingInProcess)(nil))
	if uuid.Nil != repoID {
		query = query.Where("repository_id = ?", repoID)
	}

	sqlResult, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	affectedRows, _ := sqlResult.RowsAffected()
	return affectedRows, nil
}

func (r *RepositoryRepo) deleteBranches(ctx context.Context, tx *bun.Tx, repoID uuid.UUID) (int64, error) {
	query := tx.NewDelete().Model((*Branch)(nil))
	if uuid.Nil != repoID {
		query = query.Where("repository_id = ?", repoID)
	}

	sqlResult, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	affectedRows, _ := sqlResult.RowsAffected()
	return affectedRows, nil
}

func (r *RepositoryRepo) deleteRepos(ctx context.Context, tx *bun.Tx, params *DeleteRepoParams) (int64, error) {
	query := tx.NewDelete().Model((*Repository)(nil))
	if uuid.Nil != params.ID {
		query = query.Where("id = ?", params.ID)
	}

	if params.Name != nil {
		query = query.Where("name = ?", params.Name)
	}

	if uuid.Nil != params.OwnerID {
		query = query.Where("owner_id = ?", params.OwnerID)
	}

	sqlResult, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	affectedRows, _ := sqlResult.RowsAffected()
	return affectedRows, nil
}

func (r *RepositoryRepo) UpdateByID(ctx context.Context, updateModel *UpdateRepoParams) error {
	updateQuery := r.db.NewUpdate().Model((*Repository)(nil)).Where("id = ?", updateModel.ID)
	if updateModel.Description != nil {
		updateQuery.Set("description = ?", *updateModel.Description)
	}
	if updateModel.HEAD != nil {
		updateQuery.Set("head = ?", *updateModel.HEAD)
	}
	_, err := updateQuery.Exec(ctx)
	return err
}
