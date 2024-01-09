package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MergeState int

const (
	MergeStateInit   MergeState = 1
	MergeStateMerged MergeState = 2
	MergeStateClosed MergeState = 3
)

type MergeRequest struct {
	bun.BaseModel `bun:"table:merge_requests"`
	ID            uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Sequence      uint64    `bun:"mr_sequence,unique:target_seq,notnull" json:"sequence"`
	SourceRepoID  uuid.UUID `bun:"source_repo_id,type:bytea,notnull" json:"source_repo_id"`
	TargetRepoID  uuid.UUID `bun:"target_repo_id,unique:target_seq,type:bytea,notnull" json:"target_repo_id"`

	TargetBranchID uuid.UUID  `bun:"target_branch_id,type:bytea,notnull" json:"target_branch_id"`
	SourceBranchID uuid.UUID  `bun:"source_branch_id,type:bytea,notnull" json:"source_branch_id"`
	Title          string     `bun:"title,notnull" json:"title"`
	MergeState     MergeState `bun:"merge_state,notnull" json:"merge_state"`
	Description    *string    `bun:"description" json:"description"`

	AuthorID uuid.UUID `bun:"author_id,type:bytea,notnull" json:"author_id"`

	CreatedAt time.Time `bun:"created_at,notnull" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updated_at"`
}

type GetMergeRequestParams struct {
	id             uuid.UUID
	sequence       *uint64
	targetRepoID   uuid.UUID
	targetBranchID uuid.UUID
	sourceBranchID uuid.UUID
	state          *MergeState
}

func NewGetMergeRequestParams() *GetMergeRequestParams {
	return &GetMergeRequestParams{}
}

func (gmr *GetMergeRequestParams) SetID(id uuid.UUID) *GetMergeRequestParams {
	gmr.id = id
	return gmr
}

func (gmr *GetMergeRequestParams) SetNumber(sequence uint64) *GetMergeRequestParams {
	gmr.sequence = &sequence
	return gmr
}

func (gmr *GetMergeRequestParams) SetTargetRepo(targetRepoID uuid.UUID) *GetMergeRequestParams {
	gmr.targetRepoID = targetRepoID
	return gmr
}

func (gmr *GetMergeRequestParams) SetTargetBranch(targetBranchID uuid.UUID) *GetMergeRequestParams {
	gmr.targetBranchID = targetBranchID
	return gmr
}

func (gmr *GetMergeRequestParams) SetSourceBranch(sourceBranchID uuid.UUID) *GetMergeRequestParams {
	gmr.sourceBranchID = sourceBranchID
	return gmr
}

func (gmr *GetMergeRequestParams) SetState(state MergeState) *GetMergeRequestParams {
	gmr.state = &state
	return gmr
}

type DeleteMergeRequestParams struct {
	sequence     *uint64
	targetRepoID uuid.UUID
}

func NewDeleteMergeRequestParams() *DeleteMergeRequestParams {
	return &DeleteMergeRequestParams{}
}

func (dmr *DeleteMergeRequestParams) SetNumber(sequence uint64) *DeleteMergeRequestParams {
	dmr.sequence = &sequence
	return dmr
}

func (dmr *DeleteMergeRequestParams) SetTargetRepo(targetRepoID uuid.UUID) *DeleteMergeRequestParams {
	dmr.targetRepoID = targetRepoID
	return dmr
}

type UpdateMergeRequestParams struct {
	sequence    uint64
	targetRepo  uuid.UUID
	updateTime  time.Time
	title       *string
	description *string
	state       *MergeState
}

func NewUpdateMergeRequestParams(targetRepoID uuid.UUID, sequence uint64) *UpdateMergeRequestParams {
	return &UpdateMergeRequestParams{
		sequence:   sequence,
		targetRepo: targetRepoID,
		updateTime: time.Now(),
	}
}

func (u *UpdateMergeRequestParams) SetTitle(title string) *UpdateMergeRequestParams {
	u.title = &title
	return u
}

func (u *UpdateMergeRequestParams) SetDescription(description string) *UpdateMergeRequestParams {
	u.description = &description
	return u
}

func (u *UpdateMergeRequestParams) SetState(state MergeState) *UpdateMergeRequestParams {
	u.state = &state
	return u
}

type ListMergeRequestParams struct {
	after        *time.Time
	amount       int
	targetRepoID uuid.UUID
	state        *MergeState
}

func NewListMergeRequestParams() *ListMergeRequestParams {
	return &ListMergeRequestParams{}
}

func (lmr *ListMergeRequestParams) SetTargetRepoID(targetRepoID uuid.UUID) *ListMergeRequestParams {
	lmr.targetRepoID = targetRepoID
	return lmr
}

func (lmr *ListMergeRequestParams) SetAfter(after time.Time) *ListMergeRequestParams {
	lmr.after = &after
	return lmr
}

func (lmr *ListMergeRequestParams) SetAmount(amount int) *ListMergeRequestParams {
	lmr.amount = amount
	return lmr
}

