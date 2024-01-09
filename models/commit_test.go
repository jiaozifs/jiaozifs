package models_test

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/stretchr/testify/require"
)

func TestCommitRepo(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repoID := uuid.New()
	commitRepo := models.NewCommitRepo(db, repoID)
	require.Equal(t, commitRepo.RepositoryID(), repoID)

	commitModel := &models.Commit{}
	require.NoError(t, gofakeit.Struct(commitModel))
	commitModel.RepositoryID = repoID
	newCommitModel, err := commitRepo.Insert(ctx, commitModel)
	require.NoError(t, err)
	commitModel, err = commitRepo.Commit(ctx, commitModel.Hash)
	require.NoError(t, err)

	require.True(t, cmp.Equal(commitModel, newCommitModel, dbTimeCmpOpt))

	t.Run("mis match repo id", func(t *testing.T) {
		mistMatchModel := &models.Commit{}
		require.NoError(t, gofakeit.Struct(mistMatchModel))
		_, err := commitRepo.Insert(ctx, mistMatchModel)
		require.ErrorIs(t, err, models.ErrRepoIDMisMatch)
	})
}

func TestDeleteCommit(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()
	t.Run("delete commit", func(t *testing.T) {
		repoID := uuid.New()
		commitRepo := models.NewCommitRepo(db, repoID)
		require.Equal(t, commitRepo.RepositoryID(), repoID)
		toDeleteModel := &models.Commit{}
		require.NoError(t, gofakeit.Struct(toDeleteModel))
		toDeleteModel.RepositoryID = repoID
		toDeleteModel, err := commitRepo.Insert(ctx, toDeleteModel)
		require.NoError(t, err)

		affectRows, err := commitRepo.Delete(ctx, models.NewDeleteParams().SetHash(toDeleteModel.Hash))
		require.NoError(t, err)
		require.Equal(t, int64(1), affectRows)
	})

	t.Run("delete batch", func(t *testing.T) {
		repoID := uuid.New()
		commitRepo := models.NewCommitRepo(db, repoID)
		require.Equal(t, commitRepo.RepositoryID(), repoID)
		for i := 0; i < 5; i++ {
			toDeleteModel := &models.Commit{}
			require.NoError(t, gofakeit.Struct(toDeleteModel))
			toDeleteModel.RepositoryID = repoID
			_, err := commitRepo.Insert(ctx, toDeleteModel)
			require.NoError(t, err)
		}

		affectRows, err := commitRepo.Delete(ctx, models.NewDeleteParams())
		require.NoError(t, err)
		require.Equal(t, int64(5), affectRows)
	})
}
