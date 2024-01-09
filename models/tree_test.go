package models_test

import (
	"context"
	"testing"

	"github.com/jiaozifs/jiaozifs/utils/hash"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/models/filemode"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/stretchr/testify/require"
)

func Test_sortSubObjects(t *testing.T) {
	entries := []models.TreeEntry{
		{
			Name: "c.txt",
			Hash: nil,
		},
		{
			Name: "a.txt",
			Hash: nil,
		},
		{
			Name: "b.txt",
			Hash: nil,
		},
	}

	models.SortSubObjects(entries)
	require.Equal(t, "a.txt", entries[0].Name)
	require.Equal(t, "b.txt", entries[1].Name)
	require.Equal(t, "c.txt", entries[2].Name)
}

func TestObjectRepo_Insert(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repoID := uuid.New()
	repo := models.NewFileTree(db, repoID)
	require.Equal(t, repo.RepositoryID(), repoID)

	objModel := &models.FileTree{}
	require.NoError(t, gofakeit.Struct(objModel))
	objModel.RepositoryID = repoID
	objModel.Properties.Mode = filemode.Regular
	newObj, err := repo.Insert(ctx, objModel)
	require.NoError(t, err)
	require.NotEqual(t, nil, newObj.Hash)

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(list))

	ref, err := repo.Get(ctx, models.NewGetObjParams().SetHash(newObj.Hash))
	require.NoError(t, err)

	require.True(t, cmp.Equal(newObj, ref, dbTimeCmpOpt))
	t.Run("mis match repo id", func(t *testing.T) {
		mistMatchModel := &models.FileTree{}
		require.NoError(t, gofakeit.Struct(mistMatchModel))
		mistMatchModel.Properties.Mode = filemode.Regular
		_, err := repo.Insert(ctx, mistMatchModel)
		require.ErrorIs(t, err, models.ErrRepoIDMisMatch)
	})
}

func TestNewTreeNode(t *testing.T) {
	id, err := uuid.Parse("a91ef678-1980-4b26-9bb9-eadc9f366429")
	require.NoError(t, err)

	t.Run("no subobjects", func(t *testing.T) {
		node, err := models.NewTreeNode(models.Property{Mode: filemode.Dir}, id)
		require.NoError(t, err)
		require.NotNil(t, node.SubObjects)
		require.Equal(t, "03c2737fb833f979f2bb5398248e8e64", node.Hash.Hex())
	})

	t.Run("no subobjects", func(t *testing.T) {
		node, err := models.NewTreeNode(models.Property{Mode: filemode.Dir}, id, models.TreeEntry{
			Name: "a.txt",
			Hash: hash.Hash("aaa"),
		})
		require.NoError(t, err)
		require.NotNil(t, node.SubObjects)
		require.Equal(t, "27d9fbf6d43195f34404a94c0de707a2", node.Hash.Hex())
	})
}

func TestFileTreeRepo_Delete(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repoID := uuid.New()
	repo := models.NewFileTree(db, repoID)
	require.Equal(t, repo.RepositoryID(), repoID)

	var treeModels []*models.FileTree
	for i := 0; i < 5; i++ {
		objModel := &models.FileTree{}
		require.NoError(t, gofakeit.Struct(objModel))
		objModel.RepositoryID = repoID
		newModel, err := repo.Insert(ctx, objModel)
		require.NoError(t, err)
		treeModels = append(treeModels, newModel)
	}

	//delete one
	affectRows, err := repo.Delete(ctx, models.NewDeleteTreeParams().SetHash(treeModels[0].Hash))
	require.NoError(t, err)
	require.Equal(t, int64(1), affectRows)

	//delete batch
	affectRows, err = repo.Delete(ctx, models.NewDeleteTreeParams())
	require.NoError(t, err)
	require.Equal(t, int64(4), affectRows)
}
