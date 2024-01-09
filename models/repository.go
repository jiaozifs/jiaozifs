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
	id        uuid.UUID
	creatorID uuid.UUID
	ownerID   uuid.UUID
	name      *string
}

func NewGetRepoParams() *GetRepoParams {
	return &GetRepoParams{}
}

func (gup *GetRepoParams) SetID(id uuid.UUID) *GetRepoParams {
	gup.id = id
	return gup
}

func (gup *GetRepoParams) SetOwnerID(id uuid.UUID) *GetRepoParams {
	gup.ownerID = id
	return gup
}

func (gup *GetRepoParams) SetCreatorID(creatorID uuid.UUID) *GetRepoParams {
	gup.creatorID = creatorID
	return gup
}

func (gup *GetRepoParams) SetName(name string) *GetRepoParams {
	gup.name = &name
	return gup
}

type ListRepoParams struct {
	id        uuid.UUID
	creatorID uuid.UUID
	ownerID   uuid.UUID
	name      *string
	nameMatch MatchMode
	after     *time.Time
	amount    int
}

func NewListRepoParams() *ListRepoParams {
	return &ListRepoParams{}
}

func (lrp *ListRepoParams) SetID(id uuid.UUID) *ListRepoParams {
	lrp.id = id
	return lrp
}
func (lrp *ListRepoParams) SetOwnerID(ownerID uuid.UUID) *ListRepoParams {
	lrp.ownerID = ownerID
	return lrp
}

func (lrp *ListRepoParams) SetName(name string, match MatchMode) *ListRepoParams {
	lrp.name = &name
	lrp.nameMatch = match
	return lrp
}

func (lrp *ListRepoParams) SetCreatorID(creatorID uuid.UUID) *ListRepoParams {
	lrp.creatorID = creatorID
	return lrp
}

func (lrp *ListRepoParams) SetAfter(after time.Time) *ListRepoParams {
	lrp.after = &after
	return lrp
}

func (lrp *ListRepoParams) SetAmount(amount int) *ListRepoParams {
	lrp.amount = amount
	return lrp
}

type DeleteRepoParams struct {
	id      uuid.UUID
	ownerID uuid.UUID
	name    *string
}

func NewDeleteRepoParams() *DeleteRepoParams {
	return &DeleteRepoParams{}
}

func (drp *DeleteRepoParams) SetID(id uuid.UUID) *DeleteRepoParams {
	drp.id = id
	return drp
}

func (drp *DeleteRepoParams) SetOwnerID(ownerID uuid.UUID) *DeleteRepoParams {
	drp.ownerID = ownerID
	return drp
}

func (drp *DeleteRepoParams) SetName(name string) *DeleteRepoParams {
	drp.name = &name
	return drp
}

type UpdateRepoParams struct {
	id          uuid.UUID
	description *string
	head        *string
}

func NewUpdateRepoParams(id uuid.UUID) *UpdateRepoParams {
	return &UpdateRepoParams{
		id: id,
	}
}

func (up *UpdateRepoParams) SetDescription(description string) *UpdateRepoParams {
	up.description = &description
	return up
}

func (up *UpdateRepoParams) SetHead(head string) *UpdateRepoParams {
	up.head = &head
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

	if uuid.Nil != params.id {
		query = query.Where("id = ?", params.id)
	}

	if uuid.Nil != params.creatorID {
		query = query.Where("creator_id = ?", params.creatorID)
	}

	if uuid.Nil != params.ownerID {
		query = query.Where("owner_id = ?", params.ownerID)
	}

	if params.name != nil {
		query = query.Where("name = ?", *params.name)
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

	if uuid.Nil != params.creatorID {
		query = query.Where("creator_id = ?", params.creatorID)
	}

	if uuid.Nil != params.ownerID {
		query = query.Where("owner_id = ?", params.ownerID)
	}

	if params.name != nil {
		switch params.nameMatch {
		case ExactMatch:
			query = query.Where("name = ?", *params.name)
		case PrefixMatch:
			query = query.Where("name LIKE ?", *params.name+"%")
		case SuffixMatch:
			query = query.Where("name LIKE ?", "%"+*params.name)
		case LikeMatch:
			query = query.Where("name LIKE ?", "%"+*params.name+"%")
		}
	}

	query = query.Order("updated_at DESC")
	if params.after != nil {
		query = query.Where("updated_at < ?", *params.after)
	}

	err := query.Limit(params.amount).Scan(ctx)
	return repos, len(repos) == params.amount, err
}

func (r *RepositoryRepo) Delete(ctx context.Context, params *DeleteRepoParams) (int64, error) {
	query := r.db.NewDelete().Model((*Repository)(nil))
	if uuid.Nil != params.id {
		query = query.Where("id = ?", params.id)
	}

	if params.name != nil {
		query = query.Where("name = ?", params.name)
	}

	if uuid.Nil != params.ownerID {
		query = query.Where("owner_id = ?", params.ownerID)
	}

	sqlResult, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	affectedRows, err := sqlResult.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affectedRows, err
}

func (r *RepositoryRepo) UpdateByID(ctx context.Context, updateModel *UpdateRepoParams) error {
	updateQuery := r.db.NewUpdate().Model((*Repository)(nil)).Where("id = ?", updateModel.id)
	if updateModel.description != nil {
		updateQuery.Set("description = ?", *updateModel.description)
	}
	if updateModel.head != nil {
		updateQuery.Set("head = ?", *updateModel.head)
	}
	_, err := updateQuery.Exec(ctx)
	return err
}
