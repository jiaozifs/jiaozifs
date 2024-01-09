package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/uptrace/bun"
)

type WipState int

const (
	Init WipState = iota
	Completed
)

type WorkingInProcess struct {
	bun.BaseModel `bun:"table:wips"`
	ID            uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	CurrentTree   hash.Hash `bun:"current_tree,type:bytea,notnull" json:"current_tree"`
	BaseCommit    hash.Hash `bun:"base_commit,type:bytea,notnull" json:"base_commit"`
	RepositoryID  uuid.UUID `bun:"repository_id,unique:creator_id_repository_id_ref_id_unique,type:uuid,notnull" json:"repository_id"`
	RefID         uuid.UUID `bun:"ref_id,unique:creator_id_repository_id_ref_id_unique,type:uuid,notnull" json:"ref_id"`
	State         WipState  `bun:"state,notnull" json:"state"`
	CreatorID     uuid.UUID `bun:"creator_id,unique:creator_id_repository_id_ref_id_unique,type:uuid,notnull" json:"creator_id"`
	CreatedAt     time.Time `bun:"created_at" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at" json:"updated_at"`
}

type GetWipParams struct {
	id           uuid.UUID
	creatorID    uuid.UUID
	repositoryID uuid.UUID
	refID        uuid.UUID
}

func NewGetWipParams() *GetWipParams {
	return &GetWipParams{}
}

func (gwp *GetWipParams) SetID(id uuid.UUID) *GetWipParams {
	gwp.id = id
	return gwp
}

func (gwp *GetWipParams) SetCreatorID(creatorID uuid.UUID) *GetWipParams {
	gwp.creatorID = creatorID
	return gwp
}

func (gwp *GetWipParams) SetRepositoryID(repositoryID uuid.UUID) *GetWipParams {
	gwp.repositoryID = repositoryID
	return gwp
}

func (gwp *GetWipParams) SetRefID(refID uuid.UUID) *GetWipParams {
	gwp.refID = refID
	return gwp
}

type ListWipParams struct {
	creatorID    uuid.UUID
	repositoryID uuid.UUID
	refID        uuid.UUID
}

func NewListWipParams() *ListWipParams {
	return &ListWipParams{}
}
func (lwp *ListWipParams) SetCreatorID(creatorID uuid.UUID) *ListWipParams {
	lwp.creatorID = creatorID
	return lwp
}

func (lwp *ListWipParams) SetRepositoryID(repositoryID uuid.UUID) *ListWipParams {
	lwp.repositoryID = repositoryID
	return lwp
}

func (lwp *ListWipParams) SetRefID(refID uuid.UUID) *ListWipParams {
	lwp.refID = refID
	return lwp
}

type DeleteWipParams struct {
	id           uuid.UUID
	creatorID    uuid.UUID
	repositoryID uuid.UUID
	refID        uuid.UUID
}

func NewDeleteWipParams() *DeleteWipParams {
	return &DeleteWipParams{}
}

func (dwp *DeleteWipParams) SetID(id uuid.UUID) *DeleteWipParams {
	dwp.id = id
	return dwp
}

func (dwp *DeleteWipParams) SetCreatorID(creatorID uuid.UUID) *DeleteWipParams {
	dwp.creatorID = creatorID
	return dwp
}

func (dwp *DeleteWipParams) SetRepositoryID(repositoryID uuid.UUID) *DeleteWipParams {
	dwp.repositoryID = repositoryID
	return dwp
}

func (dwp *DeleteWipParams) SetRefID(refID uuid.UUID) *DeleteWipParams {
	dwp.refID = refID
	return dwp
}

type UpdateWipParams struct {
	id          uuid.UUID
	currentTree hash.Hash
	baseCommit  hash.Hash
	state       *WipState
	updatedAt   time.Time
}

func NewUpdateWipParams(id uuid.UUID) *UpdateWipParams {
	return &UpdateWipParams{id: id, updatedAt: time.Now()}
}

