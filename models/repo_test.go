package models_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/stretchr/testify/require"
)

func TestRepoTransaction(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	t.Run("simple", func(t *testing.T) {
		pgRepo := models.NewRepo(db)
		err := pgRepo.Transaction(ctx, func(repo models.IRepo) error {
			userModel := &models.User{}
			require.NoError(t, gofakeit.Struct(userModel))
			_, err := repo.UserRepo().Insert(ctx, userModel)
			require.NoError(t, err)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("transaction", func(t *testing.T) {
		pgRepo := models.NewRepo(db)
		repoID := uuid.New()
		err := pgRepo.Transaction(ctx, func(repo models.IRepo) error {
			treeNode := &models.FileTree{}
			require.NoError(t, gofakeit.Struct(treeNode))
			treeNode.RepositoryID = repoID
			_, err := repo.FileTreeRepo(repoID).Insert(ctx, treeNode)
			require.NoError(t, err)
			return err
		})
		require.NoError(t, err)
	})

	t.Run("transaction  rollback", func(t *testing.T) {
		pgRepo := models.NewRepo(db)
		var id uuid.UUID
		err := pgRepo.Transaction(ctx, func(repo models.IRepo) error {
			repositoryModel := &models.Repository{}
			require.NoError(t, gofakeit.Struct(repositoryModel))
			insertedModel, err := repo.RepositoryRepo().Insert(ctx, repositoryModel)
			require.NoError(t, err)
			id = insertedModel.ID
			return fmt.Errorf("rollback")
		})
		require.Error(t, err)

		_, err = pgRepo.RepositoryRepo().Get(ctx, models.NewGetRepoParams().SetID(id))
		require.True(t, errors.Is(err, sql.ErrNoRows))
	})
}
