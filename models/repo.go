package models

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MatchMode int

const (
	ExactMatch MatchMode = iota
	PrefixMatch
	SuffixMatch
	LikeMatch
)

type TxOption func(*sql.TxOptions)

func IsolationLevelOption(level sql.IsolationLevel) TxOption {
	return func(opts *sql.TxOptions) {
		opts.Isolation = level
	}
}

type IRepo interface {
	Transaction(ctx context.Context, fn func(repo IRepo) error, opts ...TxOption) error
	UserRepo() IUserRepo
	MergeRequestRepo() IMergeRequestRepo
	FileTreeRepo(repoID uuid.UUID) IFileTreeRepo
	CommitRepo(repoID uuid.UUID) ICommitRepo
	TagRepo(repoID uuid.UUID) ITagRepo
	BranchRepo() IBranchRepo
	RepositoryRepo() IRepositoryRepo
	WipRepo() IWipRepo
}

type PgRepo struct {
	db bun.IDB
}

func NewRepo(db bun.IDB) IRepo {
	return &PgRepo{
		db: db,
	}
}

func (repo *PgRepo) Transaction(ctx context.Context, fn func(repo IRepo) error, opts ...TxOption) error {
	sqlOpt := &sql.TxOptions{}
	for _, opt := range opts {
		opt(sqlOpt)
	}
	return repo.db.RunInTx(ctx, sqlOpt, func(ctx context.Context, tx bun.Tx) error {
		return fn(NewRepo(tx))
	})
}

func (repo *PgRepo) UserRepo() IUserRepo {
	return NewUserRepo(repo.db)
}

// MergeRequestRepo returns an instance of the IMergeRequestRepo interface.
//
// It does not take any parameters.
// It returns an IMergeRequestRepo.
func (repo *PgRepo) MergeRequestRepo() IMergeRequestRepo {
	return NewMergeRequestRepo(repo.db)
}

func (repo *PgRepo) FileTreeRepo(repoID uuid.UUID) IFileTreeRepo {
	return NewFileTree(repo.db, repoID)
}

func (repo *PgRepo) CommitRepo(repoID uuid.UUID) ICommitRepo {
	return NewCommitRepo(repo.db, repoID)
}

func (repo *PgRepo) TagRepo(repoID uuid.UUID) ITagRepo {
	return NewTagRepo(repo.db, repoID)
}

func (repo *PgRepo) BranchRepo() IBranchRepo {
	return NewBranchRepo(repo.db)
}

func (repo *PgRepo) RepositoryRepo() IRepositoryRepo {
	return NewRepositoryRepo(repo.db)
}

func (repo *PgRepo) WipRepo() IWipRepo {
	return NewWipRepo(repo.db)
}