func (lmr *ListMergeRequestParams) SetMergeState(state MergeState) *ListMergeRequestParams {
	lmr.state = &state
	return lmr
}

type IMergeRequestRepo interface {
	Insert(ctx context.Context, ref *MergeRequest) (*MergeRequest, error)
	Get(ctx context.Context, params *GetMergeRequestParams) (*MergeRequest, error)
	List(ctx context.Context, params *ListMergeRequestParams) ([]MergeRequest, bool, error)
	UpdateByID(ctx context.Context, params *UpdateMergeRequestParams) error
	Delete(ctx context.Context, params *DeleteMergeRequestParams) (int64, error)
}

var _ IMergeRequestRepo = (*MergeRequestRepo)(nil)

type MergeRequestRepo struct {
	db bun.IDB
}

func NewMergeRequestRepo(db bun.IDB) IMergeRequestRepo {
	return &MergeRequestRepo{db: db}
}

func (m MergeRequestRepo) Insert(ctx context.Context, mr *MergeRequest) (*MergeRequest, error) {
	_, err := m.db.NewRaw(`
		WITH INCNUMBER AS (
				SELECT MAX(mr_sequence) as max_seq from merge_requests WHERE merge_requests.target_repo_id  = ?
			)
		INSERT INTO merge_requests (mr_sequence, target_branch_id,source_branch_id,source_repo_id,target_repo_id,title,merge_state,description,author_id,created_at,updated_at)
		SELECT COALESCE(max_seq,0)+1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? FROM INCNUMBER
		RETURNING merge_requests.id, merge_requests.mr_sequence;
`,
		mr.TargetRepoID, mr.TargetBranchID, mr.SourceBranchID, mr.SourceRepoID, mr.TargetRepoID, mr.Title, mr.MergeState, mr.Description, mr.AuthorID, mr.CreatedAt, mr.UpdatedAt,
	).Exec(ctx, mr)
	if err != nil {
		return nil, err
	}
	return mr, nil
}

func (m MergeRequestRepo) Get(ctx context.Context, params *GetMergeRequestParams) (*MergeRequest, error) {
	mergeRequest := &MergeRequest{}
	query := m.db.NewSelect().Model(mergeRequest)

	if params.sequence != nil {
		query = query.Where("mr_sequence = ?", params.sequence)
	}

	if params.targetRepoID != uuid.Nil {
		query = query.Where("target_repo_id = ?", params.targetRepoID)
	}

	if params.id != uuid.Nil {
		query = query.Where("id = ?", params.id)
	}

	if params.state != nil {
		query = query.Where("merge_state = ?", params.state)
	}

	if params.targetBranchID != uuid.Nil {
		query = query.Where("target_branch_id = ?", params.targetBranchID)
	}

	if params.sourceBranchID != uuid.Nil {
		query = query.Where("source_branch_id = ?", params.sourceBranchID)
	}
	return mergeRequest, query.Limit(1).Scan(ctx)
}

func (m MergeRequestRepo) List(ctx context.Context, params *ListMergeRequestParams) ([]MergeRequest, bool, error) {
	mergeRequest := make([]MergeRequest, 0)
	query := m.db.NewSelect().Model(&mergeRequest)

	if params.targetRepoID != uuid.Nil {
		query = query.Where("target_repo_id = ?", params.targetRepoID)
	}
	if params.state != nil {
		query = query.Where("merge_state = ?", params.state)
	}

	query = query.Order("updated_at DESC")
	if params.after != nil {
		query = query.Where("updated_at < ?", *params.after)
	}

	err := query.Limit(params.amount).Scan(ctx)
	return mergeRequest, len(mergeRequest) == params.amount, err
}

func (m MergeRequestRepo) Delete(ctx context.Context, params *DeleteMergeRequestParams) (int64, error) {
	query := m.db.NewDelete().Model((*MergeRequest)(nil))
	if params.sequence != nil {
		query = query.Where("mr_sequence = ?", params.sequence)
	}
	if params.targetRepoID != uuid.Nil {
		query = query.Where("target_repo_id = ?", params.targetRepoID)
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

func (m MergeRequestRepo) UpdateByID(ctx context.Context, updateModel *UpdateMergeRequestParams) error {
	updateQuery := m.db.NewUpdate().
		Model((*MergeRequest)(nil)).
		Where("mr_sequence = ?", updateModel.sequence).
		Where("target_repo_id = ?", updateModel.targetRepo).
		Set("updated_at = ?", updateModel.updateTime)

	if updateModel.description != nil {
		updateQuery.Set("description = ?", *updateModel.description)
	}
	if updateModel.title != nil {
		updateQuery.Set("title = ?", *updateModel.title)
	}
	if updateModel.state != nil {
		updateQuery.Set("merge_state = ?", *updateModel.state)
	}
	_, err := updateQuery.Exec(ctx)
	return err
}
