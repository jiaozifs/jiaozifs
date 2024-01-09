package models_test

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/stretchr/testify/require"
)

func TestWipRepo(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewWipRepo(db)

	wipModel := &models.WorkingInProcess{}
	require.NoError(t, gofakeit.Struct(wipModel))
	newWipModel, err := repo.Insert(ctx, wipModel)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, newWipModel.ID)

	getWipParams := models.NewGetWipParams().
		SetID(newWipModel.ID).
		SetCreatorID(newWipModel.CreatorID).
		SetRepositoryID(newWipModel.RepositoryID).
		SetRefID(newWipModel.RefID)
	user, err := repo.Get(ctx, getWipParams)
	require.NoError(t, err)
	require.True(t, cmp.Equal(newWipModel, user, dbTimeCmpOpt))

	t.Run("list", func(t *testing.T) {
		secWipModel := &models.WorkingInProcess{}
		require.NoError(t, gofakeit.Struct(secWipModel))
		secWipModel.CreatorID = newWipModel.CreatorID
		secWipModel.RepositoryID = newWipModel.RepositoryID
		secNewWipModel, err := repo.Insert(ctx, secWipModel)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, secNewWipModel.ID)

		listParams := models.NewListWipParams().
			SetCreatorID(secNewWipModel.CreatorID).
			SetRepositoryID(secNewWipModel.RepositoryID)

		list, err := repo.List(ctx, listParams)
		require.NoError(t, err)
		require.Len(t, list, 2)

		deleteParams := models.NewDeleteWipParams().
			SetID(secWipModel.ID).
			SetCreatorID(secWipModel.CreatorID).
			SetRepositoryID(secWipModel.RepositoryID).
			SetRefID(secWipModel.RefID)
		affectedRow, err := repo.Delete(ctx, deleteParams)
		require.Equal(t, int64(1), affectedRow)
		require.NoError(t, err)

		affectedRow, err = repo.Delete(ctx, deleteParams)
		require.Equal(t, int64(0), affectedRow)
		require.NoError(t, err)

		_, err = repo.Get(ctx, models.NewGetWipParams().SetID(secWipModel.ID))
		require.ErrorIs(t, err, models.ErrNotFound)
	})
}

func TestWipRepoUpdateByID(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewWipRepo(db)

	t.Run("only update tree hash", func(t *testing.T) {
		wipModel := &models.WorkingInProcess{}
		require.NoError(t, gofakeit.Struct(wipModel))
		newWipModel, err := repo.Insert(ctx, wipModel)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, newWipModel.ID)

		updateModel := models.NewUpdateWipParams(newWipModel.ID).
			SetCurrentTree(hash.Hash("mock hash"))

		err = repo.UpdateByID(ctx, updateModel)
		require.NoError(t, err)
		updatedUser, err := repo.Get(ctx, models.NewGetWipParams().SetID(newWipModel.ID))
		require.NoError(t, err)
		require.Equal(t, newWipModel.State, updatedUser.State)
		require.Equal(t, newWipModel.BaseCommit, updatedUser.BaseCommit)
		require.Equal(t, "mock hash", string(updatedUser.CurrentTree))
	})

	t.Run("update both", func(t *testing.T) {
		wipModel := &models.WorkingInProcess{}
		require.NoError(t, gofakeit.Struct(wipModel))
		newWipModel, err := repo.Insert(ctx, wipModel)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, newWipModel.ID)

		updateModel := models.NewUpdateWipParams(newWipModel.ID).
			SetState(models.Completed).
			SetBaseCommit(hash.Hash("mock base hash")).
			SetCurrentTree(hash.Hash("mock hash"))

		err = repo.UpdateByID(ctx, updateModel)
		require.NoError(t, err)
		updatedUser, err := repo.Get(ctx, models.NewGetWipParams().SetID(newWipModel.ID))
		require.NoError(t, err)
		require.Equal(t, models.Completed, updatedUser.State)
		require.Equal(t, "mock base hash", string(updatedUser.BaseCommit))
		require.Equal(t, "mock hash", string(updatedUser.CurrentTree))
	})
}
