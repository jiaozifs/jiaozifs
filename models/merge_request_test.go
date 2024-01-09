package models_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiaozifs/jiaozifs/utils"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/stretchr/testify/require"
)

func TestMergeRequestRepoInsert(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	mrRepo := models.NewMergeRequestRepo(db)

	t.Run("insert and get", func(t *testing.T) {
		mrModel := &models.MergeRequest{}
		require.NoError(t, gofakeit.Struct(mrModel))
		mrModel.MergeState = models.MergeStateInit
		newMrModel, err := mrRepo.Insert(ctx, mrModel)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, newMrModel.ID)

		getMRParams := models.NewGetMergeRequestParams().
			SetID(newMrModel.ID).
			SetTargetBranch(newMrModel.TargetBranchID).
			SetSourceBranch(newMrModel.SourceBranchID).
			SetNumber(newMrModel.Sequence).SetState(models.MergeStateInit).SetTargetRepo(newMrModel.TargetRepoID)
		mrModel, err = mrRepo.Get(ctx, getMRParams)
		require.NoError(t, err)

		require.True(t, cmp.Equal(mrModel, newMrModel, dbTimeCmpOpt))
	})

	t.Run("delete", func(t *testing.T) {
		mrModel := &models.MergeRequest{}
		require.NoError(t, gofakeit.Struct(mrModel))
		newMrModel, err := mrRepo.Insert(ctx, mrModel)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, newMrModel.ID)

		deleteParams := models.NewDeleteMergeRequestParams().SetTargetRepo(newMrModel.TargetRepoID).SetNumber(newMrModel.Sequence)
		affectRows, err := mrRepo.Delete(ctx, deleteParams)
		require.NoError(t, err)
		require.Equal(t, int64(1), affectRows)
	})

	t.Run("list", func(t *testing.T) {
		startT := time.Now()
		targetID := uuid.New()
		for i := 0; i < 10; i++ {
			mrModel := &models.MergeRequest{}
			require.NoError(t, gofakeit.Struct(mrModel))
			mrModel.TargetRepoID = targetID
			mrModel.UpdatedAt = time.Now()
			mrModel.CreatedAt = time.Now()
			newMrModel, err := mrRepo.Insert(ctx, mrModel)
			require.NoError(t, err)
			require.NotEqual(t, uuid.Nil, newMrModel.ID)
		}

		t.Run("first page", func(t *testing.T) {
			mrs, hasMore, err := mrRepo.List(ctx, models.NewListMergeRequestParams().SetTargetRepoID(targetID).SetAmount(5))
			require.NoError(t, err)
			require.True(t, hasMore)
			require.Len(t, mrs, 5)
		})
		t.Run("last page", func(t *testing.T) {
			mrs, hasMore, err := mrRepo.List(ctx, models.NewListMergeRequestParams().SetTargetRepoID(targetID).SetAmount(5).SetAfter(startT))
			require.NoError(t, err)
			require.False(t, hasMore)
			require.Len(t, mrs, 0)
		})
	})

	t.Run("updatebyid", func(t *testing.T) {
		mrModel := &models.MergeRequest{}
		require.NoError(t, gofakeit.Struct(mrModel))
		newMrModel, err := mrRepo.Insert(ctx, mrModel)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, newMrModel.ID)

		newMrModel.Title = "Merge: xxxxx"
		newMrModel.Description = utils.String("vvvv")

		updateMrParams := models.NewUpdateMergeRequestParams(newMrModel.TargetRepoID, newMrModel.Sequence).
			SetTitle("Merge: xxxx").
			SetDescription("test update").
			SetState(models.MergeStateClosed)

		err = mrRepo.UpdateByID(ctx, updateMrParams)
		require.NoError(t, err)

		getMRParams := models.NewGetMergeRequestParams().
			SetID(newMrModel.ID)
		mrModel, err = mrRepo.Get(ctx, getMRParams)
		require.NoError(t, err)

		require.Equal(t, "Merge: xxxx", mrModel.Title)
		require.Equal(t, "test update", *mrModel.Description)

		require.Equal(t, models.MergeStateClosed, mrModel.MergeState)
	})
}
