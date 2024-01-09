package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/uptrace/bun"
)

type Branch struct {
	bun.BaseModel `bun:"table:branches"`
	ID            uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	// RepositoryId which repository this branch belong
	RepositoryID uuid.UUID `bun:"repository_id,type:uuid,notnull" json:"repository_id"`
	CommitHash   hash.Hash `bun:"commit_hash,type:bytea,notnull" json:"commit_hash"`
	// Path name/path of branch
	Name string `bun:"name,notnull" json:"name"`
	// Description
	Description *string `bun:"description" json:"description,omitempty"`
	// CreatorID who create this branch
	CreatorID uuid.UUID `bun:"creator_id,type:uuid,notnull" json:"creator_id"`

	CreatedAt time.Time `bun:"created_at,notnull" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updated_at"`
}

type GetBranchParams struct {
	id           uuid.UUID
	repositoryID uuid.UUID
	name         *string
}

func NewGetBranchParams() *GetBranchParams {
	return &GetBranchParams{}
}

func (gup *GetBranchParams) SetID(id uuid.UUID) *GetBranchParams {
	gup.id = id
	return gup
}

func (gup *GetBranchParams) SetRepositoryID(repositoryID uuid.UUID) *GetBranchParams {
	gup.repositoryID = repositoryID
	return gup
}

func (gup *GetBranchParams) SetName(name string) *GetBranchParams {
	gup.name = &name
	return gup
}

type DeleteBranchParams struct {
	id           uuid.UUID
	repositoryID uuid.UUID
	name         *string
}

func NewDeleteBranchParams() *DeleteBranchParams {
	return &DeleteBranchParams{}
}

func (gup *DeleteBranchParams) SetRepositoryID(repositoryID uuid.UUID) *DeleteBranchParams {
	gup.repositoryID = repositoryID
	return gup
}
func (gup *DeleteBranchParams) SetID(id uuid.UUID) *DeleteBranchParams {
	gup.id = id
	return gup
}

func (gup *DeleteBranchParams) SetName(name string) *DeleteBranchParams {
	gup.name = &name
	return gup
}

type UpdateBranchParams struct {
	id         uuid.UUID
	commitHash hash.Hash
}

func NewUpdateBranchParams(id uuid.UUID) *UpdateBranchParams {
	return &UpdateBranchParams{id: id}
}

func (up *UpdateBranchParams) SetCommitHash(commitHash hash.Hash) *UpdateBranchParams {
	up.commitHash = commitHash
	return up
}

type ListBranchParams struct {
	RepositoryID uuid.UUID
	Name         *string
	NameMatch    MatchMode
	After        *string
	Amount       int
}

func NewListBranchParams() *ListBranchParams {
	return &ListBranchParams{}
}

func (gup *ListBranchParams) SetRepositoryID(repositoryID uuid.UUID) *ListBranchParams {
	gup.RepositoryID = repositoryID
	return gup
}

func (gup *ListBranchParams) SetName(name string, match MatchMode) *ListBranchParams {
	gup.Name = &name
	gup.NameMatch = match
	return gup
}

func (gup *ListBranchParams) SetAfter(after string) *ListBranchParams {
	gup.After = &after
	return gup
}

func (gup *ListBranchParams) SetAmount(amount int) *ListBranchParams {
	gup.Amount = amount
	return gup
}

type IBranchRepo interface {
	Insert(ctx context.Context, repo *Branch) (*Branch, error)
	UpdateByID(ctx context.Context, params *UpdateBranchParams) error
	Get(ctx context.Context, id *GetBranchParams) (*Branch, error)

	List(ctx context.Context, params *ListBranchParams) ([]*Branch, bool, error)
	Delete(ctx context.Context, params *DeleteBranchParams) (int64, error)
}

var _ IBranchRepo = (*BranchRepo)(nil)

type BranchRepo struct {
	db bun.IDB
}

func NewBranchRepo(db bun.IDB) IBranchRepo {
	return &BranchRepo{db: db}
}

func (r BranchRepo) Insert(ctx context.Context, branch *Branch) (*Branch, error) {
	_, err := r.db.NewInsert().Model(branch).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return branch, nil
}

func (r BranchRepo) Get(ctx context.Context, params *GetBranchParams) (*Branch, error) {
	repo := &Branch{}
	query := r.db.NewSelect().Model(repo)

	if uuid.Nil != params.id {
		query = query.Where("id = ?", params.id)
	}

	if uuid.Nil != params.repositoryID {
		query = query.Where("repository_id = ?", params.repositoryID)
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

func (r BranchRepo) List(ctx context.Context, params *ListBranchParams) ([]*Branch, bool, error) {
	var branches []*Branch
	query := r.db.NewSelect().Model(&branches)

	if uuid.Nil != params.RepositoryID {
		query = query.Where("repository_id = ?", params.RepositoryID)
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

	query = query.Order("name ASC")
	if params.After != nil {
		query = query.Where("name > ?", *params.After)
	}

	err := query.Limit(params.Amount).Scan(ctx)
	return branches, len(branches) == params.Amount, err
}

func (r BranchRepo) Delete(ctx context.Context, params *DeleteBranchParams) (int64, error) {
	query := r.db.NewDelete().Model((*Branch)(nil))

	if uuid.Nil != params.id {
		query = query.Where("id = ?", params.id)
	}

	if uuid.Nil != params.repositoryID {
		query = query.Where("repository_id = ?", params.repositoryID)
	}

	if params.name != nil {
		query = query.Where("name = ?", *params.name)
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

func (r BranchRepo) UpdateByID(ctx context.Context, updateModel *UpdateBranchParams) error {
	updateQuery := r.db.NewUpdate().Model((*Branch)(nil)).Where("id = ?", updateModel.id)
	if updateModel.commitHash != nil {
		updateQuery.Set("commit_hash = ?", updateModel.commitHash)
	}
	_, err := updateQuery.Exec(ctx)
	return err
}
