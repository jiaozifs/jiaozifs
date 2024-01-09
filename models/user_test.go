package models_test

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/stretchr/testify/require"
)

var dbTimeCmpOpt = cmp.Comparer(func(x, y time.Time) bool {
	return x.Unix() == y.Unix()
})

func TestNewUserRepo(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewUserRepo(db)

	userModel := &models.User{}
	require.NoError(t, gofakeit.Struct(userModel))
	newUser, err := repo.Insert(ctx, userModel)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, newUser.ID)

	user, err := repo.Get(ctx, models.NewGetUserParams().SetID(newUser.ID))
	require.NoError(t, err)

	require.True(t, cmp.Equal(userModel, user, dbTimeCmpOpt))

	ep, err := repo.GetEPByName(ctx, newUser.Name)
	require.NoError(t, err)
	require.True(t, cmp.Equal(userModel.EncryptedPassword, ep))

	userByEmail, err := repo.Get(ctx, models.NewGetUserParams().SetEmail(newUser.Email))
	require.NoError(t, err)
	require.True(t, cmp.Equal(userModel, userByEmail, dbTimeCmpOpt))

	userByName, err := repo.Get(ctx, models.NewGetUserParams().SetName(newUser.Name))
	require.NoError(t, err)
	require.True(t, cmp.Equal(userModel, userByName, dbTimeCmpOpt))
}
