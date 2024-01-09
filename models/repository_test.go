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

func TestRepositoryUpdate(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewRepositoryRepo(db)

	t.Run("only update desc", func(t *testing.T) {
		repoModel := &models.Repository{}
		require.NoError(t, gofakeit.Struct(repoModel))
		newRepo, err := repo.Insert(ctx, repoModel)
		require.NoError(t, err)
		err = repo.UpdateByID(ctx, models.NewUpdateRepoParams(newRepo.ID).SetDescription("description"))
		require.NoError(t, err)
		user, err := repo.Get(ctx, models.NewGetRepoParams().SetID(newRepo.ID))
		require.NoError(t, err)
		require.Equal(t, "description", *user.Description)
		require.Equal(t, newRepo.HEAD, user.HEAD)
	})

	t.Run("update all fields", func(t *testing.T) {
		repoModel := &models.Repository{}
		require.NoError(t, gofakeit.Struct(repoModel))
		newRepo, err := repo.Insert(ctx, repoModel)
		require.NoError(t, err)
		err = repo.UpdateByID(ctx, models.NewUpdateRepoParams(newRepo.ID).SetDescription("description").SetHead("ggg"))
		require.NoError(t, err)
		user, err := repo.Get(ctx, models.NewGetRepoParams().SetID(newRepo.ID))
		require.NoError(t, err)
		require.Equal(t, "description", *user.Description)
		require.Equal(t, "ggg", user.HEAD)
	})
}

func TestRepositoryRepoInsert(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewRepositoryRepo(db)

	repoModel := &models.Repository{}
	require.NoError(t, gofakeit.Struct(repoModel))
	repoModel.Name = "aaabbbb"
	newRepo, err := repo.Insert(ctx, repoModel)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, newRepo.ID)

	user, err := repo.Get(ctx, models.NewGetRepoParams().SetID(newRepo.ID))
	require.NoError(t, err)
	require.True(t, cmp.Equal(repoModel, user, dbTimeCmpOpt))

	//insert secondary
	secModel := &models.Repository{}
	require.NoError(t, gofakeit.Struct(secModel))
	secModel.CreatorID = repoModel.CreatorID
	secModel.Name = "adabbeb"
	secRepo, err := repo.Insert(ctx, secModel)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, secRepo.ID)

	//list
	repos, _, err := repo.List(ctx, models.NewListRepoParams())
	require.NoError(t, err)
	require.Len(t, repos, 2)

	{
		//exact adabbeb
		repos, _, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetName("adabbeb", models.PrefixMatch))
		require.NoError(t, err)
		require.Len(t, repos, 1)
	}
	{
		//prefix a
		repos, _, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetName("a", models.PrefixMatch))
		require.NoError(t, err)
		require.Len(t, repos, 2)
	}

	{
		//subfix b
		repos, _, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetName("b", models.SuffixMatch))
		require.NoError(t, err)
		require.Len(t, repos, 2)
	}
	{
		//like ab
		repos, _, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetName("ab", models.LikeMatch))
		require.NoError(t, err)
		require.Len(t, repos, 2)
	}
	{
		//like ab
		repos, _, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetName("adabbeb", models.LikeMatch))
		require.NoError(t, err)
		require.Len(t, repos, 1)
	}
	{
		//amount 1
		repos, hasMore, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetAmount(1))
		require.NoError(t, err)
		require.True(t, hasMore)
		require.Len(t, repos, 1)
	}
	{
		//amount 2
		repos, hasMore, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetAmount(2))
		require.NoError(t, err)
		require.True(t, hasMore)
		require.Len(t, repos, 2)
	}
	{
		//amount 3
		repos, hasMore, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetAmount(3))
		require.NoError(t, err)
		require.False(t, hasMore)
		require.Len(t, repos, 2)
	}
	{
		//after
		repos, hasMore, err := repo.List(ctx, models.NewListRepoParams().SetCreatorID(secModel.CreatorID).SetAfter(time.Now()).SetAmount(1))
		require.NoError(t, err)
		require.True(t, hasMore)
		require.Len(t, repos, 1)
	}
	//delete
	deleteParams := models.NewDeleteRepoParams().
		SetID(secRepo.ID).
		SetOwnerID(secRepo.OwnerID).
		SetName(secRepo.Name)
	affectRows, err := repo.Delete(ctx, deleteParams)
	require.NoError(t, err)
	require.Equal(t, int64(1), affectRows)

	_, err = repo.Get(ctx, models.NewGetRepoParams().SetID(secRepo.ID))
	require.ErrorIs(t, err, models.ErrNotFound)
}