func (up *UpdateWipParams) SetCurrentTree(currentTree hash.Hash) *UpdateWipParams {
	up.currentTree = currentTree
	return up
}

func (up *UpdateWipParams) SetBaseCommit(commitHash hash.Hash) *UpdateWipParams {
	up.baseCommit = commitHash
	return up
}

func (up *UpdateWipParams) SetState(state WipState) *UpdateWipParams {
	up.state = &state
	return up
}

type IWipRepo interface {
	Insert(ctx context.Context, repo *WorkingInProcess) (*WorkingInProcess, error)
	Get(ctx context.Context, params *GetWipParams) (*WorkingInProcess, error)
	List(ctx context.Context, params *ListWipParams) ([]*WorkingInProcess, error)
	Delete(ctx context.Context, params *DeleteWipParams) (int64, error)
	UpdateByID(ctx context.Context, params *UpdateWipParams) error
}

var _ IWipRepo = (*WipRepo)(nil)

type WipRepo struct {
	db bun.IDB
}

func NewWipRepo(db bun.IDB) IWipRepo {
	return &WipRepo{db: db}
}

func (s *WipRepo) Insert(ctx context.Context, repo *WorkingInProcess) (*WorkingInProcess, error) {
	_, err := s.db.NewInsert().Model(repo).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// Get wip by a group of conditions
func (s *WipRepo) Get(ctx context.Context, params *GetWipParams) (*WorkingInProcess, error) {
	wips := &WorkingInProcess{}
	query := s.db.NewSelect().Model(wips)

	if uuid.Nil != params.id {
		query = query.Where("id = ?", params.id)
	}

	if uuid.Nil != params.creatorID {
		query = query.Where("creator_id = ?", params.creatorID)
	}

	if uuid.Nil != params.repositoryID {
		query = query.Where("repository_id = ?", params.repositoryID)
	}

	if uuid.Nil != params.refID {
		query = query.Where("ref_id = ?", params.refID)
	}

	err := query.Limit(1).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return wips, nil
}

func (s *WipRepo) List(ctx context.Context, params *ListWipParams) ([]*WorkingInProcess, error) {
	var resp []*WorkingInProcess
	query := s.db.NewSelect().Model(&resp)

	if uuid.Nil != params.creatorID {
		query = query.Where("creator_id = ?", params.creatorID)
	}

	if uuid.Nil != params.repositoryID {
		query = query.Where("repository_id = ?", params.repositoryID)
	}

	if uuid.Nil != params.refID {
		query = query.Where("ref_id = ?", params.refID)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Delete remove wip in table by id
func (s *WipRepo) Delete(ctx context.Context, params *DeleteWipParams) (int64, error) {
	query := s.db.NewDelete().Model((*WorkingInProcess)(nil))

	if uuid.Nil != params.creatorID {
		query = query.Where("creator_id = ?", params.creatorID)
	}

	if uuid.Nil != params.repositoryID {
		query = query.Where("repository_id = ?", params.repositoryID)
	}

	if uuid.Nil != params.refID {
		query = query.Where("ref_id = ?", params.refID)
	}

	if uuid.Nil != params.id {
		query = query.Where("id = ?", params.id)
	}
	r, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	row, err := r.RowsAffected()
	if err != nil {
		return 0, err
	}
	return row, nil
}

func (s *WipRepo) UpdateByID(ctx context.Context, updateModel *UpdateWipParams) error {
	updateQuery := s.db.NewUpdate().
		Model((*WorkingInProcess)(nil)).
		Where("id = ?", updateModel.id).
		Set("updated_at = ?", updateModel.updatedAt)

	if updateModel.state != nil {
		updateQuery.Set("state = ?", *updateModel.state)
	}

	if updateModel.currentTree != nil {
		updateQuery.Set("current_tree = ?", updateModel.currentTree)
	}

	if updateModel.baseCommit != nil {
		updateQuery.Set("base_commit = ?", updateModel.baseCommit)
	}
	_, err := updateQuery.Exec(ctx)
	return err
}
