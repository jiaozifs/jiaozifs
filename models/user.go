package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel     `bun:"table:users"`
	ID                uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name              string    `bun:"name,notnull" json:"name"`
	Email             string    `bun:"email,notnull" json:"email"`
	EncryptedPassword string    `bun:"encrypted_password,notnull" json:"encrypted_password"`
	CurrentSignInAt   time.Time `bun:"current_sign_in_at" json:"current_sign_in_at"`
	LastSignInAt      time.Time `bun:"last_sign_in_at" json:"last_sign_in_at"`
	CurrentSignInIP   string    `bun:"current_sign_in_ip" json:"current_sign_in_ip"`
	LastSignInIP      string    `bun:"last_sign_in_ip" json:"last_sign_in_ip"`
	CreatedAt         time.Time `bun:"created_at,notnull" json:"created_at"`
	UpdatedAt         time.Time `bun:"updated_at,notnull" json:"updated_at"`
}

type GetUserParams struct {
	id    uuid.UUID
	name  *string
	email *string
}

func NewGetUserParams() *GetUserParams {
	return &GetUserParams{}
}

func (gup *GetUserParams) SetID(id uuid.UUID) *GetUserParams {
	gup.id = id
	return gup
}

func (gup *GetUserParams) SetName(name string) *GetUserParams {
	gup.name = &name
	return gup
}

func (gup *GetUserParams) SetEmail(email string) *GetUserParams {
	gup.email = &email
	return gup
}

type CountUserParams = GetUserParams

func NewCountUserParam() *CountUserParams {
	return &CountUserParams{}
}

type IUserRepo interface {
	Get(ctx context.Context, params *GetUserParams) (*User, error)
	Count(ctx context.Context, params *CountUserParams) (int, error)
	Insert(ctx context.Context, user *User) (*User, error)
	GetEPByName(ctx context.Context, name string) (string, error)
}

var _ IUserRepo = (*UserRepo)(nil)

type UserRepo struct {
	db bun.IDB
}

func NewUserRepo(db bun.IDB) IUserRepo {
	return &UserRepo{db: db}
}

func (userRepo *UserRepo) Get(ctx context.Context, params *GetUserParams) (*User, error) {
	user := &User{}
	query := userRepo.db.NewSelect().Model(user)

	if uuid.Nil != params.id {
		query = query.Where("id = ?", params.id)
	}

	if params.name != nil {
		query = query.Where("name = ?", params.name)
	}

	if params.email != nil {
		query = query.Where("email = ?", *params.email)
	}

	err := query.Limit(1).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (userRepo *UserRepo) Count(ctx context.Context, params *GetUserParams) (int, error) {
	query := userRepo.db.NewSelect().Model((*User)(nil))

	if uuid.Nil != params.id {
		query = query.Where("id = ?", params.id)
	}

	if params.name != nil {
		query = query.Where("name = ?", params.name)
	}

	if params.email != nil {
		query = query.Where("email = ?", *params.email)
	}

	return query.Count(ctx)
}

func (userRepo *UserRepo) Insert(ctx context.Context, user *User) (*User, error) {
	_, err := userRepo.db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (userRepo *UserRepo) GetEPByName(ctx context.Context, name string) (string, error) {
	var ep string
	return ep, userRepo.db.NewSelect().
		Model((*User)(nil)).Column("encrypted_password").
		Where("name = ?", name).
		Scan(ctx, &ep)
}
